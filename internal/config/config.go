package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func (j JWTConfig) Expiry() time.Duration {
	d, err := time.ParseDuration(j.ExpiryStr)
	if err != nil || d == 0 {
		return 24 * time.Hour
	}
	return d
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	MQTT     MQTTConfig     `yaml:"mqtt"`
	Chip     ChipConfig     `yaml:"chip"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type JWTConfig struct {
	Secret    string `yaml:"secret"`
	ExpiryStr string `yaml:"expiry"`
}

type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client-id"`
	Topic    string `yaml:"topic"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ChipConfig struct {
	DigitalPins []DigitalPin `yaml:"digital-pins"`
	PWMPins     []PWMPin     `yaml:"pwm-pins"`
}

type DigitalPin struct {
	ID        string `yaml:"id"`
	Pin       int    `yaml:"pin"`
	Direction string `yaml:"direction"`
}

type PWMPin struct {
	ID        string  `yaml:"id"`
	Pin       int     `yaml:"pin"`
	Frequency int     `yaml:"frequency"`
	DutyCycle float64 `yaml:"duty-cycle"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Port == "" {
		cfg.Server.Port = "3000"
	}
	return &cfg, nil

}
