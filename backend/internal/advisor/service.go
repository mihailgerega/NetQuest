package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/topology"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type TopologyStore interface {
	GetForOwner(ctx context.Context, topologyID, ownerID string) (topology.Topology, error)
}

type Service struct {
	topologies TopologyStore
	validator  topology.Validator
}

func NewService(topologies TopologyStore, validator topology.Validator) *Service {
	return &Service{topologies: topologies, validator: validator}
}

func (s *Service) AnalyzeRaw(data json.RawMessage, scenario *simulation.Scenario) (AnalyzeResponse, error) {
	if len(data) == 0 {
		return AnalyzeResponse{}, apperrors.Validation("topology is required", nil)
	}
	issues := []Issue{}
	validation := s.validator.ValidateRaw(data)
	if !validation.Valid {
		for _, item := range validation.Errors {
			issues = append(issues, Issue{
				Severity:     SeverityError,
				Category:     "Topology",
				Code:         "TOPOLOGY_INVALID",
				Title:        "Topology не проходит validation",
				Message:      item.Path + ": " + item.Message,
				SuggestedFix: "Исправьте структуру topology перед запуском simulation.",
			})
		}
	}
	var doc topology.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return AnalyzeResponse{}, apperrors.Validation("invalid topology JSON", err.Error())
	}
	issues = append(issues, analyzeDNS(doc)...)
	issues = append(issues, analyzeLoadBalancers(doc)...)
	issues = append(issues, analyzeFirewalls(doc)...)
	issues = append(issues, analyzeLatency(doc)...)
	issues = append(issues, analyzeRouting(doc, scenario)...)
	issues = append(issues, analyzeSource(doc, scenario)...)
	return AnalyzeResponse{Issues: issues}, nil
}

func (s *Service) AnalyzeStored(ctx context.Context, userID, topologyID string, scenario *simulation.Scenario) (AnalyzeResponse, error) {
	record, err := s.topologies.GetForOwner(ctx, topologyID, userID)
	if err != nil {
		return AnalyzeResponse{}, err
	}
	return s.AnalyzeRaw(record.Data, scenario)
}

func analyzeDNS(doc topology.Document) []Issue {
	hasDNS := false
	hasAPIRecord := false
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeDNS {
			continue
		}
		hasDNS = true
		for _, record := range anySlice(node.Config["records"]) {
			m := anyMap(record)
			if strings.EqualFold(stringValue(m["name"]), "api.netquest.local") && strings.EqualFold(defaultString(m["type"], "A"), "A") {
				hasAPIRecord = true
			}
		}
	}
	if !hasDNS || hasAPIRecord {
		return nil
	}
	return []Issue{{
		Severity:     SeverityError,
		Category:     "DNS",
		Code:         "DNS_RECORD_MISSING",
		Title:        "DNS record отсутствует",
		Message:      "Домен api.netquest.local не найден в DNS records.",
		SuggestedFix: "Добавьте A record api.netquest.local → IP Load Balancer или Server.",
	}}
}

func analyzeLoadBalancers(doc topology.Document) []Issue {
	issues := []Issue{}
	nodes := map[string]topology.Node{}
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeLoadBalancer {
			continue
		}
		backends := anySlice(node.Config["backends"])
		if len(backends) == 0 {
			issues = append(issues, Issue{
				Severity:       SeverityError,
				Category:       "Load Balancer",
				Code:           "LB_BACKEND_POOL_EMPTY",
				Title:          "Load Balancer backend pool пустой",
				Message:        "Load Balancer не сможет выбрать backend для HTTPS request.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Добавьте хотя бы один healthy Server в backend pool.",
			})
			continue
		}
		for _, item := range backends {
			backendID := stringValue(anyMap(item)["nodeId"])
			backend, ok := nodes[backendID]
			if !ok {
				issues = append(issues, Issue{Severity: SeverityError, Category: "Load Balancer", Code: "LB_STALE_BACKEND", Title: "Backend ссылается на удалённый node", Message: fmt.Sprintf("Load Balancer содержит backend %s, но такого node больше нет.", backendID), AffectedNodeID: node.ID, SuggestedFix: "Удалите stale backend из pool или добавьте node обратно."})
				continue
			}
			if backend.Type != topology.NodeTypeServer {
				issues = append(issues, Issue{Severity: SeverityError, Category: "Load Balancer", Code: "LB_BACKEND_NOT_SERVER", Title: "Backend не является Server", Message: backendID + " не является Server node.", AffectedNodeID: node.ID, SuggestedFix: "Оставьте в backend pool только Server nodes."})
			}
		}
	}
	return issues
}

