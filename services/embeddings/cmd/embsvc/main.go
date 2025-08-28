package main
import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	_ "github.com/jackc/pgx/v5/stdlib"
	openai "github.com/sashabaranov/go-openai"
	"github.com/gorilla/mux"
)
var db *sql.DB
var openaiClient *openai.Client
func health(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) }
type UpsertReq struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}
func computeEmbeddingOpenAI(ctx context.Context, text string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}
	if openaiClient == nil {
		openaiClient = openai.NewClient(apiKey)
	}
	// using text-embedding-3-small (1536) or adjust model as needed
	resp, err := openaiClient.CreateEmbeddings(ctx, openai.EmbeddingsRequest{
		Model: openai.AdaEmbeddingV2, // fallback - adjust to "text-embedding-3-small" if available in client
		Input: []string{text},
	})
	if err != nil { return nil, err }
	if len(resp.Data) == 0 { return nil, fmt.Errorf("no embedding returned") }
	vec := resp.Data[0].Embedding
	// convert []float64 to []float32
	out := make([]float32, len(vec))
	for i := range vec { out[i] = float32(vec[i]) }
	return out, nil
}
// If you want to support a local embedding endpoint, POST {"text": "..."} -> {"embedding":[...]}
func computeEmbeddingLocal(ctx context.Context, text string) ([]float32, error) {
	local := os.Getenv("LOCAL_EMBEDDING_ENDPOINT")
	if local == "" { return nil, fmt.Errorf("LOCAL_EMBEDDING_ENDPOINT not set") }
	body := map[string]string{"text": text}
	b, _ := json.Marshal(body)
	resp, err := http.Post(local, "application/json", bytes.NewReader(b))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b2, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("local embedding error: %s", string(b2))
	}
	var out struct{ Embedding []float32 `json:"embedding"` }
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
	return out.Embedding, nil
}
func upsertHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req UpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad body", 400); return
	}
	if req.ID == "" || req.Content == "" {
		http.Error(w, "id & content required", 400); return
	}
	// choose embedding backend
	var emb []float32
	var err error
	if os.Getenv("LOCAL_EMBEDDING_ENDPOINT") != "" {
		emb, err = computeEmbeddingLocal(ctx, req.Content)
	} else {
		emb, err = computeEmbeddingOpenAI(ctx, req.Content)
	}
	if err != nil {
		log.Println("embedding error:", err)
		http.Error(w, "embedding error: "+err.Error(), 500)
		return
	}
	// convert to Postgres array literal
	// pgx supports binary vector, but we'll send as float array and use pgvector::vector
	// We'll use parameterized query with $1::vector
	// Prepare the query: INSERT ... ON CONFLICT DO UPDATE
	_, err = db.ExecContext(ctx, `
		INSERT INTO doc_vectors (id, content, embedding, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, embedding = EXCLUDED.embedding, updated_at = NOW()
	`, req.ID, req.Content, emb)
	if err != nil {
		log.Println("db upsert err:", err)
		http.Error(w, "db error", 500); return
	}
	json.NewEncoder(w).Encode(map[string]string{"status":"ok"})
}
type QueryReq struct {
	Query string `json:"query"`
	K int `json:"k"`
}
type NNRow struct {
	ID string `json:"id"`
	Content string `json:"content"`
	Distance float32 `json:"distance"`
}
func queryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req QueryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad body", 400); return
	}
	if req.Query == "" { http.Error(w, "query required", 400); return }
	if req.K == 0 { req.K = 5 }
	// compute embedding for query
	var emb []float32
	var err error
	if os.Getenv("LOCAL_EMBEDDING_ENDPOINT") != "" {
		emb, err = computeEmbeddingLocal(ctx, req.Query)
	} else {
		emb, err = computeEmbeddingOpenAI(ctx, req.Query)
	}
	if err != nil {
		http.Error(w, "embedding error: "+err.Error(), 500); return
	}
	// Query nearest neighbors using pgvector's <-> operator
	// We pass embedding as parameter; driver must support vector parameterization.
	rows, err := db.QueryContext(ctx, `
		SELECT id, content, embedding <-> $1 AS distance
		FROM doc_vectors
		ORDER BY embedding <-> $1
		LIMIT $2
	`, emb, req.K)
	if err != nil {
		log.Println("nn query err:", err)
		http.Error(w, "db error", 500); return
	}
	defer rows.Close()
	out := []NNRow{}
	for rows.Next() {
		var rrow NNRow
		if err := rows.Scan(&rrow.ID, &rrow.Content, &rrow.Distance); err != nil {
			log.Println("row scan err:", err); continue
		}
		out = append(out, rrow)
	}
	json.NewEncoder(w).Encode(out)
}
func main() {
	// Setup DB
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://atlas:atlaspass@postgres:5432/atlasdb?sslmode=disable"
	}
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil { log.Fatal(err) }
	if err := db.Ping(); err != nil { log.Fatal("db ping:", err) }
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/v1/upsert", upsertHandler).Methods("POST")
	r.HandleFunc("/v1/query", queryHandler).Methods("POST")
	port := os.Getenv("PORT"); if port=="" { port = "9500" }
	log.Println("embeddings service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}