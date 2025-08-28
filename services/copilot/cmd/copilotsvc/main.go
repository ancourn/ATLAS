package main
import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	openai "github.com/sashabaranov/go-openai"
	"github.com/gorilla/mux"
)
var openaiKey = os.Getenv("OPENAI_API_KEY")
var embSvc = os.Getenv("EMB_SERVICE_URL") // e.g., http://embeddings:9500
func health(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) }
// helper: retrieve top-k docs from embeddings
func retrieveDocs(query string, k int) ([]string, error) {
	if embSvc == "" { embSvc = "http://embeddings:9500" }
	reqBody := map[string]interface{}{"query": query, "k": k}
	b, _ := json.Marshal(reqBody)
	resp, err := http.Post(embSvc+"/v1/query", "application/json", strings.NewReader(string(b)))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var rows []struct{ ID, Content string; Distance float32 }
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil { return nil, err }
	out := make([]string, 0, len(rows))
	for _, r := range rows { out = append(out, r.Content) }
	return out, nil
}
// SSE stream helper
func streamSSE(w http.ResponseWriter, req *http.Request, messages []string, prompt string) {
	flusher, ok := w.(http.Flusher)
	if !ok { http.Error(w, "streaming unsupported", 500); return }
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
    // If no OPENAI_API_KEY, we fallback to returning a mock suggestion
	if openaiKey == "" {
		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", "No OPENAI_API_KEY: copilot offline (mock suggestion).")
		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		flusher.Flush()
		return
	}
	client := openai.NewClient(openaiKey)
	// Compose system + context
	ctxBuilder := "You are ATLAS Copilot. Use the following context to propose a helpful suggestion.\n\n"
	for i, d := range messages { ctxBuilder += fmt.Sprintf("DOC[%d]: %s\n\n", i+1, d) }
	ctxBuilder += "Now produce a concise suggestion relevant to the USER_PROMPT below.\n\nUSER_PROMPT:\n" + prompt
	// Use ChatCompletions streaming
	stream, err := client.CreateChatCompletionStream(req.Context(), openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are a helpful assistant integrated into ATLAS; be concise and actionable."},
			{Role: "user", Content: ctxBuilder},
		},
		MaxTokens: 400,
		Temperature: 0.2,
	})
	if err != nil {
		log.Println("chat stream err", err)
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error()); flusher.Flush(); return
	}
	defer stream.Close()
	// Stream tokens back as SSE 'chunk' events
	for {
		resp, err := stream.Recv()
		if err == io.EOF { break }
		if err != nil {
			log.Println("stream recv err", err)
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return
		}
		for _, choice := range resp.Choices {
			if choice.Delta.Content != "" {
				escaped := strings.ReplaceAll(choice.Delta.Content, "\n", "\\n")
				fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", escaped)
				flusher.Flush()
			}
		}
	}
	// Done
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
type CopilotReq struct {
	Prompt string `json:"prompt"`
	K int `json:"k"`
}
func copilotHandler(w http.ResponseWriter, r *http.Request) {
	// Accept POST JSON {prompt, k}
	var cr CopilotReq
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		http.Error(w, "bad body", 400); return
	}
	if cr.K == 0 { cr.K = 4 }
	// Get context via embeddings
	ctxDocs, _ := retrieveDocs(cr.Prompt, cr.K)
	// stream suggestions using SSE
	streamSSE(w, r, ctxDocs, cr.Prompt)
}
func liveHandler(w http.ResponseWriter, r *http.Request) {
	// For continuous suggestions: accept query param ?q=... and stream
	q := r.URL.Query().Get("q")
	if q == "" { http.Error(w, "q required", 400); return }
	ctxDocs, _ := retrieveDocs(q, 5)
	streamSSE(w, r, ctxDocs, q)
}
func main() {
	if os.Getenv("OPENAI_API_KEY") != "" { openaiKey = os.Getenv("OPENAI_API_KEY") }
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/v1/call", copilotHandler).Methods("POST")
	r.HandleFunc("/v1/live", liveHandler).Methods("GET") // SSE GET ?q=...
	port := os.Getenv("PORT"); if port=="" { port = "9502" }
	log.Println("copilot listening on", port)
	srv := &http.Server{ Addr: ":"+port, Handler: r, ReadHeaderTimeout: 10*time.Second }
	log.Fatal(srv.ListenAndServe())
}