package main
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/mux"
	nats "github.com/nats-io/nats.go"
)
type Message struct {
	ID      string    `json:"id"`
	Source  string    `json:"source"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	Time    time.Time `json:"time"`
}
func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200); w.Write([]byte("ok"))
}
func getInbox(w http.ResponseWriter, r *http.Request) {
	// return dummy messages for now
	msgs := []Message{
		{ID: "m1", Source: "email", From: "alice@example.com", Subject: "Hello", Body: "Test message", Time: time.Now()},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/inbox", getInbox).Methods("GET")
	// connect to NATS for future eventing
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Println("warning: cannot connect to nats:", err)
	} else {
		log.Println("connected to nats")
		// example subscribe
		sub, _ := nc.SubscribeSync("inbox.events")
		_ = sub
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "9200"
	}
	log.Println("inbox service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}