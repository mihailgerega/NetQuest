package config

import "time"

// Значения по умолчанию для HTTP-сервера.
const (
	defaultHTTPAddr          = ":8080"
	defaultReadTimeout       = 5 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultRequestTimeout    = 5 * time.Second
	defaultShutdownTimeout   = 15 * time.Second
	defaultMaxBodyBytes      = 1 << 20 // 1 МиБ
	defaultJSONBodyLimit     = 1 << 20 // 1 МиБ
	defaultCORSAllowedOrigin = "http://localhost:3000"
)

// httpConfig — HTTP-сервер и ограничения на запросы.
type httpConfig struct {
	// Addr — адрес для net.Listen: ":8080" слушает на всех интерфейсах.
	Addr string
	// ReadTimeout, WriteTimeout, IdleTimeout — таймауты http.Server. Они защищают
	// от медленных клиентов (Slowloris), которые держат соединение, не досылая запрос.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// RequestTimeout — срок context.Context каждого запроса (middleware.RequestTimeout):
	// по нему отменяются запросы в PostgreSQL, если ответ считается слишком долго.
	RequestTimeout time.Duration
	// ShutdownTimeout — сколько ждать текущие запросы и закрытие ресурсов при остановке.
	ShutdownTimeout time.Duration
	// MaxRequestBodyBytes — предел тела любого запроса (middleware.BodyLimit).
	MaxRequestBodyBytes int64
	// JSONBodyLimitBytes — предел, до которого читается JSON-тело (httpx.DecodeJSON).
	JSONBodyLimitBytes int64
	// CORSAllowedOrigins — с каких origin браузеру разрешено ходить в API.
	CORSAllowedOrigins []string
}

func loadHTTPConfig(lookup LookupFunc) httpConfig {
	return httpConfig{
		Addr:                getString(lookup, "HTTP_ADDR", defaultHTTPAddr),
		ReadTimeout:         getDuration(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout),
		WriteTimeout:        getDuration(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout),
		IdleTimeout:         getDuration(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout),
		RequestTimeout:      getDuration(lookup, "REQUEST_TIMEOUT", defaultRequestTimeout),
		ShutdownTimeout:     getDuration(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		MaxRequestBodyBytes: getInt64(lookup, "MAX_REQUEST_BODY_BYTES", defaultMaxBodyBytes),
		JSONBodyLimitBytes:  getInt64(lookup, "JSON_BODY_LIMIT_BYTES", defaultJSONBodyLimit),
		CORSAllowedOrigins:  getCSV(lookup, "CORS_ALLOWED_ORIGINS", []string{defaultCORSAllowedOrigin}),
	}
}
