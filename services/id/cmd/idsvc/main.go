package main
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	User      User   `json:"user"`
	ExpiresAt int64  `json:"expires_at"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

var (
	dbPool    *pgxpool.Pool
	jwtSecret []byte
)

func init() {
	// Initialize JWT secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not provided
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			log.Fatal("Failed to generate JWT secret")
		}
		secret = hex.EncodeToString(secretBytes)
	}
	jwtSecret = []byte(secret)
}

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
			password_hash VARCHAR(255) NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create sessions table
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)
	`)
	if err != nil {
		return err
	}

	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)
	`)
	if err != nil {
		return err
	}

	_, err = dbPool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)
	`)
	if err != nil {
		return err
	}

	return nil
}

func hashPassword(password string) (string, error) {
	// In a real implementation, use bcrypt or argon2
	// For demo purposes, we'll use a simple hash
	return fmt.Sprintf("%x", password), nil
}

func checkPassword(password, hash string) bool {
	// In a real implementation, use bcrypt or argon2
	// For demo purposes, we'll use simple comparison
	return fmt.Sprintf("%x", password) == hash
}

func generateJWT(user User) (string, int64, error) {
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expiresAt, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "atlas-id",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", 0, err
	}
	
	return tokenString, expiresAt, nil
}

func validateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}
		
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "bearer token required", http.StatusUnauthorized)
			return
		}
		
		claims, err := validateJWT(tokenString)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		
		// Add user info to request context
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "userEmail", claims.Email)
		ctx = context.WithValue(ctx, "userName", claims.Name)
		
		next(w, r.WithContext(ctx))
	}
}

func getUserFromContext(r *http.Request) (string, string, string) {
	userID, _ := r.Context().Value("userID").(string)
	userEmail, _ := r.Context().Value("userEmail").(string)
	userName, _ := r.Context().Value("userName").(string)
	return userID, userEmail, userName
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

func register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "name, email, and password are required", http.StatusBadRequest)
		return
	}
	
	ctx := context.Background()
	
	// Check if user already exists
	var exists bool
	err := dbPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if exists {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}
	
	// Hash password
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Create user
	user := User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		Password:  passwordHash,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	_, err = dbPool.Exec(ctx, 
		"INSERT INTO users (id, name, email, password_hash, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		user.ID, user.Name, user.Email, user.Password, user.IsActive, user.CreatedAt, user.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Generate JWT
	token, expiresAt, err := generateJWT(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Don't return password in response
	user.Password = ""
	
	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	ctx := context.Background()
	
	// Get user by email
	var user User
	err := dbPool.QueryRow(ctx, 
		"SELECT id, name, email, password_hash, is_active, created_at, updated_at FROM users WHERE email = $1 AND is_active = true", 
		req.Email).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Check password
	if !checkPassword(req.Password, user.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Generate JWT
	token, expiresAt, err := generateJWT(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Don't return password in response
	user.Password = ""
	
	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func whoami(w http.ResponseWriter, r *http.Request) {
	userID, userEmail, userName := getUserFromContext(r)
	
	if userID == "" {
		// Check for cookie-based auth (fallback for demo)
		if cookie, err := r.Cookie("atlas_user"); err == nil {
			user := map[string]string{
				"id":   cookie.Value,
				"name": "Demo User",
				"email": "demo@atlas.local",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
		
		user := map[string]string{"id": "user_anon", "name": "Anonymous", "email": ""}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return
	}
	
	user := map[string]string{
		"id":    userID,
		"name":  userName,
		"email": userEmail,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	userID, _, _ := getUserFromContext(r)
	
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	
	ctx := context.Background()
	var user User
	err := dbPool.QueryRow(ctx, 
		"SELECT id, name, email, is_active, created_at, updated_at FROM users WHERE id = $1", 
		userID).Scan(&user.ID, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	userID, _, _ := getUserFromContext(r)
	
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	
	var updateData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	ctx := context.Background()
	
	// Update user
	_, err := dbPool.Exec(ctx, 
		"UPDATE users SET name = $1, email = $2, updated_at = $3 WHERE id = $4",
		updateData.Name, updateData.Email, time.Now(), userID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Get updated user
	var user User
	err = dbPool.QueryRow(ctx, 
		"SELECT id, name, email, is_active, created_at, updated_at FROM users WHERE id = $1", 
		userID).Scan(&user.ID, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func logout(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you might want to blacklist the token
	// For now, we'll just clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "atlas_user",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
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
		passwordHash, err := hashPassword("demo123")
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			return
		}
		
		defaultUser := User{
			ID:        "user_demo",
			Name:      "Demo User",
			Email:     "demo@atlas.local",
			Password:  passwordHash,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		_, err = dbPool.Exec(ctx, 
			"INSERT INTO users (id, name, email, password_hash, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			defaultUser.ID, defaultUser.Name, defaultUser.Email, defaultUser.Password, defaultUser.IsActive, defaultUser.CreatedAt, defaultUser.UpdatedAt)
		
		if err != nil {
			log.Printf("Error creating default user: %v", err)
			return
		}
		
		log.Println("Created default user (demo@atlas.local / demo123)")
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
	
	// Public endpoints
	r.HandleFunc("/health", health).Methods("GET")
	r.HandleFunc("/api/register", register).Methods("POST")
	r.HandleFunc("/api/login", login).Methods("POST")
	r.HandleFunc("/api/whoami", whoami).Methods("GET")
	
	// Protected endpoints
	r.HandleFunc("/api/user", authMiddleware(getUser)).Methods("GET")
	r.HandleFunc("/api/user", authMiddleware(updateUser)).Methods("PUT")
	r.HandleFunc("/api/logout", authMiddleware(logout)).Methods("POST")
	
	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "9100"
	}
	
	log.Println("Enhanced ID service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}