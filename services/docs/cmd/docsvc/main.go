package main
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Document struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

type DocumentUpdate struct {
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`
	ClientID   string `json:"client_id"`
	UserID     string `json:"user_id"`
}

type CRDTUpdate struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"document_id"`
	Operation   string    `json:"operation"` // "insert", "delete", "retain"
	Position    int       `json:"position"`
	Content     string    `json:"content"`
	Length      int       `json:"length"`
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

var (
	dbPool      *pgxpool.Pool
	clients     = make(map[*websocket.Conn]string)
	docCache    = make(map[string]Document)
	mu          sync.RWMutex
)

func initDatabase() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://atlas:atlaspass@localhost:5432/atlasdb?sslmode=disable"
	}

	var err error
	dbPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Create tables if they don't exist
	err = createTables()
	if err != nil {
		return fmt.Errorf("unable to create tables: %w", err)
	}

	// Load documents into cache
	err = loadDocumentsCache()
	if err != nil {
		return fmt.Errorf("unable to load documents cache: %w", err)
	}

	return nil
}

func createTables() error {
	ctx := context.Background()
	
	// Create users table
	_, err := dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create documents table
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS documents (
			id VARCHAR(36) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			user_id VARCHAR(36) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			version INTEGER DEFAULT 1,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create CRDT updates table for conflict resolution
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS crdt_updates (
			id VARCHAR(36) PRIMARY KEY,
			document_id VARCHAR(36) NOT NULL,
			operation VARCHAR(50) NOT NULL,
			position INTEGER NOT NULL,
			content TEXT,
			length INTEGER,
			user_id VARCHAR(36) NOT NULL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (document_id) REFERENCES documents(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_documents_user_id ON documents(user_id)
	`)
	if err != nil {
		return err
	}

	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_crdt_updates_document_id ON crdt_updates(document_id)
	`)
	if err != nil {
		return err
	}

	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_crdt_updates_timestamp ON crdt_updates(timestamp)
	`)
	if err != nil {
		return err
	}

	return nil
}

func loadDocumentsCache() error {
	ctx := context.Background()
	rows, err := dbPool.Query(ctx, "SELECT id, title, content, user_id, created_at, updated_at, version FROM documents")
	if err != nil {
		return err
	}
	defer rows.Close()

	mu.Lock()
	defer mu.Unlock()

	docCache = make(map[string]Document)
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.UserID, &doc.CreatedAt, &doc.UpdatedAt, &doc.Version)
		if err != nil {
			return err
		}
		docCache[doc.ID] = doc
	}

	return nil
}

func health(w http.ResponseWriter, r *http.Request) {
	if dbPool == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("database not connected"))
		return
	}
	
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func getDocuments(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	
	ctx := context.Background()
	var rows pgx.Rows
	var err error
	
	if userID != "" {
		rows, err = dbPool.Query(ctx, 
			"SELECT id, title, content, user_id, created_at, updated_at, version FROM documents WHERE user_id = $1 ORDER BY updated_at DESC", 
			userID)
	} else {
		rows, err = dbPool.Query(ctx, 
			"SELECT id, title, content, user_id, created_at, updated_at, version FROM documents ORDER BY updated_at DESC")
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.UserID, &doc.CreatedAt, &doc.UpdatedAt, &doc.Version)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		documents = append(documents, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

func getDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	
	ctx := context.Background()
	var doc Document
	err := dbPool.QueryRow(ctx, 
		"SELECT id, title, content, user_id, created_at, updated_at, version FROM documents WHERE id = $1", 
		docID).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.UserID, &doc.CreatedAt, &doc.UpdatedAt, &doc.Version)
	
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func createDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate user exists
	if !validateUser(doc.UserID) {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}
	
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	doc.Version = 1
	
	ctx := context.Background()
	_, err := dbPool.Exec(ctx, 
		"INSERT INTO documents (id, title, content, user_id, created_at, updated_at, version) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		doc.ID, doc.Title, doc.Content, doc.UserID, doc.CreatedAt, doc.UpdatedAt, doc.Version)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update cache
	mu.Lock()
	docCache[doc.ID] = doc
	mu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func updateDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	
	var update DocumentUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate user exists
	if !validateUser(update.UserID) {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}
	
	ctx := context.Background()
	
	// Get current document
	var currentDoc Document
	err := dbPool.QueryRow(ctx, 
		"SELECT id, title, content, user_id, created_at, updated_at, version FROM documents WHERE id = $1", 
		docID).Scan(&currentDoc.ID, &currentDoc.Title, &currentDoc.Content, &currentDoc.UserID, &currentDoc.CreatedAt, &currentDoc.UpdatedAt, &currentDoc.Version)
	
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// Check permissions (user can only edit their own documents)
	if currentDoc.UserID != update.UserID {
		http.Error(w, "permission denied", http.StatusForbidden)
		return
	}
	
	// Update document with version increment
	newVersion := currentDoc.Version + 1
	updatedAt := time.Now()
	
	_, err = dbPool.Exec(ctx, 
		"UPDATE documents SET content = $1, updated_at = $2, version = $3 WHERE id = $4",
		update.Content, updatedAt, newVersion, docID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Store CRDT update for conflict resolution
	crdtUpdate := CRDTUpdate{
		ID:         uuid.New().String(),
		DocumentID: docID,
		Operation:  "update",
		Content:    update.Content,
		UserID:     update.UserID,
		Timestamp:  updatedAt,
	}
	
	_, err = dbPool.Exec(ctx, 
		"INSERT INTO crdt_updates (id, document_id, operation, content, user_id, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
		crdtUpdate.ID, crdtUpdate.DocumentID, crdtUpdate.Operation, crdtUpdate.Content, crdtUpdate.UserID, crdtUpdate.Timestamp)
	
	if err != nil {
		log.Printf("Warning: failed to store CRDT update: %v", err)
	}
	
	// Update cache
	mu.Lock()
	updatedDoc := currentDoc
	updatedDoc.Content = update.Content
	updatedDoc.UpdatedAt = updatedAt
	updatedDoc.Version = newVersion
	docCache[docID] = updatedDoc
	mu.Unlock()
	
	// Broadcast update to all connected clients
	broadcastUpdate(docID, update.Content, update.ClientID, update.UserID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedDoc)
}

func validateUser(userID string) bool {
	ctx := context.Background()
	var exists bool
	err := dbPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND is_active = true)", userID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()
	
	clientID := uuid.New().String()
	mu.Lock()
	clients[conn] = clientID
	mu.Unlock()
	
	log.Println("Client connected:", clientID)
	
	// Send initial documents list
	mu.RLock()
	docList := make([]Document, 0, len(docCache))
	for _, doc := range docCache {
		docList = append(docList, doc)
	}
	mu.RUnlock()
	
	initialMsg := map[string]interface{}{
		"type":      "initial",
		"documents": docList,
		"client_id": clientID,
	}
	
	if err := conn.WriteJSON(initialMsg); err != nil {
		log.Println("Error sending initial data:", err)
		return
	}
	
	// Handle incoming messages
	for {
		var msg DocumentUpdate
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		
		msg.ClientID = clientID
		
		// Update document
		mu.Lock()
		if doc, exists := docCache[msg.DocumentID]; exists {
			doc.Content = msg.Content
			doc.UpdatedAt = time.Now()
			doc.Version++
			docCache[msg.DocumentID] = doc
		}
		mu.Unlock()
		
		// Broadcast to other clients
		broadcastUpdate(msg.DocumentID, msg.Content, clientID, msg.UserID)
	}
	
	// Clean up disconnected client
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()
	
	log.Println("Client disconnected:", clientID)
}

func broadcastUpdate(docID, content, senderID, userID string) {
	mu.RLock()
	defer mu.RUnlock()
	
	update := map[string]interface{}{
		"type":        "update",
		"document_id": docID,
		"content":     content,
		"sender_id":   senderID,
		"user_id":     userID,
		"timestamp":   time.Now().Unix(),
	}
	
	for conn, clientID := range clients {
		if clientID != senderID { // Don't send back to sender
			if err := conn.WriteJSON(update); err != nil {
				log.Println("Broadcast error:", err)
				conn.Close()
				delete(clients, conn)
			}
		}
	}
}

func main() {
	// Initialize database
	err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()
	
	// Create default user if none exists
	createDefaultUser()
	
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/documents", getDocuments).Methods("GET")
	r.HandleFunc("/api/documents", createDocument).Methods("POST")
	r.HandleFunc("/api/documents/{id}", getDocument).Methods("GET")
	r.HandleFunc("/api/documents/{id}", updateDocument).Methods("PUT")
	r.HandleFunc("/ws", handleWebSocket)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "9300"
	}
	
	log.Println("Enhanced docs service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func createDefaultUser() {
	ctx := context.Background()
	var count int
	err := dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		log.Printf("Error checking user count: %v", err)
		return
	}
	
	if count == 0 {
		defaultUser := User{
			ID:       "user_demo",
			Name:     "Demo User",
			Email:    "demo@atlas.local",
			IsActive: true,
		}
		
		_, err = dbPool.Exec(ctx, 
			"INSERT INTO users (id, name, email, is_active) VALUES ($1, $2, $3, $4)",
			defaultUser.ID, defaultUser.Name, defaultUser.Email, defaultUser.IsActive)
		
		if err != nil {
			log.Printf("Error creating default user: %v", err)
			return
		}
		
		// Create sample document for default user
		sampleDoc := Document{
			ID:        uuid.New().String(),
			Title:     "Welcome to ATLAS Docs",
			Content:   "Welcome to the collaborative editor!\n\nStart typing here to see real-time collaboration in action.\n\nFeatures:\n- Real-time synchronization\n- User permissions\n- Persistent storage\n- Version control",
			UserID:    defaultUser.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   1,
		}
		
		_, err = dbPool.Exec(ctx, 
			"INSERT INTO documents (id, title, content, user_id, created_at, updated_at, version) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			sampleDoc.ID, sampleDoc.Title, sampleDoc.Content, sampleDoc.UserID, sampleDoc.CreatedAt, sampleDoc.UpdatedAt, sampleDoc.Version)
		
		if err != nil {
			log.Printf("Error creating sample document: %v", err)
			return
		}
		
		// Update cache
		mu.Lock()
		docCache[sampleDoc.ID] = sampleDoc
		mu.Unlock()
		
		log.Println("Created default user and sample document")
	}
}