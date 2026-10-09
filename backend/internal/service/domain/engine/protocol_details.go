package engine

import "github.com/netquest/netquest/backend/internal/model"

// protocolDetailsBuilder собирает протокольный разбор по слоям, проходя по
// событиям в порядке их появления. Более позднее событие слоя дополняет
// (mergeProtocolMap) или заменяет данные более раннего — как описано у каждого слоя.
type protocolDetailsBuilder struct {
	details   model.ProtocolDetails
	tcpEvents []string
	tlsEvents []string
}

// buildProtocolDetails собирает разбор из уже записанных событий и сводки.
func (r *runner) buildProtocolDetails() model.ProtocolDetails {
	b := protocolDetailsBuilder{
		details: model.ProtocolDetails{
			Summary: map[string]any{
				"sourceNodeId":          r.summary.SourceNodeID,
				"destination":           r.summary.Destination,
				"status":                r.summary.Status,
				"totalLatencyMs":        r.summary.TotalLatencyMs,
				"path":                  r.summary.Path,
				"selectedBackendNodeId": r.summary.SelectedBackendNodeID,
				"selectedBackendName":   r.summary.SelectedBackendName,
			},
			Errors: []map[string]any{},
		},
		tcpEvents: []string{},
		tlsEvents: []string{},
	}

	for _, event := range r.events {
		b.add(event)
	}

	return b.finish()
}

// add раскладывает одно событие по слою.
func (b *protocolDetailsBuilder) add(event model.Event) {
	switch event.Type {
	case model.EventDNSQuery, model.EventDNSResponse, model.EventDNSError:
		b.addDNS(event)
	case model.EventRouteSelected, model.EventRouteNotFound:
		b.addRouting(event)
	case model.EventFirewallDecision, model.EventFirewallDenied:
		b.addFirewall(event)
	case model.EventTCPHandshakeStart, model.EventTCPSYN, model.EventTCPSYNACK, model.EventTCPACK, model.EventTCPHandshakeDone:
		b.tcpEvents = append(b.tcpEvents, string(event.Type))
	case model.EventTLSHandshakeStart, model.EventTLSClientHello, model.EventTLSServerHello, model.EventTLSCertValidated, model.EventTLSHandshakeDone:
		b.addTLS(event)
	case model.EventServerPortOpen, model.EventServerPortClosed:
		b.addServer(event)
	case model.EventLBBackendSelected, model.EventLBBackendUnhealthy:
		b.addLoadBalancer(event)
	case model.EventSimulationFailed:
		b.addError(event)
	}
}

// addDNS: запрос, ответ и ошибка DNS дополняют одну карту слоя.
func (b *protocolDetailsBuilder) addDNS(event model.Event) {
	switch event.Type {
	case model.EventDNSQuery:
		b.details.DNS = mergeProtocolMap(b.details.DNS, map[string]any{
			"hostname":   event.Details["hostname"],
			"resolver":   event.TargetNodeID,
			"queryEvent": event.Type,
			"path":       event.Details["path"],
		})
	case model.EventDNSResponse:
		b.details.DNS = mergeProtocolMap(b.details.DNS, map[string]any{
			"responseEvent": event.Type,
			"resolvedIp":    event.Details["value"],
			"ttl":           event.Details["ttl"],
			"recordType":    "A",
		})
	default:
		b.details.DNS = mergeProtocolMap(b.details.DNS, map[string]any{"error": event.Message, "details": event.Details})
	}
}

// addRouting: выбранный маршрут заменяет данные слоя, ошибка поиска — дополняет.
func (b *protocolDetailsBuilder) addRouting(event model.Event) {
	if event.Type == model.EventRouteSelected {
		b.details.Routing = map[string]any{
			"sourceNodeId": event.SourceNodeID,
			"targetNodeId": event.TargetNodeID,
			"path":         event.Details["path"],
			"latencyMs":    event.Details["latencyMs"],
			"algorithm":    defaultString(event.Details["algorithm"], "graph_path"),
			"explanation":  defaultString(event.Details["explanation"], "Selected lowest-latency active path."),
		}

		return
	}

	b.details.Routing = mergeProtocolMap(b.details.Routing, map[string]any{
		"error":       event.Message,
		"explanation": defaultString(event.Details["explanation"], "No active route connects source and target."),
		"details":     event.Details,
	})
}

