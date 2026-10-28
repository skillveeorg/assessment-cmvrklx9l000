package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"app/internal/api"
	"app/internal/service"
	"app/internal/store"
)

func main() {
	st := store.NewMemoryStore()
	seedData(st)

	processor := service.NewMockPaymentGateway()
	claimSvc := service.NewClaimService()
	mux := api.NewMux(st, claimSvc, processor)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("Starting cx-claims-service on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failure: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down cx-claims-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}
}

func seedData(st store.Store) {
	claims := []*store.Claim{
		{
			ID:              "CLM-1001",
			OrderID:         "ORD-8001",
			UserID:          "USR-3001",
			Amount:          15000,
			Status:          store.StatusUnderReview,
			RefundReference: "",
		},
		{
			ID:              "CLM-1002",
			OrderID:         "ORD-8002",
			UserID:          "USR-3002",
			Amount:          8500,
			Status:          store.StatusOpen,
			RefundReference: "",
		},
		{
			ID:              "CLM-1003",
			OrderID:         "ORD-8003",
			UserID:          "USR-3003",
			Amount:          24000,
			Status:          store.StatusUnderReview,
			RefundReference: "",
		},
	}

	for _, c := range claims {
		_ = st.SaveClaim(context.Background(), c)
	}
}
