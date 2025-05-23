package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"golink/config"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golink/database"
	"golink/model"
)

type Handler struct {
	DB *database.Database
}

// NewHandler creates a new Handler instance
func NewHandler(db *database.Database) *Handler {
	return &Handler{DB: db}
}

// InitializeRoutes sets up all routes
func (h *Handler) InitializeRoutes(router *mux.Router) {
	// Public route for redirecting short URLs

	// Protected routes with Basic Auth
	router.HandleFunc("/", basicAuth(h.homePage)).Methods("GET")
	router.HandleFunc("/create", basicAuth(h.createURL)).Methods("POST")
	router.HandleFunc("/list", basicAuth(h.listURLs)).Methods("GET")
	router.HandleFunc("/{shortCode}", h.redirectURL).Methods("GET")

}

// Protected routes without Basic Auth}

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

func (h *Handler) homePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (h *Handler) createURL(w http.ResponseWriter, r *http.Request) {
	originalURL := r.FormValue("url")
	customCode := r.FormValue("custom_code")

	var shortCode string
	var err error

	result := model.ResultData{}

	if customCode != "" {
		// Check if custom code already exists
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)", customCode).Scan(&exists)
		if err != nil {
			result.Error = "Database error"
			h.renderResult(w, result)
			return
		}
		if exists {
			result.Error = "Custom code already exists"
			h.renderResult(w, result)
			return
		}
		shortCode = customCode
	} else {
		// Generate random code
		for {
			shortCode, err = generateRandomCode(config.Config.ShortCodeLength)
			if err != nil {
				result.Error = "Error generating short code"
				h.renderResult(w, result)
				return
			}
			var exists bool
			err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)", shortCode).Scan(&exists)
			if err != nil {
				result.Error = "Database error"
				h.renderResult(w, result)
				return
			}
			if !exists {
				break
			}
		}
	}

	// Insert into database
	_, err = h.DB.Exec("INSERT INTO urls (original_url, short_code, created_at) VALUES ($1, $2, $3)",
		originalURL, shortCode, time.Now())
	if err != nil {
		result.Error = "Failed to create short URL"
		h.renderResult(w, result)
		return
	}

	// Set the short URL
	shortURL := fmt.Sprintf("%s://%s/%s", r.URL.Scheme, r.Host, shortCode)
	if r.URL.Scheme == "" {
		shortURL = fmt.Sprintf("http://%s/%s", r.Host, shortCode)
	}
	result.Success = true
	result.ShortURL = shortURL

	h.renderResult(w, result)
}

func (h *Handler) renderResult(w http.ResponseWriter, data model.ResultData) {
	tmpl, err := template.ParseFiles("templates/result.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func (h *Handler) redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["shortCode"]

	var originalURL string
	err := h.DB.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&originalURL)
	if err == sql.ErrNoRows {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func (h *Handler) listURLs(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	var urls []model.URL
	var rows *sql.Rows
	var err error

	if search != "" {
		rows, err = h.DB.Query("SELECT id, original_url, short_code, created_at FROM urls WHERE original_url ILIKE $1 ORDER BY created_at DESC", "%"+search+"%")
	} else {
		rows, err = h.DB.Query("SELECT id, original_url, short_code, created_at FROM urls ORDER BY created_at DESC")
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.ID, &url.Original, &url.ShortCode, &url.CreatedAt); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		urls = append(urls, url)
	}

	baseURL := fmt.Sprintf("%s://%s", r.URL.Scheme, r.Host)
	if r.URL.Scheme == "" {
		baseURL = fmt.Sprintf("http://%s", r.Host)
	}
	data := struct {
		URLs    []model.URL
		Search  string
		BaseURL string
	}{
		URLs:    urls,
		Search:  search,
		BaseURL: baseURL,
	}

	tmpl, err := template.ParseFiles("templates/list.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
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
