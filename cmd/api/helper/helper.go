package helper

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
)

type contextKey string

const UserIDKey contextKey = "userID"
const UserRoleKey contextKey = "userRole"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, APIResponse{
		Success: false,
		Error:   message,
	})
}

func RespondWithSuccess(w http.ResponseWriter, code int, message string, data interface{}) {
	RespondWithJSON(w, code, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SetUserContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func GetUserFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

func SetUserRoleContext(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, UserRoleKey, role)
}

func GetUserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}

func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func ValidateUsername(username string) bool {

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	return usernameRegex.MatchString(username)
}

func ValidatePassword(password string) bool {

	return len(password) >= 8
}

func ExampleHelper() string {
	return "helper"
}
