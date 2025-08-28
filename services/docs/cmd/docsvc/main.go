package main
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DocumentUpdate struct {
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`
	ClientID   string `json:"client_id"`
}

var (
	documents = make(map[string]Document)
	clients   = make(map[*websocket.Conn]string)
	mu        sync.RWMutex
)

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func getDocuments(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	
	docList := make([]Document, 0, len(documents))
	for _, doc := range documents {
		docList = append(docList, doc)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docList)
}

func getDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	
	mu.RLock()
	doc, exists := documents[docID]
	mu.RUnlock()
	
	if !exists {
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
	
	doc.ID = generateID()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	
	mu.Lock()
	documents[doc.ID] = doc
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
	
	mu.Lock()
	defer mu.Unlock()
	
	doc, exists := documents[docID]
	if !exists {
		http.NotFound(w, r)
		return
	}
	
	doc.Content = update.Content
	doc.UpdatedAt = time.Now()
	documents[docID] = doc
	
	// Broadcast update to all connected clients
	broadcastUpdate(docID, update.Content, update.ClientID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()
	
	clientID := generateID()
	mu.Lock()
	clients[conn] = clientID
	mu.Unlock()
	
	log.Println("Client connected:", clientID)
	
	// Send initial documents list
	mu.RLock()
	docList := make([]Document, 0, len(documents))
	for _, doc := range documents {
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
		if doc, exists := documents[msg.DocumentID]; exists {
			doc.Content = msg.Content
			doc.UpdatedAt = time.Now()
			documents[msg.DocumentID] = doc
		}
		mu.Unlock()
		
		// Broadcast to other clients
		broadcastUpdate(msg.DocumentID, msg.Content, clientID)
	}
	
	// Clean up disconnected client
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()
	
	log.Println("Client disconnected:", clientID)
}

func broadcastUpdate(docID, content, senderID string) {
	mu.RLock()
	defer mu.RUnlock()
	
	update := map[string]interface{}{
		"type":        "update",
		"document_id": docID,
		"content":     content,
		"sender_id":   senderID,
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

func generateID() string {
	return "doc_" + time.Now().Format("20060102150405") + "_" + 
		time.Now().Format("000000")
}

func main() {
	// Initialize with a sample document
	sampleDoc := Document{
		ID:        "doc_sample",
		Title:     "Welcome to ATLAS Docs",
		Content:   "Welcome to the collaborative editor!\n\nStart typing here to see real-time collaboration in action.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	documents[sampleDoc.ID] = sampleDoc
	
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
	
	log.Println("docs service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}