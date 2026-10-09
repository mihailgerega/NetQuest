package engine

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// dnsAnswer — ответ DNS: какой IP получен, от какого резолвера и по какому пути
// до него шёл запрос.
type dnsAnswer struct {
	IP       string
	Resolver model.Node
	Path     []string
}

// runDNSLookup — сценарий dns_lookup: только разрешение имени, без соединения с целью.
func (r *runner) runDNSLookup() {
	hostname := targetHost(r.req.Scenario.Target)

	answer, ok := r.resolveDNS(hostname)
	if !ok {
		return
	}

	r.summary.ResolvedIP = answer.IP
	r.summary.Path = answer.Path

	if len(r.summary.Path) == 0 {
		r.summary.Path = []string{r.req.Scenario.SourceNodeID, answer.Resolver.ID}
	}

	r.summary.LatencyFormula = "DNS Lookup latency ≈ path RTT + DNS processing delay"
	r.summary.Decisions = append(r.summary.Decisions, "DNS resolved "+hostname+" to "+answer.IP)
	r.summary.TotalLatencyMs = r.timestamp

	r.complete("DNS lookup completed")
}

// resolveDNS разрешает hostname в IP через первый исправный DNS-узел топологии.
//
// Шаги: найти резолвер → найти до него маршрут → запрос и ответ (виртуальное
// время: путь туда + обработка + путь обратно) → найти A-запись. Записи
// сравниваются без учёта регистра; запись без type считается A.
// При любой неудаче симуляция проваливается здесь же, вызывающему — ok=false.
func (r *runner) resolveDNS(hostname string) (dnsAnswer, bool) {
	source := r.req.Scenario.SourceNodeID

	var resolver model.Node

	for _, node := range r.doc.Nodes {
		if node.Type == model.NodeTypeDNS && !nodeDown(node) {
			resolver = node
			break
		}
	}

	if resolver.ID == "" {
		r.emit(model.EventDNSError, model.EventSeverityError, "DNS resolver is not available", source, "", r.packetID, map[string]any{"hostname": hostname})
		r.fail("DNS resolver is not available")

		return dnsAnswer{}, false
	}

	path, latency, ok := r.findPath(source, resolver.ID)
	if !ok {
		r.emit(model.EventRouteNotFound, model.EventSeverityError, "route to DNS resolver not found", source, resolver.ID, r.packetID, map[string]any{
			"source":      source,
			"target":      resolver.ID,
			"explanation": r.routeInfo,
		})
		r.fail("no route from " + source + " to DNS resolver " + resolver.ID)

		return dnsAnswer{}, false
	}

	start := r.timestamp
	processing := r.processingDelay(1, 4)

	r.emit(model.EventDNSQuery, model.EventSeverityInfo, "DNS query", source, resolver.ID, r.packetID, map[string]any{
		"hostname":        hostname,
		"type":            "A",
		"path":            path,
		"oneWayLatencyMs": latency,
	})
	r.advance(latency + processing + latency)
	r.addLatencyStage("dns_lookup", "DNS Lookup", r.timestamp-start, map[string]any{
		"hostname":        hostname,
		"path":            path,
		"oneWayLatencyMs": latency,
		"processingMs":    processing,
		"rttMs":           latency * 2,
	})

	for _, record := range anySlice(resolver.Config["records"]) {
		m := anyMap(record)
		if !strings.EqualFold(stringValue(m["name"]), hostname) || !strings.EqualFold(defaultString(m["type"], "A"), "A") {
			continue
		}

		value := stringValue(m["value"])
		r.emit(model.EventDNSResponse, model.EventSeverityInfo, "DNS response", resolver.ID, source, r.packetID, map[string]any{
			"hostname":  hostname,
			"value":     value,
			"ttl":       intValue(m["ttl"], 300),
			"latencyMs": latency * 2,
		})

		return dnsAnswer{IP: value, Resolver: resolver, Path: path}, true
	}

	r.emit(model.EventDNSError, model.EventSeverityError, "DNS NXDOMAIN", resolver.ID, source, r.packetID, map[string]any{
		"hostname": hostname,
		"rcode":    "NXDOMAIN",
	})
	r.fail("DNS NXDOMAIN for " + hostname)

	return dnsAnswer{}, false
}
