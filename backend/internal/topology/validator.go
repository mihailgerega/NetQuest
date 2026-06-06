package topology

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Validator struct {
	AllowedTypes map[NodeType]struct{}
	MaxNodes     int
	MaxLinks     int
}

func NewValidator() Validator {
	allowed := map[NodeType]struct{}{
		NodeTypeClient:       {},
		NodeTypeServer:       {},
		NodeTypeRouter:       {},
		NodeTypeSwitch:       {},
		NodeTypeDNS:          {},
		NodeTypeFirewall:     {},
		NodeTypeLoadBalancer: {},
		NodeTypeProxy:        {},
		NodeTypeNATGateway:   {},
		NodeTypeVPNGateway:   {},
		NodeTypeDatabase:     {},
		NodeTypeInternet:     {},
	}
	return Validator{AllowedTypes: allowed, MaxNodes: MaxNodes, MaxLinks: MaxLinks}
}

func (v Validator) ValidateRaw(data json.RawMessage) ValidationResult {
	result := ValidationResult{Valid: true, Errors: []ValidationError{}}
	if len(bytes.TrimSpace(data)) == 0 {
		return result.add("$", "topology JSON is required")
	}

	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return result.add("$", fmt.Sprintf("invalid JSON structure: %v", err))
	}
	if decoder.Decode(&struct{}{}) == nil {
		return result.add("$", "topology JSON must contain a single object")
	}

	nodesRaw, hasNodes := raw["nodes"]
	linksRaw, hasLinks := raw["links"]
	if !hasNodes {
		result = result.add("nodes", "nodes field is required")
	}
	if !hasLinks {
		result = result.add("links", "links field is required")
	}
	if !hasNodes || !hasLinks {
		return result
	}

	var doc Document
	if err := json.Unmarshal(nodesRaw, &doc.Nodes); err != nil {
		result = result.add("nodes", "nodes must be an array")
	}
	if err := json.Unmarshal(linksRaw, &doc.Links); err != nil {
		result = result.add("links", "links must be an array")
	}
	if len(result.Errors) > 0 {
		result.Valid = false
		return result
	}

	if len(doc.Nodes) > v.limitNodes() {
		result = result.add("nodes", fmt.Sprintf("too many nodes: max %d", v.limitNodes()))
	}
	if len(doc.Links) > v.limitLinks() {
		result = result.add("links", fmt.Sprintf("too many links: max %d", v.limitLinks()))
	}

	nodeIDs := make(map[string]struct{}, len(doc.Nodes))
	nodesByID := make(map[string]Node, len(doc.Nodes))
	for i, node := range doc.Nodes {
		path := fmt.Sprintf("nodes[%d]", i)
		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			result = result.add(path+".id", "node id is required")
			continue
		}
		if _, exists := nodeIDs[nodeID]; exists {
			result = result.add(path+".id", "node id must be unique")
		}
		nodeIDs[nodeID] = struct{}{}
		nodesByID[nodeID] = node

		if _, ok := v.AllowedTypes[node.Type]; !ok {
			result = result.add(path+".type", fmt.Sprintf("unsupported node type %q", node.Type))
		}
	}

	for i, link := range doc.Links {
		path := fmt.Sprintf("links[%d]", i)
		if strings.TrimSpace(link.ID) == "" {
			result = result.add(path+".id", "link id is required")
		}
		source := strings.TrimSpace(link.SourceNodeID)
		target := strings.TrimSpace(link.TargetNodeID)
		if source == "" {
			result = result.add(path+".sourceNodeId", "source node is required")
		} else if _, ok := nodeIDs[source]; !ok {
			result = result.add(path+".sourceNodeId", "source node does not exist")
		}
		if target == "" {
			result = result.add(path+".targetNodeId", "target node is required")
		} else if _, ok := nodeIDs[target]; !ok {
			result = result.add(path+".targetNodeId", "target node does not exist")
		}
	}

	for i, node := range doc.Nodes {
		if node.Type != NodeTypeLoadBalancer {
			continue
		}
		result = validateLoadBalancerBackends(result, node, i, nodesByID)
	}

	result.Valid = len(result.Errors) == 0
	return result
}

func (r ValidationResult) add(path, message string) ValidationResult {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Path: path, Message: message})
	return r
}

func (v Validator) limitNodes() int {
	if v.MaxNodes <= 0 {
		return MaxNodes
	}
	return v.MaxNodes
}

func (v Validator) limitLinks() int {
	if v.MaxLinks <= 0 {
		return MaxLinks
	}
	return v.MaxLinks
}

func validateLoadBalancerBackends(result ValidationResult, node Node, nodeIndex int, nodesByID map[string]Node) ValidationResult {
	if node.Config == nil {
		return result
	}
	raw, exists := node.Config["backends"]
	if !exists || raw == nil {
		return result
	}
	backends, ok := raw.([]any)
	if !ok {
		return result.add(fmt.Sprintf("nodes[%d].config.backends", nodeIndex), "load balancer backends must be an array")
	}

	seen := map[string]struct{}{}
	for i, item := range backends {
		path := fmt.Sprintf("nodes[%d].config.backends[%d]", nodeIndex, i)
		backend, ok := item.(map[string]any)
		if !ok {
			result = result.add(path, "backend entry must be an object")
			continue
		}
		nodeID := strings.TrimSpace(fmt.Sprint(backend["nodeId"]))
		if nodeID == "" || nodeID == "<nil>" {
			result = result.add(path+".nodeId", "backend nodeId is required")
			continue
		}
		if _, exists := seen[nodeID]; exists {
			result = result.add(path+".nodeId", "backend nodeId must be unique")
			continue
		}
		seen[nodeID] = struct{}{}

		backendNode, exists := nodesByID[nodeID]
		if !exists {
			result = result.add(path+".nodeId", "backend node does not exist")
			continue
		}
		if backendNode.Type != NodeTypeServer {
			result = result.add(path+".nodeId", "backend node must reference a server")
		}
	}
	return result
}
