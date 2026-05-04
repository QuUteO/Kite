package repository

import (
	"Kite/internal/model"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)

	CreateSession(ctx context.Context, session *model.Session) error
	DeleteSession(ctx context.Context, refreshToken string) error

	CreateVerification(ctx context.Context, verification *model.Verification) error
	FindVerificationByCode(ctx context.Context, code string, userID uuid.UUID) (*model.Verification, error)
	UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status model.VerificationStatus) error
	DeleteExpiredVerifications(ctx context.Context) error
	MarkUserVerified(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func (r *repository) CreateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("repository.CreateUser: nil user")
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, username, password, is_verified, created_at, updated_at) 
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID,
		user.Email,
		user.UserName,
		user.PasswordHash,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateUser: %w", err)
	}
	r.logger.Info("Inserted new user", "user_id", user.ID, "email", user.Email)
	return nil
}

func (r *repository) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return errors.New("repository.UpdateUser: nil user")
	}

	tag, err := r.db.Exec(ctx, `
				UPDATE users SET email = $2, username = $3, password = $4 WHERE id = $1
			`,
		user.ID,
		user.Email,
		user.UserName,
		user.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("repository.UpdateUser: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.UpdateUser: %w", pgx.ErrNoRows)
	}
	r.logger.Info("Updated user")

	return nil
}

func (r *repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
			DELETE FROM users WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("repository.DeleteUser: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.DeleteUser: %w", pgx.ErrNoRows)
	}
	r.logger.Info("deleted user")

	return nil
}

func (r *repository) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{ID: id}

	err := r.db.QueryRow(ctx,
		`SELECT email, username, password, is_verified, verified_at, created_at, updated_at 
         FROM users WHERE id = $1`,
		id,
	).Scan(
		&user.Email,
		&user.UserName,
		&user.PasswordHash,
		&user.IsVerified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository.FindUserByID: %w", err)
	}

	r.logger.Info("Found user", "id", id, "is_verified", user.IsVerified) // <--- ДОБАВЛЕНО
	return user, nil
}

func (r *repository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}

	// <--- ИСПРАВЛЕНО: Добавлены все поля
	err := r.db.QueryRow(
		ctx,
		`SELECT id, email, username, password, is_verified, verified_at, created_at, updated_at 
         FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.UserName,
		&user.PasswordHash,
		&user.IsVerified, // <--- ДОБАВЛЕНО
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository.FindUserByEmail: %w", err)
	}

	r.logger.Info("Found user", "email", email, "is_verified", user.IsVerified)
	return user, nil
}

func (r *repository) CreateSession(ctx context.Context, session *model.Session) error {
	if session == nil {
		return errors.New("repository.CreateSession: nil session")
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO sessions (user_id, refresh_token, expires_at) VALUES ($1, $2, $3)`,
		session.UserID, session.RefreshToken, session.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("repository.CreateSession: %w", err)
	}

	r.logger.Info("Created session")
	return nil
}

func (r *repository) DeleteSession(ctx context.Context, refreshToken string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM sessions WHERE refresh_token = $1`,
		refreshToken,
	)
	if err != nil {
		return fmt.Errorf("repository.DeleteSession: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.DeleteSession: %w", pgx.ErrNoRows)
	}

	r.logger.Info("Deleted session")
	return nil
}

func (r *repository) CreateVerification(ctx context.Context, verification *model.Verification) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO verifications (id, user_id, code, status, type, expires_at, created_at) 
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		verification.ID,
		verification.UserID,
		verification.Code,
		verification.Status,
		verification.Type,
		verification.ExpiresAt,
		verification.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create verification: %w", err)
	}
	return nil
}

func (r *repository) FindVerificationByCode(ctx context.Context, code string, userID uuid.UUID) (*model.Verification, error) {
	var verification model.Verification
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, code, status, type, expires_at, created_at
         FROM verifications
         WHERE code = $1 AND user_id = $2 AND status = 'pending' AND expires_at > NOW()`,
		code, userID,
	).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Code,
		&verification.Status,
		&verification.Type,
		&verification.ExpiresAt,
		&verification.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find verification by code: %w", err)
	}
	return &verification, nil
}

func (r *repository) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status model.VerificationStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE verifications SET status = $1, updated_at = NOW() WHERE id = $2`,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("update verification status: %w", err)
	}
	return nil
}

func (r *repository) MarkUserVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET is_verified = TRUE, verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("mark user verified: %w", err)
	}
	return nil
}

func (r *repository) DeleteExpiredVerifications(ctx context.Context) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM verifications WHERE expires_at < NOW() OR status = 'expired'`,
	)
	if err != nil {
		return fmt.Errorf("delete expired verifications: %w", err)
	}
	return nil
}

func New(db *pgxpool.Pool, logger *slog.Logger) Repository {
	return &repository{
		db:     db,
		logger: logger.With("op", "repository.go"),
	}
}
