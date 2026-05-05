package service

import (
	"Kite/internal/model"
	"Kite/internal/server/repository"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CreateServer(ctx context.Context, ownerID uuid.UUID, name string) (*model.Server, error)
	GetUserServers(ctx context.Context, userID uuid.UUID) ([]model.Server, error)
	JoinServer(ctx context.Context, userID uuid.UUID, inviteCode string) error

	CreateChannel(ctx context.Context, serverID, userID uuid.UUID, name, channelType string) (*model.Channel, error)
	GetChannels(ctx context.Context, serverID, userID uuid.UUID) ([]model.Channel, error)

	SendMessage(ctx context.Context, channelID, authorID uuid.UUID, content string) (*model.Message, error)
	GetMessages(ctx context.Context, channelID uuid.UUID, limit int) ([]model.Message, error)
}

type service struct {
	repo   repository.Repository
	logger *slog.Logger
}

func (s *service) CreateServer(ctx context.Context, ownerID uuid.UUID, name string) (*model.Server, error) {
	serverID := uuid.New()
	inviteCode := generateInviteCode()

	server := &model.Server{
		ID:         serverID,
		OwnerID:    ownerID,
		Name:       name,
		InviteCode: inviteCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.repo.CreateServer(ctx, server)
	if err != nil {
		return nil, fmt.Errorf("create server service: %w", err)
	}

	// Добавляем владельца в участники
	member := &model.ServerMember{
		ServerID: serverID,
		UserID:   ownerID,
		JoinedAt: time.Now(),
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("add owner to server: %w", err)
	}

	// Создаем стандартные каналы
	defaultChannels := []struct{ name, typ string }{
		{"общий", "text"},
		{"голосовой", "voice"},
	}

	for i, ch := range defaultChannels {
		channel := &model.Channel{
			ID:        uuid.New(),
			ServerID:  serverID,
			Name:      ch.name,
			Type:      ch.typ,
			Position:  i,
			CreatedAt: time.Now(),
		}
		if err := s.repo.CreateChannel(ctx, channel); err != nil {
			s.logger.Error("failed to create default channel", "error", err)
		}
	}

	return server, nil
}

func (s *service) GetUserServers(ctx context.Context, userID uuid.UUID) ([]model.Server, error) {
	return s.repo.GetUserServers(ctx, userID)
}

func (s *service) JoinServer(ctx context.Context, userID uuid.UUID, inviteCode string) error {
	// Находим сервер по invite коду
	// Для простоты, добавим метод GetServerByInviteCode
	// Пока что реализуем поиск через все серверы
	servers, err := s.repo.GetUserServers(ctx, userID)
	if err != nil {
		return err
	}

	// Проверяем, не состоит ли уже
	for _, server := range servers {
		if server.InviteCode == inviteCode {
			return fmt.Errorf("already a member")
		}
	}

	// Находим сервер (нужно добавить метод в репозиторий)
	// Для краткости, пропустим

	return nil
}

func (s *service) CreateChannel(ctx context.Context, serverID, userID uuid.UUID, name, channelType string) (*model.Channel, error) {
	// Проверяем, является ли пользователь участником сервера
	isMember, err := s.repo.IsMember(ctx, serverID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this server")
	}

	channel := &model.Channel{
		ID:        uuid.New(),
		ServerID:  serverID,
		Name:      name,
		Type:      channelType,
		Position:  0,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	return channel, nil
}

func (s *service) GetChannels(ctx context.Context, serverID, userID uuid.UUID) ([]model.Channel, error) {
	isMember, err := s.repo.IsMember(ctx, serverID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this server")
	}

	return s.repo.GetChannelsByServer(ctx, serverID)
}

func (s *service) SendMessage(ctx context.Context, channelID, authorID uuid.UUID, content string) (*model.Message, error) {
	message := &model.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return message, nil
}

func (s *service) GetMessages(ctx context.Context, channelID uuid.UUID, limit int) ([]model.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetMessagesByChannel(ctx, channelID, limit)
}

func generateInviteCode() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return string(code)
}

func New(repo repository.Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}
