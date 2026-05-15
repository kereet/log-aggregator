package main

import (
	"log-aggregator/internal/database"
	"log-aggregator/internal/handlers"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func SetupLogLevel(level string) slog.Level {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	return slogLevel
}

func main() {
	logLevel := SetupLogLevel(os.Getenv("LOG_LEVEL"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://loguser:logpass@localhost:5432/logsdb?sslmode=disable"
		slog.Warn("DATABASE_URL not set, using default", "url", databaseURL)
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
		slog.Warn("PORT not set, using default", "port", serverPort)
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to database")

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
	slog.Info("starting server", "port", serverPort)
	slog.Info("API available", "url", "http://localhost:"+serverPort+"/api/v1/")

	if err := http.ListenAndServe(serverAddr, loggedMux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
