package main
import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"github.com/gorilla/mux"
)

type AIService struct {
	Endpoint string
	APIKey   string
}

type InboxMessage struct {
	ID      string    `json:"id"`
	Source  string    `json:"source"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	Time    time.Time `json:"time"`
}

type Document struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
	UserID  string    `json:"user_id"`
}

type AISummaryRequest struct {
	Content   string `json:"content"`
	Type      string `json:"type"` // "inbox", "document", "search"
	Query     string `json:"query,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
}

type AISummaryResponse struct {
	Summary      string   `json:"summary"`
	Suggestions  []string `json:"suggestions,omitempty"`
	KeyPoints    []string `json:"key_points,omitempty"`
	Sentiment    string   `json:"sentiment,omitempty"`
	ProcessingTime int64  `json:"processing_time_ms"`
}

type AISearchRequest struct {
	Query      string   `json:"query"`
	Documents  []string `json:"documents"`
	MaxResults int      `json:"max_results"`
}

type AISearchResponse struct {
	Results      []SearchResult `json:"results"`
	ProcessingTime int64        `json:"processing_time_ms"`
}

type SearchResult struct {
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
	Relevance  float64 `json:"relevance"`
}

var aiService AIService

func init() {
	aiService.Endpoint = os.Getenv("AI_ENDPOINT")
	if aiService.Endpoint == "" {
		aiService.Endpoint = "http://localhost:8000" // Default local AI endpoint
	}
	aiService.APIKey = os.Getenv("AI_API_KEY")
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func summarizeInbox(w http.ResponseWriter, r *http.Request) {
	var messages []InboxMessage
	if err := json.NewDecoder(r.Body).Decode(&messages); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	
	// Combine message content for analysis
	var content strings.Builder
	for _, msg := range messages {
		content.WriteString(fmt.Sprintf("From: %s\nSubject: %s\n%s\n\n", msg.From, msg.Subject, msg.Body))
	}

	summary := generateSummary(content.String(), "inbox")
	response := AISummaryResponse{
		Summary:      summary.Summary,
		Suggestions:  generateReplySuggestions(messages),
		KeyPoints:    extractKeyPoints(content.String()),
		Sentiment:    analyzeSentiment(content.String()),
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func summarizeDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	summary := generateSummary(doc.Content, "document")
	
	response := AISummaryResponse{
		Summary:      summary.Summary,
		KeyPoints:    extractKeyPoints(doc.Content),
		Sentiment:    analyzeSentiment(doc.Content),
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchDocuments(w http.ResponseWriter, r *http.Request) {
	var searchReq AISearchRequest
	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	
	// Simulate AI-powered search (in real implementation, would use vector embeddings)
	results := performAISearch(searchReq.Query, searchReq.Documents, searchReq.MaxResults)
	
	response := AISearchResponse{
		Results:        results,
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateReplySuggestions(messages []InboxMessage) []string {
	// Generate context-aware reply suggestions based on message content
	suggestions := []string{}
	
	for _, msg := range messages {
		content := strings.ToLower(msg.Subject + " " + msg.Body)
		
		switch {
		case strings.Contains(content, "meeting") || strings.Contains(content, "schedule"):
			suggestions = append(suggestions, "I'd be happy to schedule a meeting. What time works best for you?")
		case strings.Contains(content, "question") || strings.Contains(content, "help"):
			suggestions = append(suggestions, "I'd be happy to help with that. Could you provide more details?")
		case strings.Contains(content, "thank") || strings.Contains(content, "appreciate"):
			suggestions = append(suggestions, "You're welcome! Let me know if you need anything else.")
		case strings.Contains(content, "urgent") || strings.Contains(content, "asap"):
			suggestions = append(suggestions, "I'll prioritize this and get back to you shortly.")
		default:
			suggestions = append(suggestions, "Thank you for your message. I'll review this and respond soon.")
		}
	}
	
	// Remove duplicates and limit to 3 suggestions
	uniqueSuggestions := make(map[string]bool)
	var finalSuggestions []string
	
	for _, suggestion := range suggestions {
		if !uniqueSuggestions[suggestion] && len(finalSuggestions) < 3 {
			uniqueSuggestions[suggestion] = true
			finalSuggestions = append(finalSuggestions, suggestion)
		}
	}
	
	return finalSuggestions
}

func generateSummary(content, contentType string) AISummaryResponse {
	// Simulate AI summarization (in real implementation, would call AI service)
	words := strings.Fields(content)
	if len(words) == 0 {
		return AISummaryResponse{Summary: "No content to summarize"}
	}

	// Simple extractive summarization for demo
	sentences := strings.Split(content, ".")
	if len(sentences) > 3 {
		// Take first sentence and a middle one for summary
		summary := sentences[0] + ". " + sentences[len(sentences)/2] + "."
		return AISummaryResponse{Summary: strings.TrimSpace(summary)}
	}
	
	return AISummaryResponse{Summary: strings.TrimSpace(content)}
}

func extractKeyPoints(content string) []string {
	// Simple key point extraction (in real implementation, would use NLP)
	keyPoints := []string{}
	
	// Look for common patterns
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.Contains(line, ":")) {
			keyPoints = append(keyPoints, strings.TrimLeft(line, "-* "))
		}
	}
	
	// If no bullet points found, extract sentences with important keywords
	if len(keyPoints) == 0 {
		sentences := strings.Split(content, ".")
		keywords := []string{"important", "key", "main", "critical", "essential", "significant"}
		
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(sentence), keyword) && len(sentence) > 20 {
					keyPoints = append(keyPoints, sentence)
					break
				}
			}
			if len(keyPoints) >= 3 {
				break
			}
		}
	}
	
	return keyPoints
}

func analyzeSentiment(content string) string {
	// Simple sentiment analysis (in real implementation, would use AI service)
	content = strings.ToLower(content)
	
	positiveWords := []string{"good", "great", "excellent", "happy", "pleased", "thank", "appreciate", "love", "wonderful"}
	negativeWords := []string{"bad", "terrible", "awful", "unhappy", "disappointed", "angry", "hate", "problem", "issue"}
	
	positiveCount := 0
	negativeCount := 0
	
	for _, word := range positiveWords {
		if strings.Contains(content, word) {
			positiveCount++
		}
	}
	
	for _, word := range negativeWords {
		if strings.Contains(content, word) {
			negativeCount++
		}
	}
	
	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	return "neutral"
}

func performAISearch(query string, documentIDs []string, maxResults int) []SearchResult {
	// Simulate AI-powered search (in real implementation, would use vector embeddings)
	results := []SearchResult{}
	
	// Mock search results based on query keywords
	query = strings.ToLower(query)
	
	for i, docID := range documentIDs {
		if i >= maxResults {
			break
		}
		
		// Simulate relevance scoring
		relevance := 0.5 + float64(i)*0.1 // Decreasing relevance
		
		// Generate mock snippet based on query
		snippet := fmt.Sprintf("Document %s contains information related to '%s'. This is a preview of the content...", docID, query)
		
		results = append(results, SearchResult{
			DocumentID: docID,
			Title:      fmt.Sprintf("Document %s", docID),
			Snippet:    snippet,
			Relevance:  relevance,
		})
	}
	
	return results
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/health", health)
	r.HandleFunc("/api/ai/inbox/summarize", summarizeInbox).Methods("POST")
	r.HandleFunc("/api/ai/document/summarize", summarizeDocument).Methods("POST")
	r.HandleFunc("/api/ai/search", searchDocuments).Methods("POST")
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "9400"
	}
	
	log.Println("AI Router service listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}