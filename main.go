package main

import (
	"golink/config"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"golink/database"
	"golink/handler"
)

func main() {
	// Initialize database
	db, err := database.NewDatabase(
		config.Config.DB.Host,
		config.Config.DB.Port,
		config.Config.DB.User,
		config.Config.DB.Pass,
		config.Config.DB.Name,
	)
	if err != nil {
		log.Fatal("Error initializing database: ", err)
	}
	defer db.Close()

	// Initialize router
	router := mux.NewRouter()

	// Initialize handlers
	h := handler.NewHandler(db)

	// Set up routes
	h.InitializeRoutes(router)

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
