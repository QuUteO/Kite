package repository

import (
	"Kite/internal/model"
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateServer(ctx context.Context, server *model.Server) error
	GetServerByID(ctx context.Context, serverID uuid.UUID) (*model.Server, error)
	GetUserServers(ctx context.Context, userID uuid.UUID) ([]model.Server, error)
	AddMember(ctx context.Context, member *model.ServerMember) error
	IsMember(ctx context.Context, serverID, userID uuid.UUID) (bool, error)

	CreateChannel(ctx context.Context, channel *model.Channel) error
	GetChannelsByServer(ctx context.Context, serverID uuid.UUID) ([]model.Channel, error)

	CreateMessage(ctx context.Context, message *model.Message) error
	GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit int) ([]model.Message, error)
}

type repository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func (r *repository) CreateServer(ctx context.Context, server *model.Server) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO servers (id, owner_id, name, invite_code) 
         VALUES ($1, $2, $3, $4)`,
		server.ID, server.OwnerID, server.Name, server.InviteCode,
	)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	return nil
}

func (r *repository) GetServerByID(ctx context.Context, serverID uuid.UUID) (*model.Server, error) {
	var server model.Server

	err := r.db.QueryRow(ctx,
		`SELECT id, owner_id, name, description, icon_url, invite_code, created_at, updated_at
         FROM servers WHERE id = $1`,
		serverID,
	).Scan(
		&server.ID,
		&server.OwnerID,
		&server.Name,
		&server.Description,
		&server.IconURL,
		&server.InviteCode,
		&server.CreatedAt,
		&server.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("get server by id: %w", err)
	}

	return &server, nil
}

func (r *repository) GetUserServers(ctx context.Context, userID uuid.UUID) ([]model.Server, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.id, s.owner_id, s.name, s.description, s.icon_url, s.invite_code, s.created_at, s.updated_at
         FROM servers s
         JOIN server_members sm ON s.id = sm.server_id
         WHERE sm.user_id = $1
         ORDER BY s.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get user servers: %w", err)
	}
	defer rows.Close()

	var servers []model.Server
	for rows.Next() {

		var server model.Server

		err := rows.Scan(&server.ID, &server.OwnerID, &server.Name, &server.Description,
			&server.IconURL, &server.InviteCode, &server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan server: %w", err)
		}

		servers = append(servers, server)
	}
	return servers, nil
}

func (r *repository) AddMember(ctx context.Context, member *model.ServerMember) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO server_members (server_id, user_id, nickname) VALUES ($1, $2, $3)`,
		member.ServerID, member.UserID, member.Nickname,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *repository) IsMember(ctx context.Context, serverID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM server_members WHERE server_id = $1 AND user_id = $2)`,
		serverID, userID,
	).Scan(&exists)

	return exists, err
}

func (r *repository) CreateChannel(ctx context.Context, channel *model.Channel) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO channels (id, server_id, name, type, position) 
         VALUES ($1, $2, $3, $4, $5)`,
		channel.ID, channel.ServerID, channel.Name, channel.Type, channel.Position,
	)
	if err != nil {
		return fmt.Errorf("create channel: %w", err)
	}

	return nil
}

func (r *repository) GetChannelsByServer(ctx context.Context, serverID uuid.UUID) ([]model.Channel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, server_id, name, type, position, created_at
				 FROM channels WHERE server_id = $1 ORDER BY position ASC`,
		serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("get channels: %w", err)
	}
	defer rows.Close()

	var channels []model.Channel
	for rows.Next() {
		var channel model.Channel
		err := rows.Scan(&channel.ID, &channel.ServerID, &channel.Name,
			&channel.Type, &channel.Position, &channel.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

func (r *repository) CreateMessage(ctx context.Context, message *model.Message) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO messages (id, channel_id, author_id, content) 
         VALUES ($1, $2, $3, $4)`,
		message.ID, message.ChannelID, message.AuthorID, message.Content,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}

func (r *repository) GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit int) ([]model.Message, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, channel_id, author_id, content, edited_at, deleted_at, created_at
         FROM messages WHERE channel_id = $1 AND deleted_at IS NULL
         ORDER BY created_at DESC LIMIT $2`,
		channelID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content,
			&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func New(db *pgxpool.Pool, logger *slog.Logger) Repository {
	return &repository{
		db:     db,
		logger: logger,
	}
}
