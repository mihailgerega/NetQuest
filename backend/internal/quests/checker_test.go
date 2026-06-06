package quests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/topology"
)

func TestCatalogContainsTwentyQuestsWithLearningMetadata(t *testing.T) {
	quests := Catalog()
	if len(quests) < 20 {
		t.Fatalf("expected at least 20 quests, got %d", len(quests))
	}
	counts := map[Difficulty]int{}
	v2Counts := map[Difficulty]int{}
	for _, quest := range quests {
		counts[quest.Difficulty]++
		if len(quest.ID) >= len("quest-v2-") && quest.ID[:len("quest-v2-")] == "quest-v2-" {
			v2Counts[quest.Difficulty]++
		}
		if quest.Goal == "" || len(quest.ExpectedChecks) == 0 || len(quest.InitialTopology) == 0 {
			t.Fatalf("quest %s is missing goal/checks/topology", quest.ID)
		}
		minHints := 4
		if quest.Difficulty == DifficultyHard {
			minHints = 5
		}
		if len(quest.ProgressiveHints) < minHints {
			t.Fatalf("quest %s should have at least %d progressive hints, got %d", quest.ID, minHints, len(quest.ProgressiveHints))
		}
		if quest.AfterSolution == "" {
			t.Fatalf("quest %s should have after-solution explanation", quest.ID)
		}
		if strings.Contains(quest.AfterSolution, "backend-checker") || strings.Contains(quest.AfterSolution, "canvas") {
			t.Fatalf("quest %s has mixed-language after-solution text: %q", quest.ID, quest.AfterSolution)
		}
		if len(quest.GlossaryTerms) == 0 {
			t.Fatalf("quest %s should have glossary terms", quest.ID)
		}
	}
	if counts[DifficultyEasy] < 6 || counts[DifficultyMedium] < 7 || counts[DifficultyHard] < 7 {
		t.Fatalf("unexpected total difficulty spread: %#v", counts)
	}
	if v2Counts[DifficultyEasy] != 3 || v2Counts[DifficultyMedium] != 4 || v2Counts[DifficultyHard] != 3 {
		t.Fatalf("unexpected v2 difficulty spread: %#v", v2Counts)
	}
}

func containsString(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func TestCheckerFailsBrokenInitialDNSQuest(t *testing.T) {
	quest := questByID(t, "quest-dns-lookup")
	result := checkerForTest().Check(context.Background(), quest, quest.InitialTopology, 7)
	if result.Passed {
		t.Fatalf("initial broken DNS quest should fail: %#v", result)
	}
	if result.Score == 100 || len(result.Hints) == 0 {
		t.Fatalf("expected partial score and hints, got score=%d hints=%#v", result.Score, result.Hints)
	}
}

func TestCheckerPassesFixedDNSQuest(t *testing.T) {
	quest := questByID(t, "quest-dns-lookup")
	fixed := addDNSRecord(t, quest.InitialTopology, "api.netquest.local", "10.0.2.21")
	result := checkerForTest().Check(context.Background(), quest, fixed, 7)
	if !result.Passed || result.Score != 100 {
		t.Fatalf("expected fixed DNS quest to pass, got %#v", result)
	}
	if result.AfterSolutionExplanation == "" {
		t.Fatalf("expected after-solution explanation for passed quest")
	}
}

func TestCheckerReturnsRelevantProgressiveHint(t *testing.T) {
	quest := questByID(t, "quest-v2-dns-resolver-down")
	result := checkerForTest().Check(context.Background(), quest, quest.InitialTopology, 7)
	if result.Passed {
		t.Fatalf("initial resolver-down quest should fail")
	}
	found := false
	for _, hint := range result.Hints {
		if hint == "" {
			t.Fatalf("empty hint returned")
		}
		if containsString(hint, "DNS") || containsString(hint, "resolver") || containsString(hint, "status") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DNS/status-related hint, got %#v", result.Hints)
	}
}

func TestCheckerValidatesFailoverQuestWithNewBackend(t *testing.T) {
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
	result := checkerForTest().Check(context.Background(), quest, fixed, 7)
	if !result.Passed {
		t.Fatalf("expected failover quest to pass with new server-3 backend, got %#v", result)
	}
}

func checkerForTest() Checker {
	validator := topology.NewValidator()
	return NewChecker(simulation.NewBasicEngine(validator), validator)
}

func questByID(t *testing.T, id string) Quest {
	t.Helper()
	for _, quest := range Catalog() {
		if quest.ID == id {
			return quest
		}
	}
	t.Fatalf("quest %s not found", id)
	return Quest{}
}

func addDNSRecord(t *testing.T, data json.RawMessage, name, value string) json.RawMessage {
	t.Helper()
	var doc topology.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode topology: %v", err)
	}
	for i := range doc.Nodes {
		if doc.Nodes[i].Type == topology.NodeTypeDNS {
			doc.Nodes[i].Config["records"] = []map[string]any{{"name": name, "type": "A", "value": value, "ttl": 300}}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("encode topology: %v", err)
	}
	return out
}