// addFirewall: решение firewall заменяет данные слоя. Протокол и порт
// фиксированы — firewall проверяется только в HTTPS-сценариях.
func (b *protocolDetailsBuilder) addFirewall(event model.Event) {
	b.details.Firewall = map[string]any{
		"nodeId":      event.TargetNodeID,
		"decision":    event.Details["decision"],
		"allowed":     event.Details["allowed"],
		"protocol":    httpsProtocol,
		"port":        httpsPort,
		"explanation": event.Message,
	}
}

// addTLS копит шаги рукопожатия; из ClientHello берёт имя сервера.
func (b *protocolDetailsBuilder) addTLS(event model.Event) {
	b.tlsEvents = append(b.tlsEvents, string(event.Type))

	if event.Type == model.EventTLSClientHello {
		b.details.TLS = mergeProtocolMap(b.details.TLS, map[string]any{"hostname": event.Details["serverName"]})
	}
}

// addServer: проверка порта сервера заменяет данные слоя.
func (b *protocolDetailsBuilder) addServer(event model.Event) {
	b.details.Server = map[string]any{
		"nodeId":          event.TargetNodeID,
		"nodeName":        event.Details["nodeName"],
		"protocol":        event.Details["protocol"],
		"port":            event.Details["port"],
		"open":            event.Details["open"],
		"implicitOpen":    event.Details["implicitOpen"],
		"openPorts":       event.Details["openPorts"],
		"matchedOpenPort": event.Details["matchedOpenPort"],
		"event":           event.Type,
		"message":         event.Message,
	}
}

// addLoadBalancer: выбор сервера заменяет данные слоя, пропуски — дополняют.
func (b *protocolDetailsBuilder) addLoadBalancer(event model.Event) {
	if event.Type == model.EventLBBackendSelected {
		b.details.LoadBalancer = map[string]any{
			"algorithm":             event.Details["algorithm"],
			"selectedBackendNodeId": event.Details["selectedBackendNodeId"],
			"selectedBackendName":   event.Details["selectedBackendName"],
			"reason":                event.Details["reason"],
			"healthyBackends":       event.Details["healthyBackends"],
			"skippedBackends":       event.Details["skippedBackends"],
		}

		return
	}

	b.details.LoadBalancer = mergeProtocolMap(b.details.LoadBalancer, map[string]any{
		"unhealthyEvent":  event.Message,
		"healthyBackends": event.Details["healthyBackends"],
		"skippedBackends": event.Details["skippedBackends"],
	})
}

// addError переводит провал симуляции в понятную пользователю ошибку
// с советом, что исправить.
func (b *protocolDetailsBuilder) addError(event model.Event) {
	code := defaultString(event.Details["code"], codeSimulationFailed)

	b.details.Errors = append(b.details.Errors, map[string]any{
		"code":             event.Details["code"],
		"userMessage":      errorUserMessage(code),
		"technicalMessage": event.Details["error"],
		"suggestedFix":     errorSuggestedFix(code),
	})
}

// finish дописывает сводные данные TCP и TLS по накопленным шагам.
func (b *protocolDetailsBuilder) finish() model.ProtocolDetails {
	if len(b.tcpEvents) > 0 {
		b.details.TCP = mergeProtocolMap(b.details.TCP, map[string]any{
			"events":      b.tcpEvents,
			"explanation": "TCP handshake uses SYN, SYN-ACK and ACK over the selected route.",
		})
	}

	if len(b.tlsEvents) > 0 {
		b.details.TLS = mergeProtocolMap(b.details.TLS, map[string]any{
			"events":      b.tlsEvents,
			"validation":  "simulated",
			"explanation": "TLS is virtual; NetQuest does not perform real cryptography.",
		})
	}

	return b.details
}

// mergeProtocolMap дописывает ключи updates в current (создаёт карту, если её нет)
// и возвращает её. Одинаковые ключи перезаписываются.
func mergeProtocolMap(current, updates map[string]any) map[string]any {
	if current == nil {
		current = map[string]any{}
	}

	for key, value := range updates {
		current[key] = value
	}

	return current
}
