package handlers

import (
	"log/slog"
	"strconv"

	"github.com/EraldCaka/pi-web/internal/services"
	"github.com/gofiber/fiber/v2"
)

type DeviceHandler struct {
	device *services.DeviceService
	log    *slog.Logger
}

func NewDeviceHandler(device *services.DeviceService, log *slog.Logger) *DeviceHandler {
	return &DeviceHandler{device: device, log: log}
}

func (h *DeviceHandler) Health(c *fiber.Ctx) error {
	return c.JSON(h.device.Health())
}

func (h *DeviceHandler) Metrics(c *fiber.Ctx) error {
	return c.JSON(h.device.Metrics())
}

func (h *DeviceHandler) Info(c *fiber.Ctx) error {
	return c.JSON(h.device.Info())
}

func (h *DeviceHandler) SystemMetrics(c *fiber.Ctx) error {
	sm, err := h.device.SystemMetrics()
	if err != nil {
		h.log.Error("system metrics failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sm)
}

func (h *DeviceHandler) Sensors(c *fiber.Ctx) error {
	data, err := h.device.ReadSensors()
	if err != nil {
		h.log.Error("read sensors failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func (h *DeviceHandler) Sensor(c *fiber.Ctx) error {
	id := c.Params("id")
	data, err := h.device.ReadSensor(id)
	if err != nil {
		h.log.Error("read sensor failed", "id", id, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

type setPWMRequest struct {
	DutyCycle float64 `json:"duty_cycle" form:"duty_cycle"`
}

func (h *DeviceHandler) SetPWM(c *fiber.Ctx) error {
	pin, err := strconv.Atoi(c.Params("pin"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pin"})
	}
	var req setPWMRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.device.SetDutyCycle(pin, req.DutyCycle); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"pin": pin, "duty_cycle": req.DutyCycle})
}

type writeGPIORequest struct {
	Value int `json:"value" form:"value"`
}

func (h *DeviceHandler) WriteGPIO(c *fiber.Ctx) error {
	pin, err := strconv.Atoi(c.Params("pin"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pin"})
	}
	var req writeGPIORequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.device.WritePin(pin, req.Value); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"pin": pin, "value": req.Value})
}
