package handler

import (
	"Kite/internal/model"
	"Kite/internal/user/service"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Handler struct {
	srv       service.Service
	logger    *slog.Logger
	jwtSecret string
}

type contextKey string

const userIDContextKey contextKey = "userID"

func New(srv service.Service, logger *slog.Logger, jwtSecret string) *Handler {
	return &Handler{
		srv:       srv,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrUserNotFound):
		status = http.StatusNotFound
	}
	h.writeError(w, status, err)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var userDTO model.UserCreateDTO

	if err := decodeJSON(r, &userDTO); err != nil {
		h.logger.Warn("unable to decode request body", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	// отправляем данные в сервис с контекстом
	if err := h.srv.CreateUser(r.Context(), &userDTO); err != nil {
		h.logger.Warn("unable to create user", "error", err)
		h.writeServiceError(w, err)
		return
	}

	h.logger.Info("user created")
	h.writeJSON(w, http.StatusCreated, map[string]string{
		"status": http.StatusText(http.StatusCreated),
		"email":  userDTO.Email,
	})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uuidParse, err := uuid.Parse(id)
	if err != nil {
		h.logger.Warn("unable to parse id", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	var userDTO model.UserUpdateDTO
	if err := decodeJSON(r, &userDTO); err != nil {
		h.logger.Warn("unable to decode request body", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.srv.UpdateUser(r.Context(), uuidParse, &userDTO); err != nil {
		h.logger.Warn("unable to update user", "error", err)
		h.writeServiceError(w, err)
		return
	}

	h.logger.Info("user updated")
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": http.StatusText(http.StatusOK),
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uuidParse, err := uuid.Parse(id)
	if err != nil {
		h.logger.Warn("unable to parse id", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.srv.DeleteUser(r.Context(), uuidParse); err != nil {
		h.logger.Warn("unable to delete user", "error", err)
		h.writeServiceError(w, err)
		return
	}

	h.logger.Info("user deleted")
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": http.StatusText(http.StatusOK),
	})
}

func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uuidParse, err := uuid.Parse(id)
	if err != nil {
		h.logger.Warn("unable to parse id", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.srv.FindUserByID(r.Context(), uuidParse)
	if err != nil {
		h.logger.Warn("unable to find user", "error", err)
		h.writeServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, user)
}

func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var userDTO model.UserLoginDTO
	if err := decodeJSON(r, &userDTO); err != nil {
		h.logger.Warn("unable to decode request body", "error", err)
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.srv.LoginUser(r.Context(), &userDTO)
	if err != nil {
		h.logger.Warn("unable to login", "error", err)
		h.writeServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":        http.StatusText(http.StatusOK),
		"access_token":  user.AccessToken,
		"refresh_token": user.RefreshToken,
	})
}

func (h *Handler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.srv.Logout(r.Context(), req.RefreshToken); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := &model.UserClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(h.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("unable to encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": http.StatusText(status),
		"error":  err.Error(),
	})
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Публичные
	r.Post("/register", h.CreateUser)
	r.Post("/login", h.LoginUser)

	// Приватные (нужен токен)
	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Post("/logout", h.LogoutUser)
		r.Put("/{id}", h.UpdateUser)
		r.Delete("/{id}", h.DeleteUser)
		r.Get("/{id}", h.GetUserByID)
	})

	return r
}
