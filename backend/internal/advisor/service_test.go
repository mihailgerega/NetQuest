package advisor

import (
	"encoding/json"
	"testing"

	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/topology"
)

func TestAnalyzeRawReportsMissingDNSRecord(t *testing.T) {
	res := analyzeForTest(t, `{
		"nodes":[
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"dns-1","type":"dns","config":{"ip":"10.0.1.53","records":[]}}
		],
		"links":[{"id":"l1","sourceNodeId":"client-1","targetNodeId":"dns-1"}]
	}`, nil)
	if !hasIssue(res, "DNS_RECORD_MISSING") {
		t.Fatalf("expected DNS_RECORD_MISSING issue, got %#v", res.Issues)
	}
}

func TestAnalyzeRawReportsLoadBalancerPoolProblems(t *testing.T) {
	res := analyzeForTest(t, `{
		"nodes":[
			{"id":"lb-empty","type":"load_balancer","config":{"backends":[]}},
			{"id":"lb-stale","type":"load_balancer","config":{"backends":[{"nodeId":"server-deleted"}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links":[]
	}`, nil)
	if !hasIssue(res, "LB_BACKEND_POOL_EMPTY") || !hasIssue(res, "LB_STALE_BACKEND") {
		t.Fatalf("expected LB pool and stale backend issues, got %#v", res.Issues)
	}
}

func TestAnalyzeRawReportsFirewallAndLatency(t *testing.T) {
	res := analyzeForTest(t, `{
		"nodes":[
			{"id":"client-1","type":"client","config":{"ip":"10.0.1.10"}},
			{"id":"firewall-1","type":"firewall","config":{"defaultPolicy":"deny","rules":[{"priority":100,"action":"deny","protocol":"tcp","port":443}]}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links":[
			{"id":"slow","sourceNodeId":"client-1","targetNodeId":"firewall-1","config":{"latencyMs":900}},
			{"id":"l2","sourceNodeId":"firewall-1","targetNodeId":"server-1","config":{"latencyMs":10}}
		]
	}`, nil)
	if !hasIssue(res, "FIREWALL_BLOCKS_HTTPS") || !hasIssue(res, "HIGH_LATENCY_LINK") {
		t.Fatalf("expected firewall and high latency issues, got %#v", res.Issues)
	}
}

func TestAnalyzeRawReportsRoutingAndSourceIssues(t *testing.T) {
	scenario := &simulation.Scenario{Type: "icmp_ping", SourceNodeID: "client-1", Target: "server-1"}
	res := analyzeForTest(t, `{
		"nodes":[
			{"id":"client-1","type":"client","status":"down","config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
			{"id":"server-1","type":"server","config":{"ip":"10.0.2.21"}}
		],
		"links":[]
	}`, scenario)
	if !hasIssue(res, "ROUTE_MISSING") || !hasIssue(res, "DEFAULT_GATEWAY_MISSING") || !hasIssue(res, "SOURCE_CLIENT_DOWN") {
		t.Fatalf("expected route, gateway and source issues, got %#v", res.Issues)
	}
}

func analyzeForTest(t *testing.T, raw string, scenario *simulation.Scenario) AnalyzeResponse {
	t.Helper()
	service := NewService(nil, topology.NewValidator())
	res, err := service.AnalyzeRaw(json.RawMessage(raw), scenario)
	if err != nil {
		t.Fatalf("AnalyzeRaw returned error: %v", err)
	}
	return res
}

func hasIssue(res AnalyzeResponse, code string) bool {
	for _, issue := range res.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
