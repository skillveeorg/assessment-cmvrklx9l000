package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/internal/store"
)

var (
	ErrIneligibleClaim = errors.New("claim is not eligible for refund")
	ErrInvalidAmount   = errors.New("claim amount must be greater than zero")
)

type PaymentProcessor interface {
	ProcessRefund(ctx context.Context, claimID string, amount int64) (string, error)
}

type ClaimService struct{}

func NewClaimService() *ClaimService {
	return &ClaimService{}
}

func (s *ClaimService) ValidateRefundEligibility(claim *store.Claim) error {
	if claim == nil {
		return errors.New("claim cannot be nil")
	}
	if claim.Status != store.StatusUnderReview {
		return ErrIneligibleClaim
	}
	if claim.Amount <= 0 {
		return ErrInvalidAmount
	}
	return nil
}

type MockPaymentGateway struct{}

func NewMockPaymentGateway() *MockPaymentGateway {
	return &MockPaymentGateway{}
}

func (g *MockPaymentGateway) ProcessRefund(ctx context.Context, claimID string, amount int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("GW-%s-%d", claimID, time.Now().Unix()), nil
}
