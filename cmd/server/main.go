package main

import (
	"log"
	"log-aggregator/internal/database"
	"log-aggregator/internal/handlers"
	"net/http"
	"os"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://loguser:logpass@localhost:5432/logsdb?sslmode=disable"
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	logStore := database.NewLogStore(db)
	logHandler := handlers.NewHandlers(logStore)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/parse/", logHandler.ParseLog)
	mux.HandleFunc("GET /api/v1/topology/{log_id}", logHandler.GetTopology)
	mux.HandleFunc("GET /api/v1/node/{node_id}", logHandler.GetNode)
	mux.HandleFunc("GET /api/v1/port/{node_id}", logHandler.GetPorts)
	mux.HandleFunc("GET /api/v1/log/{log_id}", logHandler.GetLogInfo)

	loggedMux := loggingMiddleware(mux)

	serverAddr := ":" + serverPort
	log.Printf("Starting server on port %s", serverPort)
	log.Printf("API available at http://localhost:%s/api/v1/", serverPort)

	if err := http.ListenAndServe(serverAddr, loggedMux); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
