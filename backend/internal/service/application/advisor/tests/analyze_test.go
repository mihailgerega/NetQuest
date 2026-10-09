package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	advisorService "github.com/netquest/netquest/backend/internal/service/application/advisor"
	"github.com/netquest/netquest/backend/internal/service/application/advisor/mocks"
	"github.com/netquest/netquest/backend/internal/service/domain/validator"
)

// TestAnalyzeRaw: каждое правило советника находит свою проблему в топологии.
func TestAnalyzeRaw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		topology  string
		scenario  *model.Scenario
		wantCodes []string
	}{
		{
			name: "нет DNS-записи api.netquest.local",
			topology: `{
				"nodes":[
					{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
					{"id":"dns-1","type":"dns","config":{"ip":"10.0.1.53","records":[]}}
				],
				"links":[{"id":"l1","sourceNodeId":"client-1","targetNodeId":"dns-1"}]
			}`,
			wantCodes: []string{"DNS_RECORD_MISSING"},
		},
		{
			name: "пустой пул и устаревший бэкенд Load Balancer",
			topology: `{
				"nodes":[
					{"id":"lb-empty","type":"load_balancer","config":{"backends":[]}},
					{"id":"lb-stale","type":"load_balancer","config":{"backends":[{"nodeId":"server-deleted"}]}},
					{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
				],
				"links":[]
			}`,
			wantCodes: []string{"LB_BACKEND_POOL_EMPTY", "LB_STALE_BACKEND"},
		},
		{
			name: "firewall закрывает HTTPS и медленный канал",
			topology: `{
				"nodes":[
					{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
					{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"deny","rules":[{"priority":100,"action":"deny","protocol":"tcp","port":443}]}},
					{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
				],
				"links":[
					{"id":"slow","sourceNodeId":"client-1","targetNodeId":"firewall-1","config":{"latencyMs":900}},
					{"id":"l2","sourceNodeId":"firewall-1","targetNodeId":"server-1","config":{"latencyMs":10}}
				]
			}`,
			wantCodes: []string{"FIREWALL_BLOCKS_HTTPS", "HIGH_LATENCY_LINK"},
		},
		{
			name:     "нет маршрута, нет шлюза, источник выключен",
			scenario: &model.Scenario{Type: model.ScenarioICMPPing, SourceNodeID: "client-1", Target: "server-1"},
			topology: `{
				"nodes":[
					{"id":"client-1","type":"client","status":"down","config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
					{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
				],
				"links":[]
			}`,
			wantCodes: []string{"ROUTE_MISSING", "DEFAULT_GATEWAY_MISSING", "SOURCE_CLIENT_DOWN"},
		},
		{
			name:      "невалидная топология — замечание на каждую ошибку",
			topology:  `{"nodes":[{"id":"a","type":"mainframe"}],"links":[]}`,
			wantCodes: []string{"TOPOLOGY_INVALID"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := advisorService.New(mocks.NewTopologyReader(t), validator.New())

			issues, err := svc.AnalyzeRaw(json.RawMessage(tc.topology), tc.scenario)
			require.NoError(t, err)

			for _, code := range tc.wantCodes {
				assert.True(t, hasIssue(issues, code), "нет замечания %s в %#v", code, issues)
			}
		})
	}
}

// TestAnalyzeRawErrors: без топологии и с не-объектом вместо топологии — 422.
func TestAnalyzeRawErrors(t *testing.T) {
	t.Parallel()

	svc := advisorService.New(mocks.NewTopologyReader(t), validator.New())

	_, err := svc.AnalyzeRaw(nil, nil)
	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "topology is required", validationErr.Message)

	_, err = svc.AnalyzeRaw(json.RawMessage(`[]`), nil)
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "invalid topology JSON", validationErr.Message)
}

// TestAnalyzeStored: сохранённая версия читается с проверкой владельца;
// чужая версия — ErrTopologyNotFound без изменений.
func TestAnalyzeStored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	reader := mocks.NewTopologyReader(t)
	svc := advisorService.New(reader, validator.New())

	reader.EXPECT().GetForOwner(ctx, "topology-1", "user-1").
		Return(model.Topology{Data: json.RawMessage(`{"nodes":[{"id":"lb","type":"load_balancer","config":{"backends":[]}}],"links":[]}`)}, nil).Once()
	reader.EXPECT().GetForOwner(ctx, "topology-2", "user-1").
		Return(model.Topology{}, errs.ErrTopologyNotFound).Once()

	issues, err := svc.AnalyzeStored(ctx, "user-1", "topology-1", nil)
	require.NoError(t, err)
	assert.True(t, hasIssue(issues, "LB_BACKEND_POOL_EMPTY"))

	_, err = svc.AnalyzeStored(ctx, "user-1", "topology-2", nil)
	assert.True(t, errors.Is(err, errs.ErrTopologyNotFound))
}

func hasIssue(issues []model.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}

	return false
}
