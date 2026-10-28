package service_test

import (
	"testing"

	"app/internal/service"
	"app/internal/store"
)

func TestValidatesClaimEligibilityForRefund(t *testing.T) {
	svc := service.NewClaimService()

	validClaim := &store.Claim{
		ID:     "CLM-201",
		Status: store.StatusUnderReview,
		Amount: 5000,
	}
	if err := svc.ValidateRefundEligibility(validClaim); err != nil {
		t.Errorf("expected valid claim to pass validation, got error: %v", err)
	}

	invalidStatuses := []store.ClaimStatus{
		store.StatusOpen,
		store.StatusRefunded,
		store.StatusRejected,
	}

	for _, status := range invalidStatuses {
		invalidClaim := &store.Claim{
			ID:     "CLM-202",
			Status: status,
			Amount: 5000,
		}
		if err := svc.ValidateRefundEligibility(invalidClaim); err == nil {
			t.Errorf("expected claim with status %s to fail validation, got nil", status)
		}
	}
}
