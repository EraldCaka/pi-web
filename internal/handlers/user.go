package handlers

import (
	"log/slog"

	"github.com/EraldCaka/pi-web/internal/models"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
)

// UserHandler exposes user management endpoints (admin only).
type UserHandler struct {
	users *services.UserService
	log   *slog.Logger
}

func NewUserHandler(users *services.UserService, log *slog.Logger) *UserHandler {
	return &UserHandler{users: users, log: log}
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	users, err := h.users.List()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(users)
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.users.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *UserHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Promote changes a user's role to admin (admin only).
func (h *UserHandler) Promote(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	user, err := h.users.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	user.Role = models.RoleAdmin
	// UserService doesn't expose Update yet – keep it simple with a direct save via
	// a method we add inline so we don't over-engineer.
	if err := h.users.SetRole(id, models.RoleAdmin); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(user)
}
