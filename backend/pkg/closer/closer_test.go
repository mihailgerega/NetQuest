package closer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCloseAllLIFO: ресурсы закрываются в обратном порядке регистрации,
// ошибка одного не останавливает остальные, наружу — первая ошибка,
// повторный CloseAll ничего не делает.
func TestCloseAllLIFO(t *testing.T) {
	t.Parallel()

	c := newCloser()

	var order []string

	errFirst := errors.New("первая")

	c.Add("postgres", func(context.Context) error {
		order = append(order, "postgres")
		return nil
	})
	c.Add("redis", func(context.Context) error {
		order = append(order, "redis")
		return errors.New("вторая")
	})
	c.Add("http", func(context.Context) error {
		order = append(order, "http")
		return errFirst
	})

	err := c.CloseAll(context.Background())

	assert.Equal(t, []string{"http", "redis", "postgres"}, order)
	assert.ErrorIs(t, err, errFirst)
	assert.NoError(t, c.CloseAll(context.Background()), "повторный вызов — no-op")
	assert.Len(t, order, 3)
}
