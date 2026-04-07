package server

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/EraldCaka/PIoneer"
	"github.com/EraldCaka/pi-web/internal/config"
	"github.com/EraldCaka/pi-web/internal/db"
	"github.com/EraldCaka/pi-web/internal/handlers"
	"github.com/EraldCaka/pi-web/internal/routes"
	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/EraldCaka/pi-web/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Server struct {
	app    *fiber.App
	logger *slog.Logger
	device PIoneer.Device
	devSvc *services.DeviceService
	cancel context.CancelFunc
}

func New(logger *slog.Logger) *Server {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	f, err := os.Open("config.yaml")
	if err != nil {
		logger.Error("failed to open config.yaml", "error", err)
		os.Exit(1)
	}
	defer f.Close()

	device, err := PIoneer.New(f)
	if err != nil {
		logger.Error("failed to create device", "error", err)
		os.Exit(1)
	}

	database, err := db.Connect(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	hub := ws.NewHub(logger)
	authSvc := services.NewAuthService(cfg.JWT)
	userSvc := services.NewUserService(database)
	devSvc := services.NewDeviceService(device, hub, cfg, logger)

	authH := handlers.NewAuthHandler(userSvc, authSvc, logger)
	userH := handlers.NewUserHandler(userSvc, logger)
	deviceH := handlers.NewDeviceHandler(devSvc, logger)
	wsH := handlers.NewWSHandler(hub)
	pagesH := handlers.NewPagesHandler(cfg, logger)
	fragH := handlers.NewFragmentsHandler(devSvc, userSvc, logger)

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})
	app.Use(recover.New())
	app.Use(cors.New())

	routes.Register(app, authSvc, authH, userH, deviceH, wsH, pagesH, fragH)

	_, cancel := context.WithCancel(context.Background())
	go hub.Run()

	return &Server{
		app:    app,
		logger: logger,
		device: device,
		devSvc: devSvc,
		cancel: cancel,
	}
}

func (s *Server) Start(addr string) error {
	if err := s.device.Start(); err != nil {
		s.logger.Error("failed to start device", "error", err)
	} else {
		s.logger.Info("device started")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.devSvc.Start(ctx)

	return s.app.Listen(addr)
}

func (s *Server) Shutdown(timeout time.Duration) error {
	s.cancel()
	if err := s.device.Stop(); err != nil {
		s.logger.Error("failed to stop device", "error", err)
	}
	return s.app.ShutdownWithTimeout(timeout)
}
