package main
import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
)
var openaiKey = os.Getenv("OPENAI_API_KEY")
func health(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) }
func transcribeHandler(w http.ResponseWriter, r *http.Request) {
	// Accept audio file as multipart/form-data 'file'
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400); return
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, f)
	// If LOCAL_TRANSCRIBE_ENDPOINT is set, proxy to it
	local := os.Getenv("LOCAL_TRANSCRIBE_ENDPOINT")
	if local != "" {
		resp, err := http.Post(local, "audio/wav", bytes.NewReader(buf.Bytes()))
		if err != nil { http.Error(w, "local transcribe err", 500); return }
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
		return
	}
	// Else use OpenAI Whisper via client lib
	if openaiKey == "" {
		http.Error(w, "OPENAI_API_KEY not set", 500); return
	}
	client := openai.NewClient(openaiKey)
	resp, err := client.CreateTranscription(
		r.Context(),
		openai.AudioRequest{
			File: bytes.NewReader(buf.Bytes()),
			Model: "whisper-1",
		},
	)
	if err != nil {
		log.Println("whisper err:", err)
		http.Error(w, "transcribe error", 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"text": resp.Text})
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/v1/transcribe", transcribeHandler).Methods("POST")
	port := os.Getenv("PORT")
	if port == "" { port = "9501" }
	log.Println("transcribe listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}