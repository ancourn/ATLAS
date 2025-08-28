package main
import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	openai "github.com/sashabaranov/go-openai"
	"github.com/gorilla/mux"
)
var openaiClient *openai.Client
func proxyOpenAI(prompt string) (string, error) {
	// If LOCAL_AI_ENDPOINT is set, proxy there.
	local := os.Getenv("LOCAL_AI_ENDPOINT")
	if local != "" {
		// simple forward
		resp, err := http.Post(local, "application/json", strings.NewReader(`{"prompt":`+jsonEscape(prompt)+`}`))
		if err != nil { return "", err }
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b), nil
	}
	// Use OpenAI compatible client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", nil
	}
	if openaiClient == nil {
		openaiClient = openai.NewClient(apiKey)
	}
	resp, err := openaiClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 400,
	})
	if err != nil { return "", err }
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", nil
}
func jsonEscape(s string) string {
	j, _ := json.Marshal(s)
	return string(j)
}
func summarizeHandler(w http.ResponseWriter, r *http.Request) {
	var body struct{ Text string `json:"text"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", 400); return
	}
	out, err := proxyOpenAI("Summarize clearly and concisely:\n\n" + body.Text)
	if err != nil {
		http.Error(w, "ai error: "+err.Error(), 500); return
	}
	json.NewEncoder(w).Encode(map[string]string{"summary": out})
}
func inboxSuggestHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", 400); return
	}
	out, err := proxyOpenAI("Write a polite reply suggestion to this message:\n\n" + body.Message)
	if err != nil {
		http.Error(w, "ai error", 500); return
	}
	json.NewEncoder(w).Encode(map[string]string{"reply": out})
}
func health(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) }
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/v1/summarize", summarizeHandler).Methods("POST")
	r.HandleFunc("/v1/inbox/suggest", inboxSuggestHandler).Methods("POST")
	port := os.Getenv("PORT"); if port=="" {port="9400"}
	log.Println("ai router listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}