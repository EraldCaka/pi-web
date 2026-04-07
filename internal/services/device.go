package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/EraldCaka/PIoneer"
	pioneerConfig "github.com/EraldCaka/PIoneer/pkg/config"
	"github.com/EraldCaka/pi-web/internal/config"
	"github.com/EraldCaka/pi-web/internal/ws"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const sensorPollInterval = 5 * time.Second

type DeviceService struct {
	device PIoneer.Device
	hub    *ws.Hub
	cfg    *config.Config
	logger *slog.Logger
	mqttC  mqtt.Client
}

func NewDeviceService(device PIoneer.Device, hub *ws.Hub, cfg *config.Config, logger *slog.Logger) *DeviceService {
	return &DeviceService{device: device, hub: hub, cfg: cfg, logger: logger}
}

func (s *DeviceService) Start(ctx context.Context) {
	go s.pollSensors(ctx)
	go s.watchGPIOPins(ctx)
	go s.connectMQTT(ctx)
}

func (s *DeviceService) Health() pioneerConfig.HealthStatus {
	return s.device.Health()
}

func (s *DeviceService) Metrics() pioneerConfig.DeviceMetrics {
	return s.device.Metrics()
}

func (s *DeviceService) ReadSensor(id string) (map[string]any, error) {
	return s.device.ReadSensor(id)
}

func (s *DeviceService) ReadSensors() (map[string]map[string]any, error) {
	return s.device.ReadSensors()
}

func (s *DeviceService) Info() pioneerConfig.DeviceInfo {
	return s.device.Info()
}

func (s *DeviceService) SystemMetrics() (pioneerConfig.SystemMetrics, error) {
	return s.device.SystemMetrics()
}

func (s *DeviceService) SetDutyCycle(pin int, duty float64) error {
	return s.device.SetDutyCycle(pin, duty)
}

func (s *DeviceService) WritePin(pin, value int) error {
	return s.device.Write(pin, value)
}

func (s *DeviceService) pollSensors(ctx context.Context) {
	ticker := time.NewTicker(sensorPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := s.device.ReadSensors()
			if err != nil {
				s.logger.Warn("sensor poll failed", "error", err)
				continue
			}
			msg := ws.NewMessage(ws.TypeSensorData, s.cfg.MQTT.ClientID, data)
			s.hub.Broadcast(msg)
		}
	}
}

func (s *DeviceService) watchGPIOPins(ctx context.Context) {
	for _, pin := range s.cfg.Chip.DigitalPins {
		if pin.Direction != "input" {
			continue
		}
		ch, err := s.device.Watch(pin.Pin)
		if err != nil {
			s.logger.Error("watch pin failed", "pin", pin.Pin, "error", err)
			continue
		}
		go func(id string, pinNum int, events <-chan pioneerConfig.PinEvent) {
			for {
				select {
				case <-ctx.Done():
					s.device.StopWatch(pinNum)
					return
				case evt, ok := <-events:
					if !ok {
						return
					}
					payload := map[string]any{
						"id":        id,
						"pin":       evt.Pin,
						"old_value": evt.OldValue,
						"new_value": evt.NewValue,
					}
					msg := ws.NewMessage(ws.TypeGPIOEvent, s.cfg.MQTT.ClientID, payload)
					s.hub.Broadcast(msg)
				}
			}
		}(pin.ID, pin.Pin, ch)
	}
}

func (s *DeviceService) connectMQTT(ctx context.Context) {
	mqttCfg := s.cfg.MQTT
	if mqttCfg.Broker == "" {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttCfg.Broker)
	// distinct client ID so we don't conflict with pioneer's own client
	opts.SetClientID(mqttCfg.ClientID + "-server-sub")
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	if mqttCfg.Username != "" {
		opts.SetUsername(mqttCfg.Username)
		opts.SetPassword(mqttCfg.Password)
	}

	s.mqttC = mqtt.NewClient(opts)
	if tok := s.mqttC.Connect(); tok.Wait() && tok.Error() != nil {
		s.logger.Error("MQTT subscriber connect failed", "error", tok.Error())
		return
	}
	s.logger.Info("MQTT subscriber connected", "broker", mqttCfg.Broker)

	topic := mqttCfg.Topic + "/#"
	s.mqttC.Subscribe(topic, 1, s.onMQTTMessage)

	<-ctx.Done()
	s.mqttC.Disconnect(500)
}

func (s *DeviceService) onMQTTMessage(_ mqtt.Client, msg mqtt.Message) {
	var parsed any
	if err := json.Unmarshal(msg.Payload(), &parsed); err != nil {
		parsed = string(msg.Payload())
	}

	payload := map[string]any{
		"topic":   msg.Topic(),
		"payload": parsed,
	}
	out := ws.NewMessage(ws.TypeMQTTMessage, s.cfg.MQTT.ClientID, payload)
	s.hub.Broadcast(out)
}
