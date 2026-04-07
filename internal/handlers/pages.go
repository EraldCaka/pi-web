package handlers

import (
	"html/template"
	"log/slog"

	"github.com/EraldCaka/pi-web/internal/config"
	"github.com/EraldCaka/pi-web/internal/middleware"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/EraldCaka/pi-web/web"
	"github.com/gofiber/fiber/v2"
)

// PagesHandler renders full HTML pages for the HTMX frontend.
type PagesHandler struct {
	tmpl *template.Template
	cfg  *config.Config
	log  *slog.Logger
}

func NewPagesHandler(cfg *config.Config, log *slog.Logger) *PagesHandler {
	tmpl := template.Must(template.ParseFS(web.FS, "views/*.html"))
	return &PagesHandler{tmpl: tmpl, cfg: cfg, log: log}
}

func (h *PagesHandler) render(c *fiber.Ctx, name string, data any) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(c, name, data)
}

// Login serves the login/register page.
func (h *PagesHandler) Login(c *fiber.Ctx) error {
	// Already logged in → redirect to dashboard.
	if c.Cookies("token") != "" {
		return c.Redirect("/dashboard")
	}
	return h.render(c, "login.html", nil)
}

// Dashboard serves the main IoT dashboard.
func (h *PagesHandler) Dashboard(c *fiber.Ctx) error {
	claims := middleware.GetClaims(c)
	return h.render(c, "dashboard.html", map[string]any{
		"Email":       claims.Email,
		"Role":        string(claims.Role),
		"Token":       c.Cookies("token"),
		"PWMPins":     h.cfg.Chip.PWMPins,
		"DigitalPins": h.cfg.Chip.DigitalPins,
	})
}

// Admin serves the admin user management page.
func (h *PagesHandler) Admin(c *fiber.Ctx) error {
	claims := middleware.GetClaims(c)
	return h.render(c, "admin.html", map[string]any{
		"Email": claims.Email,
		"Role":  string(claims.Role),
	})
}

// Logout clears the auth cookie and redirects to login.
func (h *PagesHandler) Logout(c *fiber.Ctx) error {
	c.ClearCookie("token")
	return c.Redirect("/")
}

// PagesMW is an auth middleware that redirects to login instead of returning 401.
func PagesMW(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("token")
		if token == "" {
			return c.Redirect("/")
		}
		claims, err := authSvc.Verify(token)
		if err != nil {
			c.ClearCookie("token")
			return c.Redirect("/")
		}
		c.Locals("claims", claims)
		return c.Next()
	}
}

// PagesAdminMW is like PagesMW but also requires the admin role.
func PagesAdminMW(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("token")
		if token == "" {
			return c.Redirect("/")
		}
		claims, err := authSvc.Verify(token)
		if err != nil {
			c.ClearCookie("token")
			return c.Redirect("/")
		}
		if string(claims.Role) != "admin" {
			return c.Redirect("/dashboard")
		}
		c.Locals("claims", claims)
		return c.Next()
	}
}
