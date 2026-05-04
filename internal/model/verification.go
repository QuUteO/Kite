package model

import (
	"time"

	"github.com/google/uuid"
)

type VerificationStatus string

const (
	StatusPending  VerificationStatus = "pending"
	StatusVerified VerificationStatus = "verified"
	StatusExpired  VerificationStatus = "expired"
)

type Verification struct {
	ID        uuid.UUID          `json:"id"`
	UserID    uuid.UUID          `json:"user_id"`
	Code      string             `json:"code"`
	Status    VerificationStatus `json:"status"`
	Type      string             `json:"type"`
	ExpiresAt time.Time          `json:"expires_at"`
	CreatedAt time.Time          `json:"created_at"`
}

type VerificationDTO struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type ResendCodeDTO struct {
	Email string `json:"email"`
}
