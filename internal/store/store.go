package store

import (
	"context"
	"errors"
	"sync"
)

type ClaimStatus string

const (
	StatusOpen        ClaimStatus = "OPEN"
	StatusUnderReview ClaimStatus = "UNDER_REVIEW"
	StatusRefunded    ClaimStatus = "REFUNDED"
	StatusRejected    ClaimStatus = "REJECTED"
)

type Claim struct {
	ID              string      `json:"id"`
	OrderID         string      `json:"order_id"`
	UserID          string      `json:"user_id"`
	Amount          int64       `json:"amount"`
	Status          ClaimStatus `json:"status"`
	RefundReference string      `json:"refund_reference,omitempty"`
}

var (
	ErrClaimNotFound = errors.New("claim not found")
	ErrInvalidStatus = errors.New("invalid claim status for action")
)

type Store interface {
	GetClaim(ctx context.Context, id string) (*Claim, error)
	SaveClaim(ctx context.Context, claim *Claim) error
	UpdateClaimStatus(ctx context.Context, id string, status ClaimStatus, refundRef string) (*Claim, error)
	RollbackRefund(ctx context.Context, id string) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	claims map[string]*Claim
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		claims: make(map[string]*Claim),
	}
}

func (s *MemoryStore) GetClaim(ctx context.Context, id string) (*Claim, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.claims[id]
	if !ok {
		return nil, ErrClaimNotFound
	}
	copied := *c
	return &copied, nil
}

func (s *MemoryStore) SaveClaim(ctx context.Context, claim *Claim) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := *claim
	s.claims[claim.ID] = &copied
	return nil
}

func (s *MemoryStore) UpdateClaimStatus(ctx context.Context, id string, status ClaimStatus, refundRef string) (*Claim, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.claims[id]
	if !ok {
		return nil, ErrClaimNotFound
	}
	c.Status = status
	c.RefundReference = refundRef
	copied := *c
	return &copied, nil
}

func (s *MemoryStore) RollbackRefund(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.claims[id]
	if !ok {
		return ErrClaimNotFound
	}
	c.Status = StatusUnderReview
	c.RefundReference = ""
	return nil
}
