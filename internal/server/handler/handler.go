package handler

import (
	"Kite/internal/model"
	"Kite/internal/server/service"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	srv    service.Service
	logger *slog.Logger
}

type contextKey string

const userIDKey contextKey = "userID"

func New(srv service.Service, logger *slog.Logger) *Handler {
	return &Handler{
		srv:    srv,
		logger: logger,
	}
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из контекста (устанавливается middleware)
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req model.CreateServerDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	server, err := h.srv.CreateServer(r.Context(), userID, req.Name)
	if err != nil {
		h.logger.Error("failed to create server", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, server)
}

func (h *Handler) GetMyServers(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	servers, err := h.srv.GetUserServers(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get servers", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusOK, servers)
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	serverID := chi.URLParam(r, "server_id")
	if serverID == "" {
		h.writeError(w, http.StatusBadRequest, errors.New("server_id required"))
		return
	}

	var req model.CreateChannelDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	channel, err := h.srv.CreateChannel(r.Context(), uuid.MustParse(serverID), userID, req.Name, req.Type)
	if err != nil {
		h.logger.Error("failed to create channel", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, channel)
}

func (h *Handler) GetChannels(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	serverID := chi.URLParam(r, "server_id")
	if serverID == "" {
		h.writeError(w, http.StatusBadRequest, errors.New("server_id required"))
		return
	}

	channels, err := h.srv.GetChannels(r.Context(), uuid.MustParse(serverID), userID)
	if err != nil {
		h.logger.Error("failed to get channels", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusOK, channels)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	channelID := chi.URLParam(r, "channel_id")
	if channelID == "" {
		h.writeError(w, http.StatusBadRequest, errors.New("channel_id required"))
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	message, err := h.srv.SendMessage(r.Context(), uuid.MustParse(channelID), userID, req.Content)
	if err != nil {
		h.logger.Error("failed to send message", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, message)
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	channelID := chi.URLParam(r, "channel_id")
	if channelID == "" {
		h.writeError(w, http.StatusBadRequest, errors.New("channel_id required"))
		return
	}

	messages, err := h.srv.GetMessages(r.Context(), uuid.MustParse(channelID), userID, 50)
	if err != nil {
		h.logger.Error("failed to get messages", "error", err)
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}

	h.writeJSON(w, http.StatusOK, messages)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Все маршруты сервера
	r.Post("/", h.CreateServer)
	r.Get("/", h.GetMyServers)
	r.Post("/{server_id}/channels", h.CreateChannel)
	r.Get("/{server_id}/channels", h.GetChannels)
	r.Post("/channels/{channel_id}/messages", h.SendMessage)
	r.Get("/channels/{channel_id}/messages", h.GetMessages)

	return r
}
