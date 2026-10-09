// Package catalog — каталог квестов NetQuest: упражнения с намеренно
// сломанной топологией, целью и проверками решения.
//
// Каталог живёт в коде, а не в базе: квест — это вместе и текст, и топология,
// и проверки, и их удобнее ревьюить и версионировать как код. При обращении
// к квестам service/application/quest синхронизирует каталог в таблицу quests
// (UPSERT), чтобы попытки ссылались на существующие строки.
//
// Каждый квест описан своей функцией (quests_core.go, quests_v2.go). Перед
// отдачей каталог проходит «обогащение» (enrich.go): тексты приводятся
// к единой русской терминологии, недостающие прогрессивные подсказки,
// глоссарий и объяснения достраиваются по категории квеста.
package catalog

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// Цель большинства HTTPS-квестов.
const (
	apiHostname = "api.netquest.local"
	apiURL      = "https://api.netquest.local/users"
)

// Catalog возвращает все квесты в порядке показа: сначала базовые, затем
// квесты второй волны.
//
// Каждый вызов собирает каталог заново, со свежими срезами: вызывающий
// может менять результат, не задевая других.
func Catalog() []model.Quest {
	quests := []model.Quest{
		dnsLookupQuest(),
		allowPingQuest(),
		firewallHTTPSQuest(),
		lbBackendPoolQuest(),
		defaultRouteQuest(),
		latencyDiagnosticsQuest(),
		backendFailoverQuest(),
		denyDirectServerQuest(),
		backupRouteQuest(),
		productionAPIDiagnosticsQuest(),
		v2ClientSourceQuest(),
		v2DNSResolverDownQuest(),
		v2PacketLossQuest(),
		v2LBAddFallbackQuest(),
		v2DefaultGatewayQuest(),
		v2StaleLBBackendQuest(),
		v2LatencyThresholdQuest(),
		v2BackupRouteHardQuest(),
		v2SecureLBBoundaryQuest(),
		v2ProductionMultiIssueQuest(),
	}

	return enrichQuestCatalog(quests)
}

// rawTopology помечает JSON-строку как готовый документ топологии.
func rawTopology(input string) json.RawMessage {
	return json.RawMessage(input)
}

// lbTopology — общая топология Load Balancer-квестов: client → router →
// firewall → Load Balancer → три сервера, плюс прямой канал firewall → server-1
// (для квестов про запрет прямого доступа). Параметры подставляют поломку
// конкретного квеста: пул серверов, статус server-1, действие первого правила
// firewall и значение DNS-записи.
func lbTopology(backends, server1Status, firewallAction, dnsValue string) json.RawMessage {
	return rawTopology(`{
		"nodes":[
			{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":250},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
			{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":160,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"` + dnsValue + `","ttl":300}]}},
			{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":250,"y":250},"config":{"ip":"10.0.1.1"}},
			{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":440,"y":250},"config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"` + firewallAction + `","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443},{"priority":200,"action":"allow","protocol":"tcp","source":"0.0.0.0/0","destination":"0.0.0.0/0","port":443}]}},
			{"id":"lb-1","name":"Load Balancer","type":"load_balancer","status":"healthy","position":{"x":640,"y":250},"config":{"ip":"10.0.2.10","algorithm":"round_robin","autoDiscoverConnectedServers":false,"backends":` + backends + `}},
			{"id":"server-1","name":"Server 1","type":"server","status":"` + server1Status + `","position":{"x":850,"y":170},"config":{"ip":"10.0.2.21","port":443}},
			{"id":"server-2","name":"Server 2","type":"server","status":"healthy","position":{"x":850,"y":330},"config":{"ip":"10.0.2.22","port":443}},
			{"id":"server-3","name":"Server 3","type":"server","status":"healthy","position":{"x":850,"y":460},"config":{"ip":"10.0.2.23","port":443}}
		],
		"links":[
			{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
			{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
			{"id":"l-router-fw","sourceNodeId":"router-1","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
			{"id":"l-fw-lb","sourceNodeId":"firewall-1","targetNodeId":"lb-1","status":"active","config":{"latencyMs":12}},
			{"id":"l-fw-s1-direct","sourceNodeId":"firewall-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8}},
			{"id":"l-lb-s1","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":4}},
			{"id":"l-lb-s2","sourceNodeId":"lb-1","targetNodeId":"server-2","status":"active","config":{"latencyMs":6}},
			{"id":"l-lb-s3","sourceNodeId":"lb-1","targetNodeId":"server-3","status":"active","config":{"latencyMs":9}}
		]
	}`)
}
