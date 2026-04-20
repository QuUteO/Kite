package service

import (
	"Kite/internal/config"
	"Kite/internal/model"
	"Kite/internal/user/repository"
	"Kite/pkg/rabbitmq"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	CreateUser(ctx context.Context, user *model.UserCreateDTO) error
	UpdateUser(ctx context.Context, id uuid.UUID, user *model.UserUpdateDTO) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	LoginUser(ctx context.Context, userDTO *model.UserLoginDTO) (*model.TokenResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

type service struct {
	repo      repository.Repository
	logger    *slog.Logger
	jwtSecret string
	producer  rabbitmq.Publisher
	cfg       *config.Config
}

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func (s *service) CreateUser(ctx context.Context, userDto *model.UserCreateDTO) error {
	logger := s.logger.With("op", "service/CreateUser")

	if userDto == nil {
		return fmt.Errorf("create user: %w", ErrInvalidInput)
	}

	userName := strings.TrimSpace(userDto.UserName)
	email := strings.TrimSpace(userDto.Email)
	password := strings.TrimSpace(userDto.Password)
	if userName == "" || email == "" || password == "" {
		return fmt.Errorf("create user: %w", ErrInvalidInput)
	}

	id := uuid.New()

	hashPass, err := s.hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:           id,
		UserName:     userName,
		Email:        email,
		PasswordHash: hashPass,
	}

	logger.Info("user created in service", "user_id", id, "email", email)

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	emailMsg := model.EmailMessage{
		Email: user.Email,
		Value: rand.Intn(100000),
	}

	if err := s.producer.PublishToExchange(ctx,
		s.cfg.RabbitMQ.Exchange,
		s.cfg.RabbitMQ.EmailLoginRoutingKey,
		emailMsg); err != nil {
		s.logger.Error("failed to publish email message", "error", err)
	}

	return nil
}

func (s *service) UpdateUser(ctx context.Context, id uuid.UUID, userDto *model.UserUpdateDTO) error {
	logger := s.logger.With("op", "service/UpdateUser", "user_id", id)

	if userDto == nil {
		return fmt.Errorf("update user: %w", ErrInvalidInput)
	}

	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("update user: %w", ErrUserNotFound)
		}
		return fmt.Errorf("find user by id: %w", err)
	}
	if user == nil {
		return fmt.Errorf("update user: %w", ErrUserNotFound)
	}

	if userDto.UserName == nil && userDto.Email == nil {
		return fmt.Errorf("update user: %w", ErrInvalidInput)
	}

	if userDto.UserName != nil {
		userName := strings.TrimSpace(*userDto.UserName)
		if userName == "" {
			return fmt.Errorf("update user: %w", ErrInvalidInput)
		}
		user.UserName = userName
	}

	if userDto.Email != nil {
		email := strings.TrimSpace(*userDto.Email)
		if email == "" {
			return fmt.Errorf("update user: %w", ErrInvalidInput)
		}
		user.Email = email
	}

	logger.Info("user updated in service")

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	logger := s.logger.With("op", "service/DeleteUser", "user_id", id)

	if err := s.repo.DeleteUser(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("delete user: %w", ErrUserNotFound)
		}
		return fmt.Errorf("delete user: %w", err)
	}

	logger.Info("user deleted in service")

	return nil
}

func (s *service) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	logger := s.logger.With("op", "service/FindUserByID", "user_id", id)

	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by id: %w", ErrUserNotFound)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("find user by id: %w", ErrUserNotFound)
	}

	logger.Info("user found in service")

	return user, nil
}

func (s *service) LoginUser(ctx context.Context, userDTO *model.UserLoginDTO) (*model.TokenResponse, error) {
	if userDTO == nil {
		return nil, fmt.Errorf("login user: %w", ErrInvalidInput)
	}

	email := strings.TrimSpace(userDTO.Email)
	password := strings.TrimSpace(userDTO.Password)
	if email == "" || password == "" {
		return nil, fmt.Errorf("login user: %w", ErrInvalidInput)
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("login user: %w", ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("login user: %w", ErrInvalidCredentials)
	}

	if !s.verifyPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("login user: %w", ErrInvalidCredentials)
	}

	accessToken, err := s.generateJWT(user.ID, time.Minute*15)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken := uuid.New().String()

	err = s.repo.CreateSession(ctx, &model.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Hour * 24),
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return fmt.Errorf("logout user: %w", ErrInvalidInput)
	}

	if err := s.repo.DeleteSession(ctx, refreshToken); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("logout user: %w", ErrUserNotFound)
		}
		return fmt.Errorf("logout user: %w", err)
	}

	return nil
}

func (s *service) hash(password string) (string, error) {
	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}

	return string(hashPass), nil
}

func (s *service) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false
	}
	return true
}

func (s *service) generateJWT(userID uuid.UUID, ttl time.Duration) (string, error) {
	claims := model.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
		},
		UserID: userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func New(repo repository.Repository, logger *slog.Logger, jwtSecret string, producer rabbitmq.Publisher, cfg *config.Config) Service {
	return &service{
		repo:      repo,
		logger:    logger,
		jwtSecret: jwtSecret,
		producer:  producer,
		cfg:       cfg,
	}
}
