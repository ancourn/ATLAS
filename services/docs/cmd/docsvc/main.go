package main
import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/gorilla/mux"
)
var db *sql.DB
func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
func saveDoc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var body struct{ Content string `json:"content"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", 400)
		return
	}
	ctx := r.Context()
	_, err := db.ExecContext(ctx, `
		INSERT INTO docs (id, content, updated_at) VALUES ($1,$2,NOW())
		ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, updated_at = NOW()
	`, id, body.Content)
	if err != nil {
		log.Println("save err", err)
		http.Error(w, "db error", 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status":"ok"})
}
func loadDoc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	row := db.QueryRow("SELECT content FROM docs WHERE id=$1", id)
	var content sql.NullString
	if err := row.Scan(&content); err != nil {
		if err == sql.ErrNoRows {
			json.NewEncoder(w).Encode(map[string]string{"content": ""})
			return
		}
		http.Error(w, "db error", 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"content": content.String})
}
func ensureTables() {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS docs (
	  id TEXT PRIMARY KEY,
	  content TEXT,
	  updated_at TIMESTAMP WITH TIME ZONE
	)
	`)
	if err != nil {
		log.Fatal("cannot create table", err)
	}
}
func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://atlas:atlaspass@postgres:5432/atlasdb?sslmode=disable"
	}
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}
	ensureTables()
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/docs/save/{id}", saveDoc).Methods("POST")
	r.HandleFunc("/api/docs/load/{id}", loadDoc).Methods("GET")
	port := os.Getenv("PORT")
	if port == "" { port = "9300" }
	log.Println("docs persistence listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}