package configs

import "time"

type RabbitMQConfig struct {
	URL            string        `env:"RABBITMQ_URL"`
	MaxRetries     int           `env:"RABBITMQ_MAX_RETRIES"`
	RetryDelay     time.Duration `env:"RABBITMQ_RETRY_DELAY"`
	PrefetchCount  int           `env:"RABBITMQ_PREFETCH_COUNT"`
	ReconnectDelay time.Duration `env:"RABBITMQ_RECONNECT_DELAY"`
}
