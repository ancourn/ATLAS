package main
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"github.com/gorilla/mux"
)
func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
func whoami(w http.ResponseWriter, r *http.Request) {
	// Very simple stub - in real world validate session/jwt
	user := map[string]string{"id": "user_anon", "name": "Anonymous"}
	if cookie, err := r.Cookie("atlas_user"); err == nil {
		user["id"] = cookie.Value
		user["name"] = "SignedInUser"
		_ = cookie
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/whoami", whoami).Methods("GET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "9100"
	}
	log.Println("id service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}