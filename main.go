package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"golink/config"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	DB     *sql.DB
	Router *mux.Router
}

type URL struct {
	ID        int
	Original  string
	ShortCode string
	CreatedAt time.Time
}

// generateRandomCode creates a random short code
func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// basicAuth middleware for protected routes
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != config.Config.BasicAuthUsername || password != config.Config.BasicAuthPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (a *App) initializeRoutes() {
	// Public route for redirecting short URLs
	a.Router.HandleFunc("/{shortCode}", a.redirectURL).Methods("GET")

	// Protected routes with Basic Auth
	a.Router.HandleFunc("/", basicAuth(a.homePage)).Methods("GET")
	a.Router.HandleFunc("/create", basicAuth(a.createURL)).Methods("POST")
}

func (a *App) homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (a *App) createURL(w http.ResponseWriter, r *http.Request) {
	originalURL := r.FormValue("url")
	customCode := r.FormValue("custom_code")

	var shortCode string
	var err error

	if customCode != "" {
		// Check if custom code already exists
		var exists bool
		err = a.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)", customCode).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "Custom code already exists", http.StatusBadRequest)
			return
		}
		shortCode = customCode
	} else {
		// Generate random code
		for {
			shortCode, err = generateRandomCode(config.Config.ShortCodeLength)
			if err != nil {
				http.Error(w, "Error generating short code", http.StatusInternalServerError)
				return
			}
			var exists bool
			err = a.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)", shortCode).Scan(&exists)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			if !exists {
				break
			}
		}
	}

	// Insert into database
	_, err = a.DB.Exec("INSERT INTO urls (original_url, short_code, created_at) VALUES ($1, $2, $3)",
		originalURL, shortCode, time.Now())
	if err != nil {
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	// Return the short URL
	shortURL := fmt.Sprintf("%s/%s", r.Host, shortCode)
	fmt.Fprintf(w, "Short URL created: %s", shortURL)
}

func (a *App) redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	var originalURL string
	err := a.DB.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&originalURL)
	if err == sql.ErrNoRows {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func (a *App) initializeDB() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Config.DB.Host,
		config.Config.DB.Port,
		config.Config.DB.User,
		config.Config.DB.Pass,
		config.Config.DB.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	a.DB = db

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			original_url TEXT NOT NULL,
			short_code TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`)
	return err
}

func main() {
	app := &App{}
	app.Router = mux.NewRouter()

	// Initialize database
	if err := app.initializeDB(); err != nil {
		log.Fatal("Error initializing database: ", err)
	}
	defer app.DB.Close()

	// Initialize routes
	app.initializeRoutes()

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", app.Router); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
