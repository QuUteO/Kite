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
		` INSERT INTO users (id, email, username, password) 
			  VALUES ($1, $2, $3, $4)`,
		user.ID,
		user.Email,
		user.UserName,
		user.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateUser: %w", err)
	}
	r.logger.Info("Inserted new user")
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
		`SELECT email, username, password FROM users WHERE id = $1`,
		id,
	).Scan(&user.Email, &user.UserName, &user.PasswordHash)

	if err != nil {
		return nil, fmt.Errorf("repository.FindUserByID: %w", err)
	}

	r.logger.Info("Found user", "id", id)
	return user, nil
}

func (r *repository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}

	err := r.db.QueryRow(
		ctx,
		`SELECT id, email, username, password FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.UserName, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("repository.FindUserByEmail: %w", err)
	}

	r.logger.Info("Found user")
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

func New(db *pgxpool.Pool, logger *slog.Logger) Repository {
	return &repository{
		db:     db,
		logger: logger.With("op", "repository.go"),
	}
}
