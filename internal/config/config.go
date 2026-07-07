package config

import (
	"errors"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPPort           string        `mapstructure:"HTTP_PORT"`
	LogLevel           string        `mapstructure:"LOG_LEVEL"`
	RabbitMQURL        string        `mapstructure:"RABBITMQ_URL"`
	RabbitMQExchange   string        `mapstructure:"RABBITMQ_EXCHANGE"`
	RabbitMQRoutingKey string        `mapstructure:"RABBITMQ_ROUTING_KEY"`
	WebhookSecret      string        `mapstructure:"WEBHOOK_SECRET"`
	RequestTimeout     time.Duration `mapstructure:"REQUEST_TIMEOUT"`
	MaxRequestSize     int64         `mapstructure:"MAX_REQUEST_SIZE"`
	RabbitMQMaxRetries int           `mapstructure:"RABMQ_MAX_RETRIES"`
	RabbitMQRetryDelay time.Duration `mapstructure:"RABMQ_RETRY_DELAY"`
}

func Load() (*Config, error) {
	viper.SetDefault("HTTP_PORT", "3000")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("RABBITMQ_EXCHANGE", "your_exchange_name")
	viper.SetDefault("RABBITMQ_ROUTING_KEY", "your_routing_key")
	viper.SetDefault("REQUEST_TIMEOUT", 10*time.Second)
	viper.SetDefault("MAX_REQUEST_SIZE", 1048576) // 1MB default
	viper.SetDefault("RABMQ_MAX_RETRIES", 5)
	viper.SetDefault("RABMQ_RETRY_DELAY", 500*time.Millisecond)

	// Explicitly bind required env vars so Unmarshal picks them up
	_ = viper.BindEnv("RABBITMQ_URL")
	_ = viper.BindEnv("WEBHOOK_SECRET")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.RabbitMQURL == "" {
		return nil, errors.New("RABBITMQ_URL environment variable is required")
	}

	if cfg.WebhookSecret == "" {
		return nil, errors.New("WEBHOOK_SECRET environment variable is required for security")
	}

	return &cfg, nil
}
