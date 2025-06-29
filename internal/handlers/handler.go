package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "os"
    "strconv"
    "strings"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"

    "go-backend-project/internal/services"
)

type Handler struct {
    Service *services.Service
}

func NewHandler(service *services.Service) *Handler {
    return &Handler{Service: service}
}

type AddDomainRequest struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}

type AddDomainResponse struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    URL     string `json:"url"`
    Message string `json:"message"`
}

func (h *Handler) FetchAlert(w http.ResponseWriter, r *http.Request) {
    alerts, err := h.Service.FetchAlerts()
    if err != nil {
        http.Error(w, "Failed to fetch alerts", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(alerts)
}

func (h *Handler) AddDomainHandler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    token := strings.TrimPrefix(authHeader, "Bearer ")
    userInfo, err := GetUserFromGoogleToken(token)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }
    user, err := h.Service.GetUserByEmail(userInfo.Email)
    if err != nil {
        http.Error(w, "User not found", http.StatusUnauthorized)
        return
    }
    userID, ok := user["id"].(int)
    if !ok {
        http.Error(w, "Invalid user ID", http.StatusInternalServerError)
        return
    }

    var req AddDomainRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    if req.Name == "" || req.URL == "" {
        http.Error(w, "Name and URL are required", http.StatusBadRequest)
        return
    }
    id, name, url, err := h.Service.AddDomain(req.Name, req.URL, userID)
    if err != nil {
        http.Error(w, "Failed to add domain", http.StatusInternalServerError)
        return
    }
    resp := AddDomainResponse{
        ID:      id,
        Name:    name,
        URL:     url,
        Message: "Domain added successfully",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetDomainsHandler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    token := strings.TrimPrefix(authHeader, "Bearer ")
    userInfo, err := GetUserFromGoogleToken(token)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }
    user, err := h.Service.GetUserByEmail(userInfo.Email)
    if err != nil {
        http.Error(w, "User not found", http.StatusUnauthorized)
        return
    }
    userID, ok := user["id"].(int)
    if !ok {
        http.Error(w, "Invalid user ID", http.StatusInternalServerError)
        return
    }
    domains, err := h.Service.GetDomains(userID)
    if err != nil {
        http.Error(w, "Failed to fetch domains", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(domains)
}

func (h *Handler) DeleteDomainHandler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    token := strings.TrimPrefix(authHeader, "Bearer ")
    userInfo, err := GetUserFromGoogleToken(token)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }
    user, err := h.Service.GetUserByEmail(userInfo.Email)
    if err != nil {
        http.Error(w, "User not found", http.StatusUnauthorized)
        return
    }
    userID, ok := user["id"].(int)
    if !ok {
        http.Error(w, "Invalid user ID", http.StatusInternalServerError)
        return
    }
    domainId := r.URL.Query().Get("domainId")
    if domainId == "" {
        http.Error(w, "domainId is required", http.StatusBadRequest)
        return
    }
    id, err := strconv.Atoi(domainId)
    if err != nil {
        http.Error(w, "Invalid domainId", http.StatusBadRequest)
        return
    }
    err = h.Service.DeleteDomain(id, userID)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Domain not found", http.StatusNotFound)
        } else {
            http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
        }
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": "Domain deleted successfully",
    })
}

var googleOauthConfig *oauth2.Config

func InitGoogleOauthConfig() {
    googleOauthConfig = &oauth2.Config{
        RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
        ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        Scopes: []string{
            "https://www.googleapis.com/auth/userinfo.email",
            "https://www.googleapis.com/auth/userinfo.profile",
        },
        Endpoint: google.Endpoint,
    }
}

func OauthLoginHandler(w http.ResponseWriter, r *http.Request) {
    url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type GoogleUserInfo struct {
    Email string `json:"email"`
    Name  string `json:"name"`
    // You can add more fields if needed
}

func (h *Handler) OauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "Code not found", http.StatusBadRequest)
        return
    }
    token, err := googleOauthConfig.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
        return
    }

    userInfo, err := GetUserFromGoogleToken(token.AccessToken)
    if err != nil {
        http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Upsert user in DB
    if err := h.Service.UpsertUser(userInfo.Name, userInfo.Email); err != nil {
        http.Error(w, "Failed to upsert user: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Use FE redirect URL from environment variable
    feRedirectURL := os.Getenv("FE_REDIRECT_URL")
    if feRedirectURL == "" {
        feRedirectURL = "http://syra.insec.my.id/auth/callback"
    }
    redirectURL := feRedirectURL + "?token=" + token.AccessToken
    http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func GetUserFromGoogleToken(token string) (*GoogleUserInfo, error) {
    oauth2Token := &oauth2.Token{AccessToken: token}
    client := googleOauthConfig.Client(oauth2.NoContext, oauth2Token)
    resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var userInfo GoogleUserInfo
    if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
        return nil, err
    }
    return &userInfo, nil
}

func (h *Handler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
        return
    }
    const prefix = "Bearer "
    if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
        http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
        return
    }
    token := authHeader[len(prefix):]

    userInfo, err := GetUserFromGoogleToken(token)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    user, err := h.Service.GetUserByEmail(userInfo.Email)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

// AuthMiddleware checks for the Authorization header and validates the token
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow unauthenticated access to login and callback
        if strings.HasPrefix(r.URL.Path, "/api/auth/login") || strings.HasPrefix(r.URL.Path, "/api/auth/callback") {
            next.ServeHTTP(w, r)
            return
        }

        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Optionally: Validate the token here (e.g., check with Google or verify JWT)
        // If invalid, return 401

        // If valid, continue
        next.ServeHTTP(w, r)
    })
}