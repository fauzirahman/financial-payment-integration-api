package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fauzirahman/financial-payment-integration-api/internal/config"
	"github.com/fauzirahman/financial-payment-integration-api/internal/database"
	"github.com/fauzirahman/financial-payment-integration-api/internal/handler"
	"github.com/fauzirahman/financial-payment-integration-api/internal/repository"
	"github.com/fauzirahman/financial-payment-integration-api/internal/service"
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

	paymentRepository := repository.NewPostgresPaymentRepository(db)
	paymentService := service.NewPaymentService(paymentRepository)
	customerRepository := repository.NewPostgresCustomerRepository(db)
	customerService := service.NewCustomerService(customerRepository)
	accountRepository := repository.NewPostgresAccountRepository(db)
	accountService := service.NewAccountService(accountRepository)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	customerHandler := handler.NewCustomerHandler(customerService)
	accountHandler := handler.NewAccountHandler(accountService)
	webhookHandler := handler.NewWebhookHandler(paymentService, cfg.WebhookSecret)

	http.HandleFunc("/health", healthHandler)
	http.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("./docs"))))
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/api/v1/customers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			customerHandler.GetCustomers(w, r)
		case http.MethodPost:
			customerHandler.CreateCustomer(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/v1/customers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/customers/")
		if strings.Contains(path, "/accounts") || strings.HasSuffix(path, "/accounts") {
			accountHandler.GetAccountsByCustomerID(w, r)
			return
		}
		customerHandler.GetCustomerByID(w, r)
	})

	http.HandleFunc("/api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			accountHandler.CreateAccount(w, r)
			return
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})
	http.HandleFunc("/api/v1/accounts/", accountHandler.GetAccountByID)

	http.HandleFunc("/api/v1/payments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			paymentHandler.GetPayments(w, r)

		case http.MethodPost:
			paymentHandler.CreatePayment(w, r)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/v1/payments/", paymentHandler.GetPaymentByID)
	http.HandleFunc("/api/v1/webhooks/payment", webhookHandler.HandlePaymentWebhook)

	fmt.Println("Financial Payment Integration API")
	fmt.Printf("Server listening on port %s\n", cfg.AppPort)

	err = http.ListenAndServe(":"+cfg.AppPort, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
