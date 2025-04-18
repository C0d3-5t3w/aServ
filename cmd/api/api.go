package api

import (
	"encoding/json"
	"net/http"
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

	if existingItem.CreatedBy != userID {
		helper.RespondWithError(w, http.StatusForbidden, "You don't have permission to delete this item")
		return
	}

	if err := st.DeleteItem(id); err != nil {
		helper.RespondWithError(w, http.StatusInternalServerError, "Could not delete item")
		return
	}

	helper.RespondWithSuccess(w, http.StatusOK, "Item deleted", nil)
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

		_, err = st.GetUser(userID)
		if err != nil {
			helper.RespondWithError(w, http.StatusUnauthorized, "User not found")
			return
		}

		ctx := helper.SetUserContext(r.Context(), userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