func analyzeFirewalls(doc topology.Document) []Issue {
	issues := []Issue{}
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeFirewall {
			continue
		}
		allowsHTTPS := false
		for _, raw := range anySlice(node.Config["rules"]) {
			rule := anyMap(raw)
			if strings.EqualFold(defaultString(rule["protocol"], "tcp"), "tcp") &&
				intValue(rule["port"], 0) == 443 &&
				strings.EqualFold(defaultString(rule["action"], "deny"), "allow") {
				allowsHTTPS = true
				break
			}
		}
		if !allowsHTTPS && strings.EqualFold(defaultString(node.Config["defaultPolicy"], "deny"), "deny") {
			issues = append(issues, Issue{
				Severity:       SeverityWarning,
				Category:       "Firewall",
				Code:           "FIREWALL_BLOCKS_HTTPS",
				Title:          "Firewall блокирует tcp/443",
				Message:        "Текущее правило Firewall может запрещать HTTPS traffic.",
				AffectedNodeID: node.ID,
				SuggestedFix:   "Добавьте allow rule для tcp/443 выше deny rule.",
			})
		}
	}
	return issues
}

func analyzeLatency(doc topology.Document) []Issue {
	issues := []Issue{}
	for _, link := range doc.Links {
		latency := intValue(link.Config["latencyMs"], 0)
		if link.Status != "down" && latency >= 500 {
			issues = append(issues, Issue{
				Severity:       SeverityWarning,
				Category:       "Latency",
				Code:           "HIGH_LATENCY_LINK",
				Title:          "Высокая latency на link",
				Message:        fmt.Sprintf("Link %s → %s имеет latency %dms и может влиять на totalLatencyMs.", link.SourceNodeID, link.TargetNodeID, latency),
				AffectedLinkID: link.ID,
				SuggestedFix:   "Уменьшите latency или выберите другой route.",
			})
		}
	}
	return issues
}

func analyzeRouting(doc topology.Document, scenario *simulation.Scenario) []Issue {
	issues := []Issue{}
	if scenario != nil && scenario.SourceNodeID != "" && scenario.Target != "" {
		targetID := scenario.Target
		if strings.HasPrefix(targetID, "http") {
			targetID = ""
		}
		if targetID != "" && !graphReachable(doc, scenario.SourceNodeID, targetID) {
			issues = append(issues, Issue{Severity: SeverityError, Category: "Routing", Code: "ROUTE_MISSING", Title: "Маршрут не найден", Message: "Между source и target нет active path.", AffectedNodeID: scenario.SourceNodeID, SuggestedFix: "Проверьте links, status nodes и routing table."})
		}
	}
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeClient {
			continue
		}
		cidr := stringValue(node.Config["cidr"])
		if cidr != "" && stringValue(node.Config["defaultGateway"]) == "" {
			issues = append(issues, Issue{Severity: SeverityInfo, Category: "Routing", Code: "DEFAULT_GATEWAY_MISSING", Title: "Default gateway не задан", Message: node.ID + " имеет CIDR, но не имеет defaultGateway.", AffectedNodeID: node.ID, SuggestedFix: "Укажите defaultGateway для доступа в другие subnet."})
		}
	}
	return issues
}

func analyzeSource(doc topology.Document, scenario *simulation.Scenario) []Issue {
	if scenario == nil || scenario.SourceNodeID == "" {
		return nil
	}
	for _, node := range doc.Nodes {
		if node.ID == scenario.SourceNodeID && node.Status == "down" {
			return []Issue{{Severity: SeverityError, Category: "Topology", Code: "SOURCE_CLIENT_DOWN", Title: "Source Client down", Message: "Выбранный Client недоступен.", AffectedNodeID: node.ID, SuggestedFix: "Восстановите Client или выберите другой source."}}
		}
	}
	return nil
}

func graphReachable(doc topology.Document, sourceID, targetID string) bool {
	if sourceID == targetID {
		return true
	}
	nodes := map[string]topology.Node{}
	for _, node := range doc.Nodes {
		nodes[node.ID] = node
	}
	queue := []string{sourceID}
	visited := map[string]bool{sourceID: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, link := range doc.Links {
			if link.Status == "down" {
				continue
			}
			next := ""
			if link.SourceNodeID == current {
				next = link.TargetNodeID
			} else if link.TargetNodeID == current {
				next = link.SourceNodeID
			}
			if next == "" || visited[next] || nodes[next].Status == "down" {
				continue
			}
			if next == targetID {
				return true
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}
	return false
}

func anyMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func anySlice(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func defaultString(value any, fallback string) string {
	if s := stringValue(value); s != "" {
		return s
	}
	return fallback
}

func intValue(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}

func sameCIDR(cidr, ipValue string) bool {
	ip := net.ParseIP(ipValue)
	if ip == nil || !strings.Contains(cidr, "/") {
		return false
	}
	_, network, err := net.ParseCIDR(cidr)
	return err == nil && network.Contains(ip)
}
