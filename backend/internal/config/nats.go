package config

import "time"

// natsConfig — NATS для публикации событий симуляции.
type natsConfig struct {
	URL string // nats://host:port
	// Timeout — срок одной операции с брокером, если у ctx нет своего дедлайна.
	Timeout time.Duration
}

func loadNATSConfig(lookup LookupFunc) natsConfig {
	return natsConfig{
		URL:     getString(lookup, "NATS_URL", "nats://localhost:4222"),
		Timeout: getDuration(lookup, "NATS_TIMEOUT", 2*time.Second),
	}
}
