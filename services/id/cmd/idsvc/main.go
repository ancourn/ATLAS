package main
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)
var jwtKey = []byte("atlas_dev_secret") // override with env JWT_SECRET
type Claims struct {
	Sub  string `json:"sub"`
	Name string `json:"name"`
	jwt.RegisteredClaims
}
func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Demo login: Accept name in JSON -> issue JWT cookie
	type req struct{ Name string `json:"name"` }
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	exp := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Sub:  "user_" + body.Name,
		Name: body.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		jwtKey = []byte(secret)
	}
	ss, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "token error", 500)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "atlas_jwt",
		Value:    ss,
		Path:     "/",
		HttpOnly: true,
		Expires:  exp,
	})
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
func whoami(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("atlas_jwt")
	if err != nil || cookie.Value == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "anon", "name": "Anonymous"})
		return
	}
	tokenStr := cookie.Value
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		jwtKey = []byte(secret)
	}
	claims := &Claims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		http.Error(w, "invalid token", 401)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"id": claims.Sub, "name": claims.Name})
}
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "atlas_jwt",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret != "" {
		jwtKey = []byte(secret)
	}
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/login", loginHandler).Methods("POST")
	r.HandleFunc("/api/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/api/whoami", whoami).Methods("GET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "9100"
	}
	log.Println("id service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}