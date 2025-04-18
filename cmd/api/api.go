package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/C0d3-5t3w/aServ/cmd/api/crypto"
	"github.com/C0d3-5t3w/aServ/cmd/api/helper"
	"github.com/C0d3-5t3w/aServ/internal/config"
	"github.com/C0d3-5t3w/aServ/internal/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type UserLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type ItemRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type TagRequest struct {
	Name string `json:"name"`
}

var cfg *config.Config
var st *storage.Storage

func RegisterRoutes(router *mux.Router, config *config.Config, store *storage.Storage) {
	cfg = config
	st = store

	apiRouter := router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/hello", helloHandler).Methods("GET")

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/login", loginHandler).Methods("POST")
	authRouter.HandleFunc("/register", registerHandler).Methods("POST")

	usersRouter := apiRouter.PathPrefix("/users").Subrouter()
	usersRouter.Use(authMiddleware)
	usersRouter.HandleFunc("", listUsersHandler).Methods("GET")
	usersRouter.HandleFunc("/{id}", getUserHandler).Methods("GET")

	itemsRouter := apiRouter.PathPrefix("/items").Subrouter()
	itemsRouter.Use(authMiddleware)
	itemsRouter.HandleFunc("", listItemsHandler).Methods("GET")
	itemsRouter.HandleFunc("", createItemHandler).Methods("POST")
	itemsRouter.HandleFunc("/{id}", getItemHandler).Methods("GET")
	itemsRouter.HandleFunc("/{id}", updateItemHandler).Methods("PUT")
	itemsRouter.HandleFunc("/{id}", deleteItemHandler).Methods("DELETE")

	tagsRouter := apiRouter.PathPrefix("/tags").Subrouter()
	tagsRouter.Use(authMiddleware)
	tagsRouter.HandleFunc("", createTagHandler).Methods("POST")
	tagsRouter.HandleFunc("/{id}/items", getTagItemsHandler).Methods("GET")

	apiRouter.HandleFunc("/search", searchHandler).Methods("GET")
	apiRouter.HandleFunc("/analytics", getAnalyticsHandler).Methods("GET")
	apiRouter.HandleFunc("/analytics/refresh", refreshAnalyticsHandler).Methods("POST")
	apiRouter.HandleFunc("/audit-logs", getAuditLogsHandler).Methods("GET")
	apiRouter.HandleFunc("/images/upload", imageUploadHandler).Methods("POST")

	router.PathPrefix("/dashboard/").HandlerFunc(DashboardHandler)
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./cmd/api/dashboard/pages/index.html")
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	helper.RespondWithSuccess(w, http.StatusOK, "API is working", map[string]string{
		"version": "1.0.0",
		"name":    cfg.AppName,
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Username == "" || req.Password == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := st.GetUserByUsername(req.Username)
	if err != nil {
		helper.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !crypto.VerifyPassword(req.Password, user.Password) {
		helper.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := crypto.Encode(user.ID, cfg.Auth.Secret)
	if err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not generate token")
		return
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Login successful", map[string]string{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if !helper.ValidateUsername(req.Username) {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid username format")
		return
	}
	if !helper.ValidateEmail(req.Email) {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
		return
	}
	if !helper.ValidatePassword(req.Password) {
		helper.RespondWithError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	_, err := st.GetUserByUsername(req.Username)
	if err == nil {
		helper.RespondWithError(w, http.StatusConflict, "Username already taken")
		return
	}

	user := storage.User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Password:  crypto.HashPassword(req.Password),
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	if err := st.CreateUser(user); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	helper.RespondWithSuccess(w, http.StatusCreated, "User created successfully", map[string]string{
		"user_id": user.ID,
	})
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := st.ListUsers()

	for i := range users {
		users[i].Password = ""
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Users retrieved", users)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := st.GetUser(id)
	if err != nil {
		helper.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	user.Password = ""

	helper.RespondWithSuccess(w, http.StatusOK, "User retrieved", user)
}

func listItemsHandler(w http.ResponseWriter, r *http.Request) {
	items := st.ListItems()
	helper.RespondWithSuccess(w, http.StatusOK, "Items retrieved", items)
}

func getItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	item, err := st.GetItem(id)
	if err != nil {
		helper.RespondWithError(w, http.StatusNotFound, "Item not found")
		return
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Item retrieved", item)
}

func createItemHandler(w http.ResponseWriter, r *http.Request) {
	var req ItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Name == "" || req.Price < 0 {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid item data")
		return
	}

	userID := r.Context().Value("userID").(string)

	item := storage.Item{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		CreatedAt:   time.Now(),
		CreatedBy:   userID,
	}

	if err := st.CreateItem(item); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not create item")
		return
	}

	helper.RespondWithSuccess(w, http.StatusCreated, "Item created", item)
}

func updateItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	existingItem, err := st.GetItem(id)
	if err != nil {
		helper.RespondWithError(w, http.StatusNotFound, "Item not found")
		return
	}

	var req ItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Name == "" || req.Price < 0 {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid item data")
		return
	}

	userID := r.Context().Value("userID").(string)

	if existingItem.CreatedBy != userID {
		helper.RespondWithError(w, http.StatusForbidden, "You don't have permission to update this item")
		return
	}

	existingItem.Name = req.Name
	existingItem.Description = req.Description
	existingItem.Price = req.Price

	if err := st.UpdateItem(existingItem); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not update item")
		return
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Item updated", existingItem)
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	existingItem, err := st.GetItem(id)
	if err != nil {
		helper.RespondWithError(w, http.StatusNotFound, "Item not found")
		return
	}

	userID := r.Context().Value("userID").(string)
	userRole := r.Context().Value("userRole").(string)

	if existingItem.CreatedBy != userID && userRole != storage.RoleAdmin {
		helper.RespondWithError(w, http.StatusForbidden, "You don't have permission to delete this item")
		return
	}

	if err := st.DeleteItem(id); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not delete item")
		return
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Item deleted", nil)
}

func createTagHandler(w http.ResponseWriter, r *http.Request) {
	var req TagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Name == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "Tag name is required")
		return
	}

	userID := r.Context().Value("userID").(string)

	tag := storage.Tag{
		ID:        uuid.New().String(),
		Name:      req.Name,
		CreatedAt: time.Now(),
		CreatedBy: userID,
	}

	if err := st.CreateTag(tag); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not create tag")
		return
	}

	helper.RespondWithSuccess(w, http.StatusCreated, "Tag created", tag)
}

func getTagItemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, err := st.GetTag(id); err != nil {
		helper.RespondWithError(w, http.StatusNotFound, "Tag not found")
		return
	}

	items := st.GetItemsByTag(id)
	helper.RespondWithSuccess(w, http.StatusOK, "Tag items retrieved", items)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		helper.RespondWithError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	entityType := r.URL.Query().Get("type")
	var results interface{}

	switch entityType {
	case "users":
		results = st.SearchUsers(query)
	case "items":
		results = st.SearchItems(query)
	default:
		userResults := st.SearchUsers(query)
		itemResults := st.SearchItems(query)

		results = map[string]interface{}{
			"users": userResults,
			"items": itemResults,
		}
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Search results", results)
}

func getAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	analytics := st.GetAnalytics()
	helper.RespondWithSuccess(w, http.StatusOK, "Analytics retrieved", analytics)
}

func refreshAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	st.UpdateAnalytics()
	analytics := st.GetAnalytics()
	helper.RespondWithSuccess(w, http.StatusOK, "Analytics refreshed", analytics)
}

func getAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	logs := st.GetAuditLogs(limit)
	helper.RespondWithSuccess(w, http.StatusOK, "Audit logs retrieved", logs)
}

func imageUploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "Could not parse form")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		helper.RespondWithError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	filename := uuid.New().String() + "-" + handler.Filename

	helper.RespondWithSuccess(w, http.StatusOK, "Image uploaded", map[string]string{
		"image_url": "/api/images/items/" + filename,
	})
}

func authMiddleware(next http.Handler) http.Handler {
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
		ctx = helper.SetUserRoleContext(ctx, user.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
