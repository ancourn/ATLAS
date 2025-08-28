package main
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
// helper to call embeddings query service
func retrieveContext(query string, k int) ([]string, error) {
	embSvc := os.Getenv("EMB_SERVICE_URL")
	if embSvc == "" {
		embSvc = "http://embeddings:9500"
	}
	reqBody, _ := json.Marshal(map[string]interface{}{"query": query, "k": k})
	resp, err := http.Post(embSvc+"/v1/query", "application/json", strings.NewReader(string(reqBody)))
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("emb query err: %s", string(b))
	}
	var rows []struct{ ID, Content string; Distance float32 }
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil { return nil, err }
	out := []string{}
	for _, r := range rows { out = append(out, r.Content) }
	return out, nil
}
func summarizeHandler(w http.ResponseWriter, r *http.Request) {
	var body struct{ Text string `json:"text"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w,"bad body",400); return }
	ctxDocs, err := retrieveContext(body.Text, 5)
	if err == nil && len(ctxDocs) > 0 {
		// prepend context into prompt
		contextText := "CONTEXT:\n"
		for i, d := range ctxDocs { contextText += fmt.Sprintf("DOC %d: %s\n", i+1, d) }
		body.Text = contextText + "\n\nUSER TEXT:\n" + body.Text
	}
	out, err := proxyOpenAI(body.Text)
	if err != nil { http.Error(w,"ai error:"+err.Error(),500); return }
	json.NewEncoder(w).Encode(map[string]string{"summary": out})
}
func inboxSuggestHandler(w http.ResponseWriter, r *http.Request) {
	var body struct{ Message string `json:"message"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w,"bad body",400); return }
	ctxDocs, _ := retrieveContext(body.Message, 5)
	var prompt string
	if len(ctxDocs) > 0 {
		prompt = "Use context from company docs to draft a polite reply and list sources. CONTEXT:\n"
		for i,d := range ctxDocs { prompt += fmt.Sprintf("DOC %d: %s\n", i+1, d) }
		prompt += "\n\nMESSAGE:\n" + body.Message
	} else {
		prompt = "Reply politely to this message:\n\n" + body.Message
	}
	out, err := proxyOpenAI(prompt)
	if err != nil { http.Error(w,"ai error:"+err.Error(),500); return }
	json.NewEncoder(w).Encode(map[string]string{"reply": out})
}
func health(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) }
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/v1/summarize", summarizeHandler).Methods("POST")
	r.HandleFunc("/v1/inbox/suggest", inboxSuggestHandler).Methods("POST")
	port := os.Getenv("PORT"); if port=="" {port="9400" }
	log.Println("ai router listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}