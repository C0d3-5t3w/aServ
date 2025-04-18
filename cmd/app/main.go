package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/C0d3-5t3w/aServ/cmd/api"
	"github.com/C0d3-5t3w/aServ/cmd/api/dashboard"
	"github.com/C0d3-5t3w/aServ/internal/config"
	"github.com/C0d3-5t3w/aServ/internal/storage"
	"github.com/gorilla/mux"
)

func main() {
	log.Println("Starting aServ application...")

	cfg := config.LoadConfig()
	log.Printf("Loaded configuration for: %s", cfg.AppName)

	st := storage.NewStorage()
	log.Println("Storage initialized")

	router := mux.NewRouter().StrictSlash(true)

	api.RegisterRoutes(router, cfg, st)
	log.Println("API routes registered")

	dashboard.Routes(router)
	log.Println("Dashboard routes registered")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
