package topology

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidatorAcceptsValidTopology(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [
			{"id": "client-1", "type": "client"},
			{"id": "server-1", "type": "server"}
		],
		"links": [
			{"id": "link-1", "sourceNodeId": "client-1", "targetNodeId": "server-1"}
		]
	}`)

	result := NewValidator().ValidateRaw(payload)
	if !result.Valid {
		t.Fatalf("expected topology to be valid, got errors: %#v", result.Errors)
	}
}

func TestValidatorRejectsDuplicateNodeIDs(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [
			{"id": "node-1", "type": "client"},
			{"id": "node-1", "type": "server"}
		],
		"links": []
	}`)

	result := NewValidator().ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to be invalid")
	}
	if !hasError(result, "nodes[1].id", "unique") {
		t.Fatalf("expected duplicate id error, got %#v", result.Errors)
	}
}

func TestValidatorRejectsLinksToMissingNodes(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [{"id": "client-1", "type": "client"}],
		"links": [{"id": "link-1", "sourceNodeId": "client-1", "targetNodeId": "missing"}]
	}`)

	result := NewValidator().ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to be invalid")
	}
	if !hasError(result, "links[0].targetNodeId", "does not exist") {
		t.Fatalf("expected missing target node error, got %#v", result.Errors)
	}
}

func TestValidatorEnforcesLimits(t *testing.T) {
	payload := json.RawMessage(`{"nodes":[{"id":"n1","type":"router"},{"id":"n2","type":"switch"}],"links":[]}`)
	validator := NewValidator()
	validator.MaxNodes = 1

	result := validator.ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to exceed max nodes")
	}
	if !hasError(result, "nodes", "too many nodes") {
		t.Fatalf("expected node limit error, got %#v", result.Errors)
	}
}

func TestLBBackendMustReferenceExistingServer(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [
			{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "router-1"}]}},
			{"id": "router-1", "type": "router"}
		],
		"links": []
	}`)

	result := NewValidator().ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to be invalid")
	}
	if !hasError(result, "nodes[0].config.backends[0].nodeId", "must reference a server") {
		t.Fatalf("expected server type error, got %#v", result.Errors)
	}
}

func TestDuplicateLBBackendsInvalid(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [
			{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "server-1"}, {"nodeId": "server-1"}]}},
			{"id": "server-1", "type": "server"}
		],
		"links": []
	}`)

	result := NewValidator().ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to be invalid")
	}
	if !hasError(result, "nodes[0].config.backends[1].nodeId", "unique") {
		t.Fatalf("expected duplicate backend error, got %#v", result.Errors)
	}
}

func TestStaleBackendReferenceInvalid(t *testing.T) {
	payload := json.RawMessage(`{
		"nodes": [
			{"id": "lb-1", "type": "load_balancer", "config": {"backends": [{"nodeId": "deleted-server"}]}}
		],
		"links": []
	}`)

	result := NewValidator().ValidateRaw(payload)
	if result.Valid {
		t.Fatal("expected topology to be invalid")
	}
	if !hasError(result, "nodes[0].config.backends[0].nodeId", "does not exist") {
		t.Fatalf("expected stale backend error, got %#v", result.Errors)
	}
}

func hasError(result ValidationResult, path, messagePart string) bool {
	for _, err := range result.Errors {
		if err.Path == path && strings.Contains(err.Message, messagePart) {
			return true
		}
	}
	return false
}
