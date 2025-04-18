package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/C0d3-5t3w/aServ/cmd/api/crypto"
	"github.com/C0d3-5t3w/aServ/cmd/api/helper"
	"github.com/C0d3-5t3w/aServ/internal/config"
	"github.com/C0d3-5t3w/aServ/internal/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	requestCounts  = make(map[string][]time.Time)
	requestCountMu sync.Mutex
)

func AuthMiddleware(cfg *config.Config, st *storage.Storage) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				helper.RespondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			tokenParts := strings.Split(authHeader, "Bearer ")
			if len(tokenParts) != 2 {
				helper.RespondWithError(w, http.StatusUnauthorized, "Invalid token format")
				return
			}

			token := tokenParts[1]

			userID, err := crypto.Decode(token, cfg.Auth.Secret)
			if err != nil {
				helper.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			user, err := st.GetUser(userID)
			if err != nil {
				helper.RespondWithError(w, http.StatusUnauthorized, "User not found")
				return
			}

			ctx := helper.SetUserContext(r.Context(), userID)
			ctx = context.WithValue(ctx, "userRole", user.Role)

			if cfg.Features.Audit {
				path := r.URL.Path
				logAuditRequest(st, userID, r.Method, path)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("userRole").(string)

		if !ok || role != storage.RoleAdmin {
			helper.RespondWithError(w, http.StatusForbidden, "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddleware(cfg *config.Config) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.RateLimit.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientID := r.RemoteAddr
			if userID, ok := helper.GetUserFromContext(r.Context()); ok {
				clientID = userID
			}

			if isRateLimited(clientID, cfg.RateLimit.MaxPerMin) {
				helper.RespondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrw, r)

		duration := time.Since(start)
		logRequest(r, wrw.statusCode, duration)
	})
}

func isRateLimited(clientID string, maxRequestsPerMinute int) bool {
	now := time.Now()

	requestCountMu.Lock()
	defer requestCountMu.Unlock()

	var validTimes []time.Time
	for _, t := range requestCounts[clientID] {
		if now.Sub(t) < time.Minute {
			validTimes = append(validTimes, t)
		}
	}

	requestCounts[clientID] = append(validTimes, now)

	return len(requestCounts[clientID]) > maxRequestsPerMinute
}

func logAuditRequest(st *storage.Storage, userID, method, path string) {
	entity := "unknown"
	entityID := ""
	action := "access"

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		switch parts[1] {
		case "users":
			entity = "user"
			if len(parts) > 2 {
				entityID = parts[2]
			}
		case "items":
			entity = "item"
			if len(parts) > 2 {
				entityID = parts[2]
			}
		case "categories":
			entity = "category"
			if len(parts) > 2 {
				entityID = parts[2]
			}
		case "tags":
			entity = "tag"
			if len(parts) > 2 {
				entityID = parts[2]
			}
		}

		switch method {
		case "POST":
			action = "create"
		case "PUT":
			action = "update"
		case "DELETE":
			action = "delete"
		case "GET":
			action = "read"
		}
	}

	log := storage.AuditLog{
		ID:        uuid.New().String(),
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		UserID:    userID,
		Timestamp: time.Now(),
		Details:   method + " " + path,
	}

	go st.CreateAuditLog(log)
}

func logRequest(r *http.Request, statusCode int, duration time.Duration) {

}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
