package main

import (
    "database/sql"
    "log"
    "net/http"
    "os"

    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "go-backend-project/internal/handlers"
    "go-backend-project/internal/services"
)

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // <-- Add Authorization here
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func main() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using environment variables")
    }

    // Inisialisasi googleOauthConfig setelah env ter-load
    handlers.InitGoogleOauthConfig()

    // Get config from env
    connStr := os.Getenv("DB_CONN")

    // Database connection
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("Failed to connect to the database: %v", err)
    }
    defer db.Close()

    // Initialize service and handler
    service := services.NewService(db)
    handler := handlers.NewHandler(service)

    // Create a new ServeMux
    mux := http.NewServeMux()

    // Register routes
    mux.HandleFunc("/api/security-logs", handler.FetchAlert)
    mux.HandleFunc("/api/domain", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet {
            handler.GetDomainsHandler(w, r)
        } else if r.Method == http.MethodPost {
            handler.AddDomainHandler(w, r)
        } else if r.Method == http.MethodDelete {
            handler.DeleteDomainHandler(w, r)
        } else {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    })
    mux.HandleFunc("/api/auth/login", handlers.OauthLoginHandler)
    mux.HandleFunc("/api/auth/callback", handler.OauthCallbackHandler)
    mux.HandleFunc("/api/user", handler.GetUserHandler)

    // Wrap the mux with AuthMiddleware, then with CORS
    handlerWithAuth := handlers.AuthMiddleware(mux)
    log.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", corsMiddleware(handlerWithAuth)); err != nil {
        log.Fatalf("Could not start server: %s\n", err)
    }
}