package model

import (
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID          uuid.UUID `json:"id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IconURL     *string   `json:"icon_url,omitempty"`
	InviteCode  string    `json:"invite_code"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ServerMember struct {
	ServerID uuid.UUID `json:"server_id"`
	UserID   uuid.UUID `json:"user_id"`
	Nickname *string   `json:"nickname,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
}

type Channel struct {
	ID        uuid.UUID `json:"id"`
	ServerID  uuid.UUID `json:"server_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // текст или войс
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        uuid.UUID  `json:"id"`
	ChannelID uuid.UUID  `json:"channel_id"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Content   string     `json:"content"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateServerDTO struct {
	Name string `json:"name"`
}

type CreateChannelDTO struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
