package tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/domain/catalog"
	"github.com/netquest/netquest/backend/internal/service/domain/checker"
	"github.com/netquest/netquest/backend/internal/service/domain/engine"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
)

// Тесты чекера на настоящих квестах каталога, движке и валидаторе:
// проверяется, что решение засчитывается по поведению сети.

const testSeed = 7

// TestCheckDNSQuest: исходная (сломанная) топология квеста не проходит —
// с частичным счётом и подсказками; с правильной A-записью — проходит на 100.
func TestCheckDNSQuest(t *testing.T) {
	t.Parallel()

	quest := questByID(t, "quest-dns-lookup")

	broken := newChecker().Check(context.Background(), quest, quest.InitialTopology, testSeed)
	assert.False(t, broken.Passed)
	assert.Less(t, broken.Score, 100)
	assert.NotEmpty(t, broken.Hints)

	fixed := newChecker().Check(context.Background(), quest, withDNSRecord(t, quest.InitialTopology, "api.netquest.local", "10.0.2.21"), testSeed)
	assert.True(t, fixed.Passed, "%#v", fixed)
	assert.Equal(t, 100, fixed.Score)
	assert.NotEmpty(t, fixed.AfterSolutionExplanation)
}

// TestCheckReturnsRelevantHint: к проваленной проверке подбирается подсказка про её слой.
func TestCheckReturnsRelevantHint(t *testing.T) {
	t.Parallel()

	quest := questByID(t, "quest-v2-dns-resolver-down")

	result := newChecker().Check(context.Background(), quest, quest.InitialTopology, testSeed)
	require.False(t, result.Passed)

	found := false
	for _, hint := range result.Hints {
		require.NotEmpty(t, hint)

		lower := strings.ToLower(hint)
		if strings.Contains(lower, "dns") || strings.Contains(lower, "resolver") || strings.Contains(lower, "status") {
			found = true
		}
	}

	assert.True(t, found, "нет подсказки про DNS: %#v", result.Hints)
}

// TestCheckFailoverWithNewBackend: решение засчитывается, если вместо выключенного
// server-1 Load Balancer выбирает новый server-3 и явно пропускает server-1.
func TestCheckFailoverWithNewBackend(t *testing.T) {
	t.Parallel()

	quest := questByID(t, "quest-backend-failover")
	fixed := json.RawMessage(`{
		"nodes":[
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
			{"id":"router-1","type":"router","config":{}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"allow"}},
			{"id":"lb-1","type":"load_balancer","config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","enabled":true},{"nodeId":"server-3","enabled":true}]}},
			{"id":"server-1","type":"server","status":"down","config":{"ip":"10.0.2.21","port":443}},
			{"id":"server-3","type":"server","config":{"ip":"10.0.2.23","port":443}}
		],
		"links":[
			{"id":"l1","sourceNodeId":"client-1","targetNodeId":"router-1","config":{"latencyMs":5}},
			{"id":"l2","sourceNodeId":"client-1","targetNodeId":"dns-1","config":{"latencyMs":2}},
			{"id":"l3","sourceNodeId":"router-1","targetNodeId":"firewall-1","config":{"latencyMs":8}},
			{"id":"l4","sourceNodeId":"firewall-1","targetNodeId":"lb-1","config":{"latencyMs":12}},
			{"id":"l5","sourceNodeId":"lb-1","targetNodeId":"server-1","config":{"latencyMs":4}},
			{"id":"l7","sourceNodeId":"lb-1","targetNodeId":"server-3","config":{"latencyMs":6}}
		]
	}`)

	result := newChecker().Check(context.Background(), quest, fixed, testSeed)

	assert.True(t, result.Passed, "%#v", result)
}

// TestCheckInvalidTopology: невалидная топология проваливает все проверки
// с первой ошибкой валидации в тексте.
func TestCheckInvalidTopology(t *testing.T) {
	t.Parallel()

	quest := questByID(t, "quest-dns-lookup")

	result := newChecker().Check(context.Background(), quest, json.RawMessage(`{"nodes":[]}`), testSeed)

	require.Len(t, result.Checks, len(quest.ExpectedChecks))
	for _, check := range result.Checks {
		assert.False(t, check.Passed)
		assert.Equal(t, "Topology не проходит validation: links field is required", check.Message)
	}

	assert.Equal(t, 0, result.Score)
	assert.NotEmpty(t, result.Hints)
}

func newChecker() *checker.Checker {
	v := validator.New()
	return checker.New(engine.New(v), v)
}

func questByID(t *testing.T, id string) model.Quest {
	t.Helper()

	for _, quest := range catalog.Catalog() {
		if quest.ID == id {
			return quest
		}
	}

	t.Fatalf("квест %s не найден", id)

	return model.Quest{}
}

// withDNSRecord заменяет записи всех DNS-узлов одной A-записью.
func withDNSRecord(t *testing.T, data json.RawMessage, name, value string) json.RawMessage {
	t.Helper()

	var doc model.Document
	require.NoError(t, json.Unmarshal(data, &doc))

	for i := range doc.Nodes {
		if doc.Nodes[i].Type == model.NodeTypeDNS {
			doc.Nodes[i].Config["records"] = []map[string]any{{"name": name, "type": "A", "value": value, "ttl": 300}}
		}
	}

	out, err := json.Marshal(doc)
	require.NoError(t, err)

	return out
}
