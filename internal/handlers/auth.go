package handlers

import (
	"log/slog"
	"time"

	"github.com/EraldCaka/pi-web/internal/middleware"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	users *services.UserService
	auth  *services.AuthService
	log   *slog.Logger
}

func NewAuthHandler(users *services.UserService, auth *services.AuthService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{users: users, auth: auth, log: log}
}

type registerRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

type loginRequest struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

func isHTMX(c *fiber.Ctx) bool {
	return c.Get("HX-Request") == "true"
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return h.authError(c, "Invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return h.authError(c, "Email and password are required")
	}

	user, err := h.users.Register(req.Email, req.Password)
	if err != nil {
		h.log.Error("register failed", "error", err)
		return h.authError(c, "Email already in use")
	}

	token, err := h.auth.Sign(user)
	if err != nil {
		return h.authError(c, "Token generation failed")
	}

	h.setTokenCookie(c, token)

	if isHTMX(c) {
		c.Set("HX-Redirect", "/dashboard")
		return c.Status(fiber.StatusOK).SendString("")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"token": token, "user": user})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return h.authError(c, "Invalid request body")
	}

	user, err := h.users.Authenticate(req.Email, req.Password)
	if err != nil {
		return h.authError(c, "Invalid email or password")
	}

	token, err := h.auth.Sign(user)
	if err != nil {
		return h.authError(c, "Token generation failed")
	}

	h.setTokenCookie(c, token)

	if isHTMX(c) {
		c.Set("HX-Redirect", "/dashboard")
		return c.Status(fiber.StatusOK).SendString("")
	}
	return c.JSON(fiber.Map{"token": token, "user": user})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	claims := middleware.GetClaims(c)
	return c.JSON(fiber.Map{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}

func (h *AuthHandler) setTokenCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})
}

func (h *AuthHandler) authError(c *fiber.Ctx, msg string) error {
	if isHTMX(c) {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(fiber.StatusOK).SendString(
			`<div class="alert alert-error">` + msg + `</div>`,
		)
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}
