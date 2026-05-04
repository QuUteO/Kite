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

	VerifyUser(ctx context.Context, verifyDTO *model.VerificationDTO) error
	ResendVerificationCode(ctx context.Context, email string) error
	IsUserVerified(ctx context.Context, userID uuid.UUID) (bool, error)
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

func New(repo repository.Repository, logger *slog.Logger, jwtSecret string, producer rabbitmq.Publisher, cfg *config.Config) Service {
	return &service{
		repo:      repo,
		logger:    logger,
		jwtSecret: jwtSecret,
		producer:  producer,
		cfg:       cfg,
	}
}

func (s *service) CreateUser(ctx context.Context, userDto *model.UserCreateDTO) error {
	logger := s.logger.With("op", "CreateUser")

	if userDto == nil {
		return fmt.Errorf("create user: %w", ErrInvalidInput)
	}

	userName := strings.TrimSpace(userDto.UserName)
	email := strings.TrimSpace(userDto.Email)
	password := strings.TrimSpace(userDto.Password)

	if userName == "" || email == "" || password == "" {
		return fmt.Errorf("create user: %w", ErrInvalidInput)
	}

	existingUser, err := s.repo.FindUserByEmail(ctx, email)
	if existingUser != nil {
		return fmt.Errorf("user with %s already exists", email)
	}

	id := uuid.New()

	hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:           id,
		UserName:     userName,
		Email:        email,
		PasswordHash: string(hashPass),
		IsVerified:   false, // По умолчанию false
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user in repo: %w", err)
	}

	// Генерируем код верификации
	code := generateVerificationCode()

	// Сохраняем код в БД
	verification := &model.Verification{
		ID:        uuid.New(),
		UserID:    user.ID,
		Code:      code,
		Status:    model.StatusPending,
		Type:      "email_verification",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		s.logger.Error("failed to save verification code", "error", err)
	}

	// Отправляем код на email
	go func() {
		emailMsg := model.EmailMessage{
			Email: user.Email,
			Value: parseInt(code),
		}

		if err := s.producer.PublishToExchange(context.Background(),
			s.cfg.RabbitMQ.Exchange,
			s.cfg.RabbitMQ.RegisterRoutingKey,
			emailMsg); err != nil {
			s.logger.Error("failed to send verification email", "error", err)
		}
	}()

	logger.Info("user created, verification code sent", "user_id", id)
	return nil
}

func (s *service) UpdateUser(ctx context.Context, id uuid.UUID, userDto *model.UserUpdateDTO) error {
	logger := s.logger.With("op", "UpdateUser", "user_id", id)

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

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("update user in repo: %w", err)
	}

	logger.Info("user updated")
	return nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	logger := s.logger.With("op", "DeleteUser", "user_id", id)

	if err := s.repo.DeleteUser(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("delete user: %w", ErrUserNotFound)
		}
		return fmt.Errorf("delete user in repo: %w", err)
	}

	logger.Info("user deleted")
	return nil
}

func (s *service) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	logger := s.logger.With("op", "FindUserByID", "user_id", id)

	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by id: %w", ErrUserNotFound)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	logger.Info("user found")
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("login user: %w", ErrInvalidCredentials)
	}

	accessToken, err := s.generateJWT(user.ID, s.cfg.JWT.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken := uuid.New().String()

	err = s.repo.CreateSession(ctx, &model.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.cfg.JWT.RefreshTTL),
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

func (s *service) VerifyUser(ctx context.Context, verifyDTO *model.VerificationDTO) error {
	logger := s.logger.With("op", "VerifyUser", "email", verifyDTO.Email)

	user, err := s.repo.FindUserByEmail(ctx, verifyDTO.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find user by email: %w", ErrUserNotFound)
		}
		return fmt.Errorf("find user by email: %w", err)
	}

	if user.IsVerified {
		return errors.New("user already verified")
	}

	verification, err := s.repo.FindVerificationByCode(ctx, verifyDTO.Code, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find verification by code: %w", err)
		}
		return fmt.Errorf("find verification by code: %w", err)
	}

	if verification == nil {
		return fmt.Errorf("invalid or expired verification code")
	}

	if err := s.repo.UpdateVerificationStatus(ctx, verification.ID, model.StatusVerified); err != nil {
		return fmt.Errorf("update verification status: %w", err)
	}

	// Отмечаем пользователя как верифицированного
	if err := s.repo.MarkUserVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}

	logger.Info("user verified successfully", "user_id", user.ID)
	return nil
}

func (s *service) ResendVerificationCode(ctx context.Context, email string) error {
	logger := s.logger.With("op", "ResendVerificationCode")

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("find user by email: %w", ErrUserNotFound)
		}
		return fmt.Errorf("find user by email: %w", err)
	}

	if user.IsVerified {
		return errors.New("user already verified")
	}

	code := generateVerificationCode()

	verification := &model.Verification{
		ID:        uuid.New(),
		UserID:    user.ID,
		Code:      code,
		Status:    model.StatusPending,
		Type:      "email_verification",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		return fmt.Errorf("create verification: %w", err)
	}

	go func() {
		emailMsg := model.EmailMessage{
			Email: user.Email,
			Value: parseInt(code),
		}
		if err := s.producer.PublishToExchange(context.Background(),
			s.cfg.RabbitMQ.Exchange,
			s.cfg.RabbitMQ.RegisterRoutingKey,
			emailMsg,
		); err != nil {
			logger.Error("failed to publish email message", "err", err)
		} else {
			logger.Info("email message published", "email", emailMsg)
		}
	}()

	logger.Info("verification code resent")
	return nil
}

func (s *service) IsUserVerified(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("find user: %w", err)
	}
	return user.IsVerified, nil
}

func generateVerificationCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func (s *service) generateJWT(userID uuid.UUID, ttl time.Duration) (string, error) {
	claims := model.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
