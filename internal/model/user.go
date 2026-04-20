package model

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	UserName     string
	Email        string
	PasswordHash string
}

type UserCreateDTO struct {
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserUpdateDTO struct {
	UserName *string `json:"user_name"`
	Email    *string `json:"email"`
}

type UserLoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

type Session struct {
	UserID       uuid.UUID
	RefreshToken string
	ExpiresAt    time.Time
}
