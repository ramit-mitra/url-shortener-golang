package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	profiler "github.com/blackfireio/go-continuous-profiling"
	"github.com/jackc/pgx/v4"
	"github.com/robfig/cron/v3"
)

type DefaultResponse struct {
	Message string `json:"message"`
}

type Payload struct {
	URL     string `json:"url"`
	Single  *bool  `json:"single,omitempty"`
	Expires *int64 `json:"expires,omitempty"`
}

// PostgreSQL connection
var dbConn *pgx.Conn

// Connect to PostgreSQL
func connectToDB() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	dbConn, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("😭 Unable to connect to database: %v\n", err)
	}
	log.Println("🎉 Connected to PostgreSQL")
}

// Create URLs table if it doesn't exist
func createTableIfNotExists() {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		shortcode VARCHAR(255) UNIQUE NOT NULL,
		url TEXT NOT NULL,
		single_use BOOLEAN DEFAULT FALSE,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := dbConn.Exec(context.Background(), query)
	if err != nil {
		log.Fatalf("❌ Failed to create table: %v\n", err)
	}
	log.Println("🧑🏽‍💻 Table 'urls' is ready")
}

// Save URL to PostgreSQL
func saveURLToDB(payload Payload, shortcode string) error {
	var expiresAt time.Time
	if payload.Expires != nil {
		expiresAt = time.Unix(*payload.Expires, 0)
	} else {
		expiresAt = time.Time{} // Set to zero time if expires is not set
	}

	singleUse := false
	if payload.Single != nil {
		singleUse = *payload.Single
	}

	_, err := dbConn.Exec(context.Background(),
		"INSERT INTO urls (shortcode, url, single_use, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)",
		shortcode, payload.URL, singleUse, expiresAt, time.Now())
	if err != nil {
		return err
	}
	return nil
}

// Get URL from PostgreSQL
func getURLFromDB(shortcode string) (string, bool, time.Time, error) {
	var url string
	var singleUse bool
	var expiresAt time.Time
	err := dbConn.QueryRow(context.Background(),
		"SELECT url, single_use, expires_at FROM urls WHERE shortcode=$1", shortcode).Scan(&url, &singleUse, &expiresAt)
	if err != nil {
		return "", false, time.Time{}, err
	}
	return url, singleUse, expiresAt, nil
}

// Delete URL from PostgreSQL
func deleteURLFromDB(shortcode string) error {
	_, err := dbConn.Exec(context.Background(), "DELETE FROM urls WHERE shortcode=$1", shortcode)
	return err
}

// Default handler for GET and POST
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("=> " + r.Method + " " + r.URL.Path)

	// Handle GET request
	if r.Method == "GET" {
		data := DefaultResponse{
			Message: "🧑🏽‍💻 Welcome to URL shortener",
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Write(jsonData)
		return
	}

	// Handle POST request
	if r.Method == "POST" {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var payload Payload
		err = json.Unmarshal(bodyBytes, &payload)
		if err != nil {
			http.Error(w, "Error parsing request body", http.StatusInternalServerError)
			return
		}

		// Set default values if missing
		if payload.Single == nil {
			defaultSingle := false
			payload.Single = &defaultSingle
		}
		if payload.Expires == nil {
			defaultExpires := int64(0)
			payload.Expires = &defaultExpires
		}

		// Generate a shortcode (for simplicity, using the current timestamp)
		shortcode := fmt.Sprintf("%d", time.Now().UnixNano())

		// Save the URL to the database
		err = saveURLToDB(payload, shortcode)
		if err != nil {
			http.Error(w, "Error saving to database", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"shortcode":"%s"}`, shortcode)))
		return
	}
}

// Redirect handler for the shortcode
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortcode := r.URL.Path[len("/short/"):]

	// Get the URL from the database
	url, singleUse, expiresAt, err := getURLFromDB(shortcode)
	if err != nil || url == "" {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Check if the link has expired
	if time.Now().After(expiresAt) {
		// Delete expired link
		deleteURLFromDB(shortcode)
		http.Error(w, "URL has expired", http.StatusGone)
		return
	}

	// Redirect to the original URL
	http.Redirect(w, r, url, http.StatusFound)

	// If it's a single-use link, delete it after redirecting
	if singleUse {
		deleteURLFromDB(shortcode)
	}
}

// Clean expired links
func cleanExpiredLinks() {
	_, err := dbConn.Exec(context.Background(), "DELETE FROM urls WHERE expires_at < $1", time.Now())
	if err != nil {
		log.Println("Error cleaning expired links:", err)
	}
}

func main() {
	// start blackfire profiler
	err := profiler.Start(
		profiler.WithAppName("url-shortener-golang"),
	)
	if err != nil {
		panic("😭 Error while starting Profiler")
	}
	defer profiler.Stop()

	// Connect to PostgreSQL
	connectToDB()
	defer dbConn.Close(context.Background())

	// Create URLs table if it doesn't exist
	createTableIfNotExists()

	// Get port from env
	port := os.Getenv("PORT")
	if port == "" {
		port = "1001"
	}

	// Set up routes
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/short/", redirectHandler)

	// Set up cron job to clean expired links
	c := cron.New()
	c.AddFunc("@every 5m", cleanExpiredLinks)
	c.Start()

	// Start server
	log.Println("🤖 Server starting on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("❌ Unable to start server: %v", err)
	}
}
