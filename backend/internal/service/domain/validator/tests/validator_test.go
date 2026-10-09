package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
)

// TestValidateRaw: валидная топология проходит; каждая ошибка документа
// попадает в отчёт со своим путём (nodes[1].id, links[0].targetNodeId,
// nodes[0].config.backends[0].nodeId...) и понятным сообщением.
func TestValidateRaw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		topology string
		// wantErrors — ожидаемые проблемы: путь → часть сообщения. Пусто — топология валидна.
		wantErrors map[string]string
	}{
		{
			name: "валидная топология",
			topology: `{
				"nodes": [{"id": "client-1", "type": "client"}, {"id": "server-1", "type": "server"}],
				"links": [{"id": "link-1", "sourceNodeId": "client-1", "targetNodeId": "server-1"}]
			}`,
		},
		{
			name:       "пустой документ",
			topology:   `   `,
			wantErrors: map[string]string{"$": "topology JSON is required"},
		},
		{
			name:       "два объекта подряд",
			topology:   `{"nodes":[],"links":[]} {}`,
			wantErrors: map[string]string{"$": "single object"},
		},
		{
			name:       "нет полей nodes и links",
			topology:   `{}`,
			wantErrors: map[string]string{"nodes": "nodes field is required", "links": "links field is required"},
		},
		{
			name:       "nodes не массив",
			topology:   `{"nodes": {}, "links": []}`,
			wantErrors: map[string]string{"nodes": "nodes must be an array"},
		},
		{
			name: "повтор ID узла",
			topology: `{
				"nodes": [{"id": "node-1", "type": "client"}, {"id": "node-1", "type": "server"}],
				"links": []
			}`,
			wantErrors: map[string]string{"nodes[1].id": "unique"},
		},
		{
			name:       "неизвестный тип узла",
			topology:   `{"nodes": [{"id": "n1", "type": "mainframe"}], "links": []}`,
			wantErrors: map[string]string{"nodes[0].type": `unsupported node type "mainframe"`},
		},
		{
			name: "канал на несуществующий узел",
			topology: `{
				"nodes": [{"id": "client-1", "type": "client"}],
				"links": [{"id": "link-1", "sourceNodeId": "client-1", "targetNodeId": "missing"}]
			}`,
			wantErrors: map[string]string{"links[0].targetNodeId": "does not exist"},
		},
		{
			name: "бэкенд Load Balancer — не server",
			topology: `{
				"nodes": [
					{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "router-1"}]}},
					{"id": "router-1", "type": "router"}
				],
				"links": []
			}`,
			wantErrors: map[string]string{"nodes[0].config.backends[0].nodeId": "must reference a server"},
		},
		{
			name: "повтор бэкенда в пуле",
			topology: `{
				"nodes": [
					{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "server-1"}, {"nodeId": "server-1"}]}},
					{"id": "server-1", "type": "server"}
				],
				"links": []
			}`,
			wantErrors: map[string]string{"nodes[0].config.backends[1].nodeId": "unique"},
		},
		{
			name: "устаревшая ссылка на удалённый сервер",
			topology: `{
				"nodes": [{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "deleted-server"}]}}],
				"links": []
			}`,
			wantErrors: map[string]string{"nodes[0].config.backends[0].nodeId": "does not exist"},
		},
		{
			name: "кривые открытые порты сервера",
			topology: `{
				"nodes": [{"id": "server-1", "type": "server", "config": {"openPorts": [{"protocol": "icmp", "port": 443}, {"protocol": "tcp", "port": 70000}]}}],
				"links": []
			}`,
			wantErrors: map[string]string{
				"nodes[0].config.openPorts[0].protocol": "tcp or udp",
				"nodes[0].config.openPorts[1].port":     "between 1 and 65535",
			},
		},
		{
			name: "повтор открытого порта",
			topology: `{
				"nodes": [{"id": "server-1", "type": "server", "config": {"openPorts": [{"protocol": "tcp", "port": 443}, 443]}}],
				"links": []
			}`,
			wantErrors: map[string]string{"nodes[0].config.openPorts[1]": "unique"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := validator.New().ValidateRaw(json.RawMessage(tc.topology))

			if len(tc.wantErrors) == 0 {
				assert.True(t, result.Valid, "ошибки: %#v", result.Errors)
				assert.NotNil(t, result.Errors, "у валидной топологии errors — [], а не null")
				return
			}

			require.False(t, result.Valid)
			for path, messagePart := range tc.wantErrors {
				assert.True(t, hasError(result, path, messagePart), "нет ошибки %s ~ %q в %#v", path, messagePart, result.Errors)
			}
		})
	}
}

// TestValidateRawLimits: лимит узлов настраивается полем валидатора.
func TestValidateRawLimits(t *testing.T) {
	t.Parallel()

	v := validator.New()
	v.MaxNodes = 1

	result := v.ValidateRaw(json.RawMessage(`{"nodes":[{"id":"n1","type":"router"},{"id":"n2","type":"switch"}],"links":[]}`))

	require.False(t, result.Valid)
	assert.True(t, hasError(result, "nodes", "too many nodes: max 1"), "ошибки: %#v", result.Errors)
}

// hasError ищет проблему по точному пути и части сообщения.
func hasError(result model.ValidationResult, path, messagePart string) bool {
	for _, err := range result.Errors {
		if err.Path == path && strings.Contains(err.Message, messagePart) {
			return true
		}
	}

	return false
}
