package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/config"
	"github.com/fauzirahman/financial-payment-integration-api/internal/database"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "ok",
		Service: "financial-payment-api",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Config error:", err)
		return
	}

	db, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		fmt.Println("Database error:", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int

	err = db.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		fmt.Println("Database query error:", err)
		return
	}

	fmt.Println("Database connection successful")
	fmt.Println("SELECT 1 result:", result)

	http.HandleFunc("/health", healthHandler)

	fmt.Println("Financial Payment Integration API")
	fmt.Printf("Server running on http://localhost:%s\n", cfg.AppPort)

	err = http.ListenAndServe(":"+cfg.AppPort, nil)

	if err != nil {
		fmt.Println("Server error:", err)
	}
}
