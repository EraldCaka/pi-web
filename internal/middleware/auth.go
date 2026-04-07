package middleware

import (
	"strings"

	"github.com/EraldCaka/pi-web/internal/models"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/gofiber/fiber/v2"
)

const claimsKey = "claims"

func JWT(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			token = c.Cookies("token")
		}
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		claims, err := authSvc.Verify(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

func RequireRole(allowed ...models.Role) fiber.Handler {
	set := make(map[models.Role]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(claimsKey).(*services.Claims)
		if !ok || !set[claims.Role] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		return c.Next()
	}
}

func GetClaims(c *fiber.Ctx) *services.Claims {
	claims, _ := c.Locals(claimsKey).(*services.Claims)
	return claims
}
