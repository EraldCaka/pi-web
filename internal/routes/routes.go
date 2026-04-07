package routes

import (
	"github.com/EraldCaka/pi-web/internal/handlers"
	"github.com/EraldCaka/pi-web/internal/middleware"
	"github.com/EraldCaka/pi-web/internal/models"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/EraldCaka/pi-web/web"
	fiberws "github.com/gofiber/websocket/v2"

	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func Register(
	app *fiber.App,
	authSvc *services.AuthService,
	authH *handlers.AuthHandler,
	userH *handlers.UserHandler,
	deviceH *handlers.DeviceHandler,
	wsH *handlers.WSHandler,
	pagesH *handlers.PagesHandler,
	fragH *handlers.FragmentsHandler,
) {
	jwtMW := middleware.JWT(authSvc)
	adminMW := middleware.RequireRole(models.RoleAdmin)
	pagesMW := handlers.PagesMW(authSvc)
	pagesAdminMW := handlers.PagesAdminMW(authSvc)

	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(web.FS),
		PathPrefix: "static",
		Browse:     false,
	}))

	app.Get("/health", userH.Health)

	app.Get("/", pagesH.Login)
	app.Get("/logout", pagesH.Logout)
	app.Get("/dashboard", pagesMW, pagesH.Dashboard)
	app.Get("/admin", pagesAdminMW, pagesH.Admin)

	auth := app.Group("/auth")
	auth.Post("/register", authH.Register)
	auth.Post("/login", authH.Login)
	auth.Get("/me", jwtMW, authH.Me)

	frags := app.Group("/fragments", jwtMW)
	frags.Get("/health", fragH.Health)
	frags.Get("/metrics", fragH.Metrics)
	frags.Get("/sensors", fragH.Sensors)
	frags.Get("/users", jwtMW, adminMW, fragH.Users)

	device := app.Group("/device", jwtMW)
	device.Get("/health", deviceH.Health)
	device.Get("/metrics", deviceH.Metrics)
	device.Get("/info", deviceH.Info)
	device.Get("/system-metrics", deviceH.SystemMetrics)
	device.Get("/sensors", deviceH.Sensors)
	device.Get("/sensors/:id", deviceH.Sensor)
	device.Post("/gpio/:pin", deviceH.WriteGPIO)
	device.Post("/pwm/:pin", deviceH.SetPWM)

	users := app.Group("/users", jwtMW, adminMW)
	users.Get("/", userH.List)
	users.Delete("/:id", userH.Delete)
	users.Post("/:id/promote", userH.Promote)

	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", jwtMW, fiberws.New(wsH.Handle))
}
