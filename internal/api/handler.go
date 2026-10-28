package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"app/internal/service"
	"app/internal/store"
)

type Handler struct {
	store     store.Store
	service   *service.ClaimService
	processor service.PaymentProcessor
}

func NewMux(st store.Store, svc *service.ClaimService, proc service.PaymentProcessor) http.Handler {
	h := &Handler{
		store:     st,
		service:   svc,
		processor: proc,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /claims/{id}", h.handleGetClaim)
	mux.HandleFunc("POST /claims/{id}/refund", h.handleRefund)
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	return mux
}

func (h *Handler) handleGetClaim(w http.ResponseWriter, r *http.Request) {
	claimID := r.PathValue("id")
	if claimID == "" {
		http.Error(w, "missing claim id", http.StatusBadRequest)
		return
	}

	claim, err := h.store.GetClaim(r.Context(), claimID)
	if err != nil {
		if errors.Is(err, store.ErrClaimNotFound) {
			http.Error(w, "claim not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(claim)
}

func (h *Handler) handleRefund(w http.ResponseWriter, r *http.Request) {
	claimID := r.PathValue("id")
	if claimID == "" {
		http.Error(w, "missing claim id", http.StatusBadRequest)
		return
	}

	claim, err := h.store.GetClaim(r.Context(), claimID)
	if err != nil {
		if errors.Is(err, store.ErrClaimNotFound) {
			http.Error(w, "claim not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.service.ValidateRefundEligibility(claim); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	refundRef := fmt.Sprintf("REF-%s-%d", claimID, time.Now().UnixNano())

	updatedClaim, err := h.store.UpdateClaimStatus(r.Context(), claimID, store.StatusRefunded, refundRef)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, procErr := h.processor.ProcessRefund(r.Context(), claimID, claim.Amount)
	if procErr != nil {
		if errors.Is(r.Context().Err(), context.Canceled) {
			_ = h.store.RollbackRefund(r.Context(), claimID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updatedClaim)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
