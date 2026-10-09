package simulationevent

import "context"

// Publisher — то, что producer'у нужно от клиента NATS. Реализация —
// *nats.Client из pkg/nats; узкий интерфейс показывает, чем producer пользуется.
type Publisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}
