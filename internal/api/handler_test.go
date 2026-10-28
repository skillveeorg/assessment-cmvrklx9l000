package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/api"
	"app/internal/service"
	"app/internal/store"
)

type mockProcessor struct {
	processRefundFn func(ctx context.Context, claimID string, amount int64) (string, error)
}

func (m *mockProcessor) ProcessRefund(ctx context.Context, claimID string, amount int64) (string, error) {
	if m.processRefundFn != nil {
		return m.processRefundFn(ctx, claimID, amount)
	}
	return "GW-DEFAULT", nil
}

func TestRetainsReviewStatusForRejectedGatewayTransactions(t *testing.T) {
	st := store.NewMemoryStore()
	initialClaim := &store.Claim{
		ID:              "CLM-302",
		OrderID:         "ORD-5502",
		UserID:          "USR-9002",
		Amount:          3200,
		Status:          store.StatusUnderReview,
		RefundReference: "",
	}
	if err := st.SaveClaim(context.Background(), initialClaim); err != nil {
		t.Fatalf("failed to save test claim: %v", err)
	}

	proc := &mockProcessor{
		processRefundFn: func(pCtx context.Context, claimID string, amount int64) (string, error) {
			return "", errors.New("gateway: transaction rejected by issuer")
		},
	}

	svc := service.NewClaimService()
	mux := api.NewMux(st, svc, proc)

	req := httptest.NewRequest(http.MethodPost, "/claims/CLM-302/refund", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	stored, err := st.GetClaim(context.Background(), "CLM-302")
	if err != nil {
		t.Fatalf("failed to get claim from store: %v", err)
	}

	if stored.Status != store.StatusUnderReview {
		t.Fatalf("expected claim status to remain %s after gateway rejection, got %s", store.StatusUnderReview, stored.Status)
	}
}

func TestClearsPendingRefundReferenceOnClientRequestTermination(t *testing.T) {
	st := store.NewMemoryStore()
	initialClaim := &store.Claim{
		ID:              "CLM-301",
		OrderID:         "ORD-5501",
		UserID:          "USR-9001",
		Amount:          4500,
		Status:          store.StatusUnderReview,
		RefundReference: "",
	}
	if err := st.SaveClaim(context.Background(), initialClaim); err != nil {
		t.Fatalf("failed to save test claim: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc := &mockProcessor{
		processRefundFn: func(pCtx context.Context, claimID string, amount int64) (string, error) {
			cancel()
			return "", context.Canceled
		},
	}

	svc := service.NewClaimService()
	mux := api.NewMux(st, svc, proc)

	req := httptest.NewRequest(http.MethodPost, "/claims/CLM-301/refund", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	stored, err := st.GetClaim(context.Background(), "CLM-301")
	if err != nil {
		t.Fatalf("failed to get claim from store: %v", err)
	}

	if stored.RefundReference != "" {
		t.Fatalf("expected refund reference to be cleared on client cancellation, got %q", stored.RefundReference)
	}
}

func TestReturnsErrorResponsePayloadOnGatewayCommunicationFault(t *testing.T) {
	st := store.NewMemoryStore()
	initialClaim := &store.Claim{
		ID:              "CLM-303",
		OrderID:         "ORD-5503",
		UserID:          "USR-9003",
		Amount:          8900,
		Status:          store.StatusUnderReview,
		RefundReference: "",
	}
	if err := st.SaveClaim(context.Background(), initialClaim); err != nil {
		t.Fatalf("failed to save test claim: %v", err)
	}

	proc := &mockProcessor{
		processRefundFn: func(pCtx context.Context, claimID string, amount int64) (string, error) {
			return "", errors.New("gateway: connection reset by peer")
		},
	}

	svc := service.NewClaimService()
	mux := api.NewMux(st, svc, proc)

	req := httptest.NewRequest(http.MethodPost, "/claims/CLM-303/refund", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected non-200 error response status code on gateway fault, got %d", rec.Code)
	}
}

func TestFetchesClaimDetailsByIdentifier(t *testing.T) {
	st := store.NewMemoryStore()
	initialClaim := &store.Claim{
		ID:              "CLM-101",
		OrderID:         "ORD-2001",
		UserID:          "USR-4001",
		Amount:          1500,
		Status:          store.StatusUnderReview,
		RefundReference: "",
	}
	if err := st.SaveClaim(context.Background(), initialClaim); err != nil {
		t.Fatalf("failed to save test claim: %v", err)
	}

	proc := &mockProcessor{}
	svc := service.NewClaimService()
	mux := api.NewMux(st, svc, proc)

	req := httptest.NewRequest(http.MethodGet, "/claims/CLM-101", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res store.Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.ID != "CLM-101" || res.Status != store.StatusUnderReview {
		t.Fatalf("unexpected claim details: %+v", res)
	}
}

func TestProcessesValidRefundRequestSuccessfully(t *testing.T) {
	st := store.NewMemoryStore()
	initialClaim := &store.Claim{
		ID:              "CLM-102",
		OrderID:         "ORD-2002",
		UserID:          "USR-4002",
		Amount:          2500,
		Status:          store.StatusUnderReview,
		RefundReference: "",
	}
	if err := st.SaveClaim(context.Background(), initialClaim); err != nil {
		t.Fatalf("failed to save test claim: %v", err)
	}

	proc := &mockProcessor{
		processRefundFn: func(pCtx context.Context, claimID string, amount int64) (string, error) {
			return "GW-REF-999", nil
		},
	}
	svc := service.NewClaimService()
	mux := api.NewMux(st, svc, proc)

	req := httptest.NewRequest(http.MethodPost, "/claims/CLM-102/refund", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res store.Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Status != store.StatusRefunded {
		t.Fatalf("expected status %s, got %s", store.StatusRefunded, res.Status)
	}

	stored, err := st.GetClaim(context.Background(), "CLM-102")
	if err != nil {
		t.Fatalf("failed to get claim: %v", err)
	}
	if stored.Status != store.StatusRefunded {
		t.Fatalf("expected stored status %s, got %s", store.StatusRefunded, stored.Status)
	}
}
