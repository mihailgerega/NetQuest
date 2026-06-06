package quests

import (
	"encoding/json"
	"strings"
)

const (
	apiHostname = "api.netquest.local"
	apiURL      = "https://api.netquest.local/users"
)

func Catalog() []Quest {
	quests := []Quest{
		{
			ID:                 "quest-dns-lookup",
			Slug:               "pochini-dns-lookup",
			Title:              "Почини DNS Lookup",
			Difficulty:         DifficultyEasy,
			Category:           "DNS",
			Description:        "Client не может открыть HTTPS API, потому что DNS-запись отсутствует или указывает на неправильный IP.",
			Goal:               "Настройте A record api.netquest.local так, чтобы он указывал на Server.",
			LearningObjectives: []string{"Понять DNS A records", "Увидеть DNS response в Timeline"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":70,"y":210},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24","defaultGateway":"10.0.1.1"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":230,"y":90},"config":{"ip":"10.0.1.53","records":[]}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":270,"y":230},"config":{"ip":"10.0.1.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":510,"y":230},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":3,"packetLossPercent":0}},
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5,"packetLossPercent":0}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8,"packetLossPercent":0}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "dns_record_exists", Type: CheckDNS, Title: "DNS-запись настроена", SourceNodeID: "client-1", Hostname: apiHostname, ExpectedIP: "10.0.2.21", Hint: "Добавьте A-запись api.netquest.local -> 10.0.2.21."},
				{ID: "dns_lookup_completed", Type: CheckReachability, Title: "DNS Lookup проходит", SourceNodeID: "client-1", ScenarioType: "dns_lookup", Target: apiHostname, ExpectedStatus: "completed"},
			},
			Hints:            []string{"Откройте DNS-узел в инспекторе.", "Добавьте A-запись для api.netquest.local.", "Значение должно указывать на IP нужного Server."},
			SuccessMessage:   "DNS Lookup починен: hostname резолвится в правильный IP.",
			FailureMessage:   "DNS всё ещё не возвращает правильный A record.",
			EstimatedMinutes: 6,
		},
		{
			ID:                 "quest-allow-ping",
			Slug:               "razreshi-ping",
			Title:              "Разреши Ping",
			Difficulty:         DifficultyEasy,
			Category:           "ICMP / Routing",
			Description:        "Client не может выполнить Ping до Server, потому что канал связи выключен.",
			Goal:               "Сделайте active path Client -> Router -> Server.",
			LearningObjectives: []string{"Понять route.selected", "Понять Ping RTT"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":80,"y":220},"config":{"ip":"10.0.1.10"}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":300,"y":220},"config":{"ip":"10.0.1.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":540,"y":220},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"down","config":{"latencyMs":8}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "ping_completed", Type: CheckReachability, Title: "Ping completed", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed"},
				{ID: "route_includes_router", Type: CheckRoute, Title: "Route проходит через Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}},
			},
			Hints:            []string{"Проверьте status у links.", "Убедитесь, что Router соединён с Server.", "Ping использует RTT: путь туда и обратно."},
			SuccessMessage:   "Ping проходит, route найден.",
			FailureMessage:   "Ping всё ещё не проходит.",
			EstimatedMinutes: 5,
		},
		{
			ID:                 "quest-firewall-https",
			Slug:               "otkroy-https-na-firewall",
			Title:              "Открой HTTPS на Firewall",
			Difficulty:         DifficultyEasy,
			Category:           "Firewall",
			Description:        "DNS работает, route найден, но Firewall блокирует HTTPS request.",
			Goal:               "Добавьте allow rule для tcp/443 от Client subnet к Server IP.",
			LearningObjectives: []string{"Понять Firewall rules", "Понять TCP/TLS события"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":70,"y":220},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":150,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.21","ttl":300}]}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":250,"y":220},"config":{"ip":"10.0.1.1"}},
					{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":430,"y":220},"config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[]}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":640,"y":220},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-fw","sourceNodeId":"router-1","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
					{"id":"l-fw-server","sourceNodeId":"firewall-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":12}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "firewall_allows_https", Type: CheckFirewall, Title: "Firewall разрешает tcp/443", NodeID: "firewall-1", ExpectedAction: "allow", Protocol: "tcp", Port: 443, ExpectedIP: "10.0.2.21"},
				{ID: "https_completed", Type: CheckReachability, Title: "HTTPS completed", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			},
			Hints:            []string{"Firewall проверяет protocol, source, destination и port.", "Для HTTPS нужен tcp/443.", "Добавьте allow rule выше deny/default policy."},
			SuccessMessage:   "HTTPS проходит через Firewall.",
			FailureMessage:   "Firewall всё ещё блокирует HTTPS.",
			EstimatedMinutes: 8,
		},
		{
			ID:                 "quest-lb-backend-pool",
			Slug:               "nastroj-lb-backend-pool",
			Title:              "Настрой пул серверов Load Balancer",
			Difficulty:         DifficultyMedium,
			Category:           "Load Balancer",
			Description:        "DNS указывает на Load Balancer, но пул серверов пустой.",
			Goal:               "Добавьте Server-1 и Server-2 в пул серверов.",
			LearningObjectives: []string{"Понять пул серверов", "Понять решение Load Balancer"},
			InitialTopology:    lbTopology(`[]`, "healthy", "healthy", "allow", "10.0.2.10"),
			ExpectedChecks: []CheckSpec{
				{ID: "lb_pool_contains_servers", Type: CheckLB, Title: "Backend pool содержит Server-1/Server-2", NodeID: "lb-1", RequiredBackends: []string{"server-1", "server-2"}},
				{ID: "https_completed_via_lb", Type: CheckReachability, Title: "HTTPS проходит через LB", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			},
			Hints:            []string{"Выберите Load Balancer.", "В инспекторе добавьте серверы в пул.", "Сервер должен быть исправен и достижим."},
			SuccessMessage:   "Load Balancer выбирает исправный сервер.",
			FailureMessage:   "Backend pool ещё настроен неверно.",
			EstimatedMinutes: 10,
		},
		{
			ID:                 "quest-default-route",
			Slug:               "pochini-default-route",
			Title:              "Почини default route",
			Difficulty:         DifficultyMedium,
			Category:           "Routing",
			Description:        "Client и Server в разных subnet, но default gateway или route отсутствует.",
			Goal:               "Настройте defaultGateway у Client и route table на Router.",
			LearningObjectives: []string{"Понять default route", "Понять longest prefix match"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":70,"y":220},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":300,"y":220},"config":{"ip":"10.0.1.1","interfaces":[{"name":"eth0","ip":"10.0.1.1","cidr":"10.0.1.1/24"},{"name":"eth1","ip":"10.0.2.1","cidr":"10.0.2.1/24"}],"routes":[]}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":540,"y":220},"config":{"ip":"10.0.2.21","cidr":"10.0.2.21/24","port":443}}
				],
				"links":[
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "ping_uses_router", Type: CheckRoute, Title: "Route использует Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}},
			},
			Hints:            []string{"Проверьте gateway у Client.", "Проверьте routing table на Router.", "Default route обычно 0.0.0.0/0."},
			SuccessMessage:   "Routing между subnet работает.",
			FailureMessage:   "Route между subnet всё ещё не найден.",
			EstimatedMinutes: 12,
		},
		{
			ID:                 "quest-latency-diagnostics",
			Slug:               "uberi-lishnyuyu-latency",
			Title:              "Убери лишнюю latency",
			Difficulty:         DifficultyMedium,
			Category:           "Latency / Diagnostics",
			Description:        "HTTPS проходит, но слишком медленно из-за slow link.",
			Goal:               "Сделайте totalLatencyMs ниже 300ms и исключите slow link из path.",
			LearningObjectives: []string{"Понять разбор задержки", "Понять взвешенный путь"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":230},"config":{"ip":"10.0.1.10"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":110,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
					{"id":"router-a","name":"Router A","type":"router","status":"healthy","position":{"x":260,"y":160},"config":{"ip":"10.0.1.1"}},
					{"id":"router-b","name":"Router B","type":"router","status":"healthy","position":{"x":260,"y":330},"config":{"ip":"10.0.1.2"}},
					{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":470,"y":240},"config":{"ip":"10.0.1.254","defaultPolicy":"allow"}},
					{"id":"lb-1","name":"Load Balancer","type":"load_balancer","status":"healthy","position":{"x":660,"y":240},"config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","enabled":true}]}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":860,"y":240},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"slow-link","sourceNodeId":"client-1","targetNodeId":"router-a","status":"active","config":{"latencyMs":1000}},
					{"id":"fast-link","sourceNodeId":"client-1","targetNodeId":"router-b","status":"down","config":{"latencyMs":5}},
					{"id":"l-a-fw","sourceNodeId":"router-a","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
					{"id":"l-b-fw","sourceNodeId":"router-b","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
					{"id":"l-fw-lb","sourceNodeId":"firewall-1","targetNodeId":"lb-1","status":"active","config":{"latencyMs":12}},
					{"id":"l-lb-server","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":4}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "https_latency_below_300", Type: CheckLatency, Title: "HTTPS latency ниже 300ms", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MaxTotalLatencyMs: 300, MustExcludePath: []string{"router-a"}},
				{ID: "no_slow_selected_path", Type: CheckAdvisor, Title: "Advisor не видит slow path issue", ForbiddenIssueCode: "HIGH_LATENCY_LINK"},
			},
			Hints:            []string{"Откройте инспектор пакета и посмотрите расчёт задержки.", "Найдите канал связи с высокой задержкой.", "Включите быстрый канал или уменьшите задержку."},
			SuccessMessage:   "Запрос идёт быстрым path.",
			FailureMessage:   "Latency всё ещё слишком высокая.",
			EstimatedMinutes: 12,
		},
		{
			ID:                 "quest-backend-failover",
			Slug:               "failover-backend-posle-padeniya-server",
			Title:              "Failover backend после падения Server",
			Difficulty:         DifficultyHard,
			Category:           "Failover / Load Balancer",
			Description:        "Server-1 выключен, но Load Balancer должен выбрать исправный сервер.",
			Goal:               "Настройте LB pool так, чтобы request уходил на Server-2 или Server-3.",
			LearningObjectives: []string{"Понять skippedBackends", "Понять failover"},
			InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "healthy", "allow", "10.0.2.10"),
			ExpectedChecks: []CheckSpec{
				{ID: "down_backend_skipped", Type: CheckFailover, Title: "Выключенный сервер пропущен", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
				{ID: "lb_has_healthy_pool", Type: CheckLB, Title: "Пул содержит исправный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}},
			},
			Hints:            []string{"Проверьте пул серверов.", "Выключенный сервер должен быть исключён из выбора.", "Добавьте Server-2 или Server-3 как исправный сервер."},
			SuccessMessage:   "Failover работает: down backend пропускается.",
			FailureMessage:   "Load Balancer всё ещё не имеет healthy failover backend.",
			EstimatedMinutes: 15,
		},
		{
			ID:                 "quest-deny-direct-server",
			Slug:               "zapreti-direct-access-k-server",
			Title:              "Запрети direct access к Server",
			Difficulty:         DifficultyHard,
			Category:           "Security / Firewall / Load Balancer",
			Description:        "Client должен получать HTTPS только через Load Balancer, direct access к Server запрещён.",
			Goal:               "Разрешите Client -> Load Balancer tcp/443 и запретите Client -> Server tcp/443.",
			LearningObjectives: []string{"Понять security boundary", "Понять Firewall rule order"},
			InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]`, "healthy", "healthy", "allow", "10.0.2.10"),
			ExpectedChecks: []CheckSpec{
				{ID: "normal_https_completed", Type: CheckReachability, Title: "HTTPS через LB проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
				{ID: "direct_server_denied", Type: CheckSecurity, Title: "Direct access к Server запрещён", SourceNodeID: "client-1", Target: "https://10.0.2.21/users", ExpectedStatus: "failed", ForbiddenTarget: "server-1"},
				{ID: "normal_path_includes_lb", Type: CheckRoute, Title: "Normal path включает Load Balancer", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MustIncludePath: []string{"lb-1"}},
			},
			Hints:            []string{"Разделяйте доступ к Load Balancer и Server.", "Firewall rule order имеет значение.", "LB должен иметь доступ к backend, Client — нет."},
			SuccessMessage:   "Direct access закрыт, доступ через LB работает.",
			FailureMessage:   "Security policy ещё неверная.",
			EstimatedMinutes: 18,
		},
		{
			ID:                 "quest-backup-route",
			Slug:               "rezervnyj-route-posle-padeniya-link",
			Title:              "Резервный route после падения link",
			Difficulty:         DifficultyHard,
			Category:           "Routing / Failover",
			Description:        "Основной канал выключен, нужно использовать альтернативный маршрут.",
			Goal:               "Сделайте так, чтобы route.selected использовал Router-C.",
			LearningObjectives: []string{"Понять исключение выключенного канала", "Понять резервный маршрут"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":260},"config":{"ip":"10.0.1.10"}},
					{"id":"router-a","name":"Router A","type":"router","status":"healthy","position":{"x":250,"y":150},"config":{"ip":"10.0.1.1"}},
					{"id":"router-c","name":"Router C","type":"router","status":"healthy","position":{"x":250,"y":360},"config":{"ip":"10.0.1.2"}},
					{"id":"router-b","name":"Router B","type":"router","status":"healthy","position":{"x":480,"y":260},"config":{"ip":"10.0.2.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":720,"y":260},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"primary-link","sourceNodeId":"client-1","targetNodeId":"router-a","status":"down","config":{"latencyMs":4}},
					{"id":"backup-link","sourceNodeId":"client-1","targetNodeId":"router-c","status":"down","config":{"latencyMs":8}},
					{"id":"l-a-b","sourceNodeId":"router-a","targetNodeId":"router-b","status":"active","config":{"latencyMs":8}},
					{"id":"l-c-b","sourceNodeId":"router-c","targetNodeId":"router-b","status":"active","config":{"latencyMs":10}},
					{"id":"l-b-server","sourceNodeId":"router-b","targetNodeId":"server-1","status":"active","config":{"latencyMs":6}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "backup_route_works", Type: CheckRoute, Title: "Backup route использует Router-C", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-c"}, MustExcludePath: []string{"router-a"}},
			},
			Hints:            []string{"Проверьте status link.", "Проверьте route cost.", "Резервный path должен быть reachable."},
			SuccessMessage:   "Backup route работает.",
			FailureMessage:   "Request всё ещё не использует резервный path.",
			EstimatedMinutes: 16,
		},
		{
			ID:                 "quest-production-api-diagnostics",
			Slug:               "kompleksnaya-diagnostika-production-api",
			Title:              "Комплексная диагностика Production API",
			Difficulty:         DifficultyHard,
			Category:           "DNS + Routing + Firewall + TLS + Load Balancer",
			Description:        "Production API недоступен из-за нескольких ошибок topology.",
			Goal:               "Сделайте так, чтобы Client-2 получил HTTPS response от api.netquest.local через Load Balancer.",
			LearningObjectives: []string{"Диагностировать DNS", "Диагностировать Firewall", "Диагностировать LB failover"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client 1","type":"client","status":"healthy","position":{"x":60,"y":180},"config":{"ip":"10.0.1.10"}},
					{"id":"client-2","name":"Client 2","type":"client","status":"healthy","position":{"x":60,"y":310},"config":{"ip":"10.0.1.11"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":170,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.99","ttl":300}]}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":270,"y":250},"config":{"ip":"10.0.1.1"}},
					{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":460,"y":250},"config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"deny","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443}]}},
					{"id":"lb-1","name":"Load Balancer","type":"load_balancer","status":"healthy","position":{"x":650,"y":250},"config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]}},
					{"id":"server-1","name":"Server 1","type":"server","status":"down","position":{"x":860,"y":170},"config":{"ip":"10.0.2.21","port":443}},
					{"id":"server-2","name":"Server 2","type":"server","status":"healthy","position":{"x":860,"y":340},"config":{"ip":"10.0.2.22","port":443,"certificateHostname":"old.netquest.local"}}
				],
				"links":[
					{"id":"l-c1-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-c2-router","sourceNodeId":"client-2","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-c2-dns","sourceNodeId":"client-2","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"l-router-fw","sourceNodeId":"router-1","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
					{"id":"l-fw-lb","sourceNodeId":"firewall-1","targetNodeId":"lb-1","status":"active","config":{"latencyMs":12}},
					{"id":"l-lb-s1","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":4}},
					{"id":"l-lb-s2","sourceNodeId":"lb-1","targetNodeId":"server-2","status":"active","config":{"latencyMs":5}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "client2_https_completed", Type: CheckReachability, Title: "Client-2 HTTPS completed", SourceNodeID: "client-2", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
				{ID: "dns_to_lb", Type: CheckDNS, Title: "DNS указывает на Load Balancer", SourceNodeID: "client-2", Hostname: apiHostname, ExpectedIP: "10.0.2.10"},
				{ID: "server1_skipped", Type: CheckFailover, Title: "Server-1 выключен и пропущен", SourceNodeID: "client-2", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
			},
			Hints:            []string{"Начните с DNS.", "Затем проверьте Firewall.", "Потом проверьте имя TLS-сертификата.", "Последним проверьте пул серверов и health."},
			SuccessMessage:   "Production API восстановлен.",
			FailureMessage:   "В topology ещё есть блокирующая проблема.",
			EstimatedMinutes: 25,
		},
	}
	quests = append(quests, additionalQuests()...)
	return enrichQuestCatalog(quests)
}

func additionalQuests() []Quest {
	return []Quest{
		{
			ID:                 "quest-v2-client-source",
			Slug:               "zapros-ot-zdorovogo-client",
			Title:              "Запрос от другого Client",
			Difficulty:         DifficultyEasy,
			Category:           "Simulation basics",
			Description:        "В topology есть два client. Проверка ждёт запрос именно от Client-2, но этот endpoint сейчас выключен.",
			Goal:               "Восстановите Client-2 и убедитесь, что Ping от Client-2 до Server проходит.",
			LearningObjectives: []string{"Выбирать исходный узел для симуляции", "Понимать влияние состояния узла"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client 1","type":"client","status":"healthy","position":{"x":70,"y":170},"config":{"ip":"10.0.1.10"}},
					{"id":"client-2","name":"Client 2","type":"client","status":"down","position":{"x":70,"y":330},"config":{"ip":"10.0.1.11"}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":310,"y":250},"config":{"ip":"10.0.1.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":560,"y":250},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-c1-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-c2-router","sourceNodeId":"client-2","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "client2_ping_completed", Type: CheckReachability, Title: "Client-2 отправляет Ping", SourceNodeID: "client-2", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", Hint: "Выберите Client-2 и верните status healthy."},
			},
			Hints:            []string{"Источник запроса задаётся в верхней панели simulator.", "Если исходный узел выключен, пакет даже не стартует.", "Верните Client-2 в healthy и повторите Ping."},
			SuccessMessage:   "Client-2 снова может отправлять traffic.",
			FailureMessage:   "Ping от Client-2 всё ещё не проходит.",
			EstimatedMinutes: 5,
		},
		{
			ID:                 "quest-v2-dns-resolver-down",
			Slug:               "vosstanovi-dns-resolver",
			Title:              "Восстанови DNS-сервер",
			Difficulty:         DifficultyEasy,
			Category:           "DNS",
			Description:        "DNS-запись уже правильная, но сам DNS-узел выключен, поэтому lookup не отвечает.",
			Goal:               "Верните DNS-узел в healthy и добейтесь успешного DNS Lookup.",
			LearningObjectives: []string{"Отличать record error от resolver outage", "Читать dns.error в Timeline"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":70,"y":240},"config":{"ip":"10.0.1.10"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"down","position":{"x":220,"y":90},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.21","ttl":300}]}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":320,"y":240},"config":{"ip":"10.0.1.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":560,"y":240},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "dns_lookup_completed", Type: CheckReachability, Title: "DNS Lookup проходит", SourceNodeID: "client-1", ScenarioType: "dns_lookup", Target: apiHostname, ExpectedStatus: "completed", Hint: "Проверьте status DNS-узла: он должен быть healthy."},
			},
			Hints:            []string{"Запись может быть правильной, но DNS-сервер всё равно недоступен.", "Откройте DNS-узел в инспекторе.", "Status DNS должен быть healthy."},
			SuccessMessage:   "DNS-сервер отвечает и возвращает A-запись.",
			FailureMessage:   "DNS Lookup всё ещё не завершается успешно.",
			EstimatedMinutes: 5,
		},
		{
			ID:                 "quest-v2-packet-loss",
			Slug:               "uberi-packet-loss",
			Title:              "Убери packet loss",
			Difficulty:         DifficultyEasy,
			Category:           "Reliability",
			Description:        "Связь физически есть, но link теряет все packets. Timeline должен показать deterministic drop.",
			Goal:               "Снизьте packetLossPercent на проблемном link до 0 и добейтесь успешного Ping.",
			LearningObjectives: []string{"Понимать packet loss", "Видеть retry/drop в Timeline"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":80,"y":250},"config":{"ip":"10.0.1.10"}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":320,"y":250},"config":{"ip":"10.0.1.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":570,"y":250},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"lossy-link","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5,"packetLossPercent":100}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8,"packetLossPercent":0}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "ping_without_loss", Type: CheckReachability, Title: "Ping проходит без drop", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", Hint: "Откройте lossy-link и поставьте packetLossPercent = 0."},
			},
			Hints:            []string{"Active link не означает надёжный канал связи.", "Потеря пакетов на 100% гарантированно ломает симуляцию.", "Поставьте packetLossPercent в 0 и запустите Ping ещё раз."},
			SuccessMessage:   "Потеря пакетов устранена, Ping проходит.",
			FailureMessage:   "Traffic всё ещё теряется на пути.",
			EstimatedMinutes: 6,
		},
		{
			ID:                 "quest-v2-lb-add-fallback",
			Slug:               "dobav-fallback-backend",
			Title:              "Добавь резервный сервер",
			Difficulty:         DifficultyMedium,
			Category:           "Load Balancer",
			Description:        "Server-1 выключен, а Load Balancer знает только о нём. Рядом есть исправные Server-2 и Server-3.",
			Goal:               "Добавьте Server-2 или Server-3 в пул серверов, чтобы HTTPS-запрос прошёл через failover.",
			LearningObjectives: []string{"Настраивать пул серверов", "Понимать пропущенные серверы"},
			InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "healthy", "allow", "10.0.2.10"),
			ExpectedChecks: []CheckSpec{
				{ID: "fallback_backend_present", Type: CheckLB, Title: "В пуле есть резервный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}, Hint: "В инспекторе Load Balancer добавьте Server-2 или Server-3."},
				{ID: "https_failover_completed", Type: CheckFailover, Title: "Failover HTTPS проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
			},
			Hints:            []string{"Load Balancer выбирает только из пула серверов.", "Выключенный сервер должен попасть в список пропущенных серверов.", "Добавьте исправный сервер, у которого есть активный путь от LB."},
			SuccessMessage:   "Load Balancer использует исправный резервный сервер.",
			FailureMessage:   "Failover backend ещё не настроен.",
			EstimatedMinutes: 11,
		},
		{
			ID:                 "quest-v2-default-gateway",
			Slug:               "nastroj-default-gateway",
			Title:              "Настрой default gateway",
			Difficulty:         DifficultyMedium,
			Category:           "Routing",
			Description:        "Client и Server находятся в разных subnet. Без defaultGateway Client не знает, куда отправлять packet.",
			Goal:               "Укажите defaultGateway 10.0.1.1 у Client и проверьте route через Router.",
			LearningObjectives: []string{"Понимать default gateway", "Отличать graph path от routed path"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":70,"y":240},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":320,"y":240},"config":{"ip":"10.0.1.1","interfaces":[{"name":"lan","ip":"10.0.1.1","cidr":"10.0.1.1/24"},{"name":"srv","ip":"10.0.2.1","cidr":"10.0.2.1/24"}]}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":570,"y":240},"config":{"ip":"10.0.2.21","cidr":"10.0.2.21/24","port":443}}
				],
				"links":[
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-server","sourceNodeId":"router-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":8}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "route_uses_gateway", Type: CheckRoute, Title: "Route идёт через Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}, Hint: "Добавьте defaultGateway 10.0.1.1 в config Client."},
			},
			Hints:            []string{"Если destination не в subnet Client, нужен gateway.", "Gateway должен быть reachable через active link.", "Для этого topology gateway равен 10.0.1.1."},
			SuccessMessage:   "Шлюз по умолчанию настроен, маршрут найден.",
			FailureMessage:   "Client всё ещё не знает путь в subnet Server.",
			EstimatedMinutes: 10,
		},
		{
			ID:                 "quest-v2-stale-lb-backend",
			Slug:               "udal-stale-backend",
			Title:              "Удали stale backend",
			Difficulty:         DifficultyMedium,
			Category:           "Load Balancer / Validation",
			Description:        "В пуле серверов осталась ссылка на удалённый server-old. Validator должен подсказать, что ссылка устарела.",
			Goal:               "Удалите server-old из пула серверов и оставьте достижимый исправный Server-1.",
			LearningObjectives: []string{"Понимать stale references", "Читать validation errors"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":250},"config":{"ip":"10.0.1.10"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":160,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.10","ttl":300}]}},
					{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":250,"y":250},"config":{"ip":"10.0.1.1"}},
					{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":440,"y":250},"config":{"ip":"10.0.1.254","defaultPolicy":"allow"}},
					{"id":"lb-1","name":"Load Balancer","type":"load_balancer","status":"healthy","position":{"x":640,"y":250},"config":{"ip":"10.0.2.10","algorithm":"round_robin","backends":[{"nodeId":"server-old","enabled":true},{"nodeId":"server-1","enabled":true}]}},
					{"id":"server-1","name":"Server 1","type":"server","status":"healthy","position":{"x":850,"y":250},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"l-client-router","sourceNodeId":"client-1","targetNodeId":"router-1","status":"active","config":{"latencyMs":5}},
					{"id":"l-router-fw","sourceNodeId":"router-1","targetNodeId":"firewall-1","status":"active","config":{"latencyMs":8}},
					{"id":"l-fw-lb","sourceNodeId":"firewall-1","targetNodeId":"lb-1","status":"active","config":{"latencyMs":12}},
					{"id":"l-lb-server","sourceNodeId":"lb-1","targetNodeId":"server-1","status":"active","config":{"latencyMs":5}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "lb_pool_has_server1", Type: CheckLB, Title: "Пул содержит Server-1", NodeID: "lb-1", RequiredBackends: []string{"server-1"}, Hint: "Удалите nodeId=server-old из пула."},
				{ID: "https_without_stale", Type: CheckReachability, Title: "HTTPS проходит без stale reference", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			},
			Hints:            []string{"Stale backend — это ссылка на node, которого уже нет на canvas.", "Validator не должен silently ignore такие ссылки.", "Удалите server-old и оставьте существующий server-1."},
			SuccessMessage:   "Backend pool больше не содержит stale references.",
			FailureMessage:   "В пуле серверов всё ещё есть проблемная ссылка.",
			EstimatedMinutes: 12,
		},
		{
			ID:                 "quest-v2-latency-threshold",
			Slug:               "sniz-latency-do-sla",
			Title:              "Снизь latency до SLA",
			Difficulty:         DifficultyMedium,
			Category:           "Latency / SRE",
			Description:        "HTTPS request проходит, но totalLatencyMs выше SLA из-за slow-link.",
			Goal:               "Сделайте active path быстрее 300ms: включите fast-link или уменьшите latency slow-link.",
			LearningObjectives: []string{"Понимать totalLatencyMs", "Искать slow link по Timeline"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":250},"config":{"ip":"10.0.1.10"}},
					{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":150,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"10.0.2.21","ttl":300}]}},
					{"id":"router-slow","name":"Router Slow","type":"router","status":"healthy","position":{"x":290,"y":180},"config":{"ip":"10.0.1.1"}},
					{"id":"router-fast","name":"Router Fast","type":"router","status":"healthy","position":{"x":290,"y":340},"config":{"ip":"10.0.1.2"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":610,"y":250},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"l-client-dns","sourceNodeId":"client-1","targetNodeId":"dns-1","status":"active","config":{"latencyMs":2}},
					{"id":"slow-link","sourceNodeId":"client-1","targetNodeId":"router-slow","status":"active","config":{"latencyMs":900}},
					{"id":"fast-link","sourceNodeId":"client-1","targetNodeId":"router-fast","status":"down","config":{"latencyMs":5}},
					{"id":"l-slow-server","sourceNodeId":"router-slow","targetNodeId":"server-1","status":"active","config":{"latencyMs":10}},
					{"id":"l-fast-server","sourceNodeId":"router-fast","targetNodeId":"server-1","status":"active","config":{"latencyMs":10}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "https_under_sla", Type: CheckLatency, Title: "HTTPS latency ниже 300ms", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MaxTotalLatencyMs: 300, MustExcludePath: []string{"router-slow"}, Hint: "Активируйте fast-link или уменьшите latency slow-link."},
			},
			Hints:            []string{"Смотрите разбор задержки в инспекторе пакета.", "Алгоритм пути по графу выбирает активный путь с наименьшей задержкой.", "Быстрый канал сейчас down, поэтому путь вынужден идти через медленный маршрутизатор."},
			SuccessMessage:   "Path укладывается в SLA.",
			FailureMessage:   "Latency всё ещё выше заданного порога.",
			EstimatedMinutes: 12,
		},
		{
			ID:                 "quest-v2-backup-route-hard",
			Slug:               "rezervnyj-route-hard",
			Title:              "Резервный route после аварии",
			Difficulty:         DifficultyHard,
			Category:           "Routing / Failover",
			Description:        "Основной канал выключен, backup-link тоже выключен. Нужно восстановить альтернативный путь без возврата на primary router.",
			Goal:               "Включите backup-link и добейтесь Ping через Router-C, исключив Router-A из path.",
			LearningObjectives: []string{"Понимать alternative path", "Проверять route.selected"},
			InitialTopology: rawTopology(`{
				"nodes":[
					{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":260},"config":{"ip":"10.0.1.10"}},
					{"id":"router-a","name":"Router A","type":"router","status":"healthy","position":{"x":250,"y":150},"config":{"ip":"10.0.1.1"}},
					{"id":"router-c","name":"Router C","type":"router","status":"healthy","position":{"x":250,"y":360},"config":{"ip":"10.0.1.2"}},
					{"id":"router-b","name":"Router B","type":"router","status":"healthy","position":{"x":480,"y":260},"config":{"ip":"10.0.2.1"}},
					{"id":"server-1","name":"Server","type":"server","status":"healthy","position":{"x":720,"y":260},"config":{"ip":"10.0.2.21","port":443}}
				],
				"links":[
					{"id":"primary-link","sourceNodeId":"client-1","targetNodeId":"router-a","status":"down","config":{"latencyMs":4}},
					{"id":"backup-link","sourceNodeId":"client-1","targetNodeId":"router-c","status":"down","config":{"latencyMs":8}},
					{"id":"l-a-b","sourceNodeId":"router-a","targetNodeId":"router-b","status":"active","config":{"latencyMs":8}},
					{"id":"l-c-b","sourceNodeId":"router-c","targetNodeId":"router-b","status":"active","config":{"latencyMs":10}},
					{"id":"l-b-server","sourceNodeId":"router-b","targetNodeId":"server-1","status":"active","config":{"latencyMs":6}}
				]
			}`),
			ExpectedChecks: []CheckSpec{
				{ID: "backup_route_uses_router_c", Type: CheckRoute, Title: "Backup route использует Router-C", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-c"}, MustExcludePath: []string{"router-a"}, Hint: "Включите backup-link Client -> Router-C, primary-link оставьте down."},
			},
			Hints:            []string{"Primary route уже считается аварийным.", "Не надо чинить всё сразу: нужен рабочий backup path.", "Проверьте, что route.selected содержит router-c."},
			SuccessMessage:   "Backup route восстановлен.",
			FailureMessage:   "Backup route всё ещё не выбран.",
			EstimatedMinutes: 16,
		},
		{
			ID:                 "quest-v2-secure-lb-boundary",
			Slug:               "zakroj-direct-server-hard",
			Title:              "Закрой direct server access",
			Difficulty:         DifficultyHard,
			Category:           "Security / Firewall",
			Description:        "Публичный HTTPS должен идти через Load Balancer. Direct access к Server-1 не должен проходить.",
			Goal:               "Сохраните HTTPS через LB, но запретите direct request к Server-1.",
			LearningObjectives: []string{"Строить security boundary", "Понимать порядок firewall rules"},
			InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]`, "healthy", "healthy", "allow", "10.0.2.10"),
			ExpectedChecks: []CheckSpec{
				{ID: "https_via_lb_still_works", Type: CheckReachability, Title: "HTTPS через LB проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
				{ID: "direct_server_blocked", Type: CheckSecurity, Title: "Direct Server-1 access заблокирован", SourceNodeID: "client-1", Target: "https://10.0.2.21/users", ExpectedStatus: "failed", ForbiddenTarget: "server-1", Hint: "Добавьте deny rule для Client -> Server-1 tcp/443 выше широкого allow."},
			},
			Hints:            []string{"Проверяйте два сценария: нормальный URL и direct IP.", "Правило deny должно матчить Server-1.", "Не сломайте path Client -> Load Balancer."},
			SuccessMessage:   "Security boundary настроен корректно.",
			FailureMessage:   "Direct access к backend всё ещё открыт или LB path сломан.",
			EstimatedMinutes: 18,
		},
		{
			ID:                 "quest-v2-production-multi-issue",
			Slug:               "production-multi-issue",
			Title:              "Production API: несколько причин отказа",
			Difficulty:         DifficultyHard,
			Category:           "DNS + Firewall + Load Balancer",
			Description:        "Production API не открывается из-за неправильного DNS, deny rule и отсутствия исправного резервного сервера.",
			Goal:               "Почините DNS на LB, разрешите tcp/443 к LB и настройте failover на Server-2 или Server-3.",
			LearningObjectives: []string{"Диагностировать несколько слоёв", "Проверять решение серверной проверкой"},
			InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "healthy", "deny", "10.0.2.99"),
			ExpectedChecks: []CheckSpec{
				{ID: "dns_points_to_lb", Type: CheckDNS, Title: "DNS указывает на LB", SourceNodeID: "client-1", Hostname: apiHostname, ExpectedIP: "10.0.2.10", Hint: "A record api.netquest.local должен быть 10.0.2.10."},
				{ID: "fallback_backend_exists", Type: CheckLB, Title: "Есть исправный резервный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}, Hint: "Server-1 выключен, добавьте Server-2 или Server-3 в пул."},
				{ID: "production_https_completed", Type: CheckFailover, Title: "HTTPS проходит с failover", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
			},
			Hints:            []string{"Идите по слоям: DNS -> Firewall -> Load Balancer.", "После каждой правки запускайте HTTPS и смотрите первый failed event.", "Решение должно выбрать backend не server-1."},
			SuccessMessage:   "Production API восстановлен на всех нужных слоях.",
			FailureMessage:   "Один из слоёв всё ещё блокирует request.",
			EstimatedMinutes: 24,
		},
	}
}

func enrichQuestCatalog(quests []Quest) []Quest {
	enriched := make([]Quest, len(quests))
	for i, quest := range quests {
		quest.Title = normalizeHintText(quest.Title)
		quest.Description = normalizeHintText(quest.Description)
		quest.Goal = normalizeHintText(quest.Goal)
		quest.LearningObjectives = normalizeTextList(quest.LearningObjectives)
		quest.SuccessMessage = normalizeHintText(quest.SuccessMessage)
		quest.FailureMessage = normalizeHintText(quest.FailureMessage)
		quest.ExpectedChecks = normalizeCheckSpecs(quest.ExpectedChecks)
		quest.Hints = normalizeLegacyHints(quest.Hints)
		quest.ProgressiveHints = normalizeProgressiveHints(quest)
		if quest.AfterSolution == "" {
			quest.AfterSolution = defaultAfterSolution(quest)
		}
		if len(quest.GlossaryTerms) == 0 {
			quest.GlossaryTerms = defaultGlossaryTerms(quest.Category)
		}
		if quest.RealWorldImportance == "" {
			quest.RealWorldImportance = defaultRealWorldImportance(quest.Category)
		}
		enriched[i] = quest
	}
	return enriched
}

func normalizeProgressiveHints(quest Quest) []ProgressiveHint {
	minHints := 4
	if quest.Difficulty == DifficultyHard {
		minHints = 5
	}
	if len(quest.ProgressiveHints) > 0 {
		hints := make([]ProgressiveHint, 0, maxInt(minHints, len(quest.ProgressiveHints)))
		for _, hint := range quest.ProgressiveHints {
			hint.Body = normalizeHintText(hint.Body)
			hint.Actions = normalizeHintActions(hint.Actions)
			hints = append(hints, hint)
		}
		return ensureMinimumProgressiveHints(quest, hints, minHints)
	}
	hints := []ProgressiveHint{
		{
			Title: "Сначала поймите симптом",
			Body:  "Прочитайте цель упражнения и запустите подходящую симуляцию. Первое событие с ошибкой в Timeline обычно показывает слой, с которого стоит начать диагностику.",
			Level: "concept",
			Actions: []string{
				"Запустите DNS, Ping или HTTPS в зависимости от цели упражнения.",
				"Откройте Timeline и инспектор пакета.",
			},
		},
	}
	for i, hint := range quest.Hints {
		if hint == "" {
			continue
		}
		level := "guided"
		title := "Подсказка"
		if i == 0 {
			title = "Где искать"
		} else if i == len(quest.Hints)-1 {
			title = "Конкретное действие"
			level = "action"
		}
		relatedCheckID := ""
		if i < len(quest.ExpectedChecks) {
			relatedCheckID = quest.ExpectedChecks[i].ID
		}
		hints = append(hints, ProgressiveHint{
			Title:          title,
			Body:           normalizeHintText(hint),
			Level:          level,
			RelatedCheckID: relatedCheckID,
			Actions: []string{
				"Выберите связанный узел или канал связи в инспекторе.",
				"После изменения снова нажмите «Проверить решение».",
			},
		})
	}
	return ensureMinimumProgressiveHints(quest, hints, minHints)
}

func defaultAfterSolution(quest Quest) string {
	return "После исправления «" + quest.Title + "» NetQuest подтверждает решение не по картинке на рабочем поле, а по поведению сети: сервер запускает симуляцию, проверяет ожидаемые условия и сравнивает фактический путь, статусы и решения узлов с целью упражнения."
}

func defaultGlossaryTerms(category string) []GlossaryTerm {
	terms := []GlossaryTerm{
		{Term: "Узел", Definition: "Элемент топологии: client, server, DNS, router, firewall или load balancer."},
		{Term: "Канал связи", Definition: "Связь между узлами. Состояние, задержка и потеря пакетов влияют на маршрут и виртуальное время."},
		{Term: "Timeline", Definition: "Список событий виртуального пакета: DNS, route, firewall, TCP/TLS, выбор сервера и delivery."},
	}
	switch {
	case containsFold(category, "DNS"):
		terms = append(terms, GlossaryTerm{Term: "A-запись", Definition: "DNS-запись, которая сопоставляет доменное имя с IPv4-адресом."})
	case containsFold(category, "Load Balancer"):
		terms = append(terms, GlossaryTerm{Term: "Пул серверов", Definition: "Список серверов, из которых Load Balancer выбирает доступную цель."})
	case containsFold(category, "Firewall"):
		terms = append(terms, GlossaryTerm{Term: "Правило firewall", Definition: "Правило allow/deny для протокола, порта, источника и назначения."})
	case containsFold(category, "Routing"):
		terms = append(terms, GlossaryTerm{Term: "Маршрут", Definition: "Путь пакета от исходного узла до цели через активные каналы связи."})
	case containsFold(category, "Latency"):
		terms = append(terms, GlossaryTerm{Term: "Задержка", Definition: "Виртуальное время канала связи или этапа, из которого складывается totalLatencyMs."})
	}
	return terms
}

func defaultRealWorldImportance(category string) string {
	switch {
	case containsFold(category, "DNS"):
		return "Сбои DNS часто выглядят как отказ приложения, хотя серверная часть может быть полностью исправна."
	case containsFold(category, "Load Balancer"):
		return "В реальной инфраструктуре failover зависит от актуального пула серверов, health-checks и достижимости, а не от подписи узла на схеме."
	case containsFold(category, "Firewall"), containsFold(category, "Security"):
		return "Ошибки firewall rules либо блокируют легитимный traffic, либо случайно открывают прямой доступ к серверной части."
	case containsFold(category, "Latency"):
		return "SRE смотрит не только на факт успешного ответа, но и на latency budget по каждому этапу запроса."
	case containsFold(category, "Routing"):
		return "Ошибки маршрутизации приводят к недоступным подсетям, асимметричным путям и сложным инцидентам между сегментами сети."
	default:
		return "Упражнение показывает безопасную виртуальную модель: настоящие сетевые пакеты не отправляются, всё рассчитывается внутри simulation engine."
	}
}

func ensureMinimumProgressiveHints(quest Quest, hints []ProgressiveHint, minHints int) []ProgressiveHint {
	for len(hints) < minHints {
		hints = append(hints, fallbackProgressiveHint(quest, len(hints), minHints))
	}
	return hints
}

func fallbackProgressiveHint(quest Quest, index, minHints int) ProgressiveHint {
	switch index {
	case 0:
		return ProgressiveHint{
			Title: "Первый шаг",
			Body:  "Найдите проверку с ошибкой, затем проверьте состояние связанных узлов, состояние канала связи и настройки выбранного элемента.",
			Level: "guided",
			Actions: []string{
				"Откройте Timeline.",
				"Сравните первое событие с ошибкой с целью упражнения.",
			},
		}
	case 1:
		return ProgressiveHint{
			Title: "Определите слой",
			Body:  categoryLayerHint(quest.Category),
			Level: "guided",
			Actions: []string{
				"Откройте инспектор элемента из подсказки.",
				"Проверьте поля, которые влияют на этот слой.",
			},
		}
	case minHints - 1:
		return ProgressiveHint{
			Title: "Почти решение",
			Body:  finalActionHint(quest),
			Level: "action",
			Actions: []string{
				"Внесите изменение в topology.",
				"Запустите симуляцию и нажмите «Проверить решение».",
			},
		}
	default:
		return ProgressiveHint{
			Title: "Что изменить",
			Body:  checkBasedHint(quest),
			Level: "guided",
			Actions: []string{
				"Откройте связанный узел или канал в инспекторе.",
				"Проверьте, совпадают ли его настройки с целью упражнения.",
			},
		}
	}
}

func categoryLayerHint(category string) string {
	switch {
	case containsFold(category, "DNS"):
		return "Сосредоточьтесь на DNS-узле: он должен быть доступен, а A-запись должна указывать на правильный IP-адрес."
	case containsFold(category, "Routing"):
		return "Сосредоточьтесь на маршрутизации: проверьте шлюз, таблицу маршрутизации, активные каналы связи и выбранный путь."
	case containsFold(category, "Firewall"), containsFold(category, "Security"):
		return "Сосредоточьтесь на firewall: проверьте порядок правил, протокол, порт, источник и назначение."
	case containsFold(category, "Load Balancer"), containsFold(category, "Failover"):
		return "Сосредоточьтесь на Load Balancer: в пуле должен быть доступный сервер, до которого есть активный путь."
	case containsFold(category, "Latency"):
		return "Сосредоточьтесь на каналах связи: высокая задержка или потеря пакетов напрямую меняют виртуальное время и итоговый статус."
	case containsFold(category, "TLS"):
		return "Сосредоточьтесь на TLS-настройках: домен запроса должен совпадать с именем сертификата выбранного сервера."
	default:
		return "Проверьте слой, на котором появляется первое событие с ошибкой: DNS, route, firewall, Load Balancer или link."
	}
}

func checkBasedHint(quest Quest) string {
	for _, check := range quest.ExpectedChecks {
		if check.Hint != "" {
			return normalizeHintText(check.Hint)
		}
		switch check.Type {
		case CheckDNS:
			return "Проверьте DNS-запись, resolver и ожидаемый IP-адрес из условия упражнения."
		case CheckRoute, CheckReachability:
			return "Проверьте, что исходный узел исправен, каналы связи активны, а route включает ожидаемые узлы."
		case CheckFirewall, CheckSecurity:
			return "Проверьте firewall rules: нужный traffic должен быть разрешён, а запрещённый путь должен блокироваться."
		case CheckLB, CheckFailover:
			return "Проверьте пул серверов Load Balancer: выключенные и устаревшие ссылки на серверы не должны участвовать в выборе."
		case CheckLatency:
			return "Проверьте задержку на каналах связи и убедитесь, что итоговое время укладывается в цель упражнения."
		}
	}
	return "Сравните цель упражнения с текущими настройками связанных узлов и каналов связи."
}

func finalActionHint(quest Quest) string {
	if quest.Goal != "" {
		return "Сделайте ровно то, что описано в цели упражнения: " + normalizeHintText(quest.Goal)
	}
	return checkBasedHint(quest)
}

func normalizeLegacyHints(hints []string) []string {
	normalized := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint == "" {
			continue
		}
		normalized = append(normalized, normalizeHintText(hint))
	}
	return normalized
}

func normalizeTextList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, normalizeHintText(value))
	}
	return normalized
}

func normalizeCheckSpecs(checks []CheckSpec) []CheckSpec {
	normalized := make([]CheckSpec, 0, len(checks))
	for _, check := range checks {
		check.Title = normalizeHintText(check.Title)
		check.Message = normalizeHintText(check.Message)
		check.Hint = normalizeHintText(check.Hint)
		normalized = append(normalized, check)
	}
	return normalized
}

func normalizeHintActions(actions []string) []string {
	if len(actions) == 0 {
		return actions
	}
	normalized := make([]string, 0, len(actions))
	for _, action := range actions {
		normalized = append(normalized, normalizeHintText(action))
	}
	return normalized
}

func normalizeHintText(value string) string {
	replacer := strings.NewReplacer(
		"DNS node", "DNS-узел",
		"DNS record", "DNS-запись",
		"DNS resolver", "DNS-сервер",
		"Server nodes", "серверы",
		"Server node", "сервер",
		"node/link", "узел или канал связи",
		"node status", "состояние узла",
		"link status", "состояние канала связи",
		"status link", "состояние канала связи",
		"links", "каналы связи",
		"link", "канал связи",
		"Inspector", "инспектор",
		"Packet Inspector", "инспектор пакета",
		"Protocol Inspector", "протокольный разбор",
		"A record", "A-запись",
		"Record", "Запись",
		"Value", "Значение",
		"backend pool", "пул серверов",
		"Backend pool", "Пул серверов",
		"fallback backend", "резервный сервер",
		"skipped backends", "пропущенные серверы",
		"LB decision", "решение Load Balancer",
		"Backend", "Сервер приложения",
		"backend", "сервер приложения",
		"healthy backend", "исправный сервер",
		"healthy", "исправный",
		"reachable", "достижимый",
		"Down backend", "Выключенный сервер",
		"down", "выключен",
		"skippedBackends", "список пропущенных серверов",
		"stale backend", "устаревшая ссылка на сервер",
		"silently ignore", "молча игнорировать",
		"source node", "исходный узел",
		"source", "источник",
		"destination", "назначение",
		"packet loss", "потеря пакетов",
		"Packet", "Пакет",
		"packet", "пакет",
		"Simulation", "Симуляция",
		"simulation", "симуляция",
		"error", "ошибка",
		"failed event", "событие с ошибкой",
		"failed check", "проверку с ошибкой",
		"config", "настройки",
		"canvas", "рабочем поле",
		"latencyBreakdown", "разбор задержки",
		"Latency", "Задержка",
		"latency", "задержку",
		"Graph path", "Алгоритм пути по графу",
		"lowest-latency", "с наименьшей задержкой",
		"Fast-link", "Быстрый канал",
		"fast-link", "быстрый канал",
		"slow router", "медленный маршрутизатор",
		"Primary route", "Основной маршрут",
		"backup path", "резервный путь",
		"active path", "активный путь",
		"Direct Server-1 access", "Прямой доступ к Server-1",
		"Direct access", "Прямой доступ",
		"direct IP", "прямой IP-адрес",
		"Path", "Путь",
		"path", "путь",
		"route cost", "стоимость маршрута",
		"routing table", "таблицу маршрутизации",
		"routed path", "маршрут",
		"Default gateway", "Шлюз по умолчанию",
		"gateway", "шлюз",
		"Default route", "Маршрут по умолчанию",
		"defaultGateway", "шлюз по умолчанию",
		"Request", "Запрос",
		"request", "запрос",
		"TLS hostname", "имя TLS-сертификата",
		"hostname", "имя хоста",
		"backend-checker'ом", "серверной проверкой",
		"port", "порт",
		"protocol", "протокол",
	)
	return replacer.Replace(value)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func containsFold(value, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(value) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(value); i++ {
		if equalFoldASCII(value[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca := a[i]
		cb := b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func rawTopology(input string) json.RawMessage {
	return json.RawMessage(input)
}

func lbTopology(backends, server1Status, server2Status, firewallAction, dnsValue string) json.RawMessage {
	return rawTopology(`{
		"nodes":[
			{"id":"client-1","name":"Client","type":"client","status":"healthy","position":{"x":60,"y":250},"config":{"ip":"10.0.1.10","cidr":"10.0.1.10/24"}},
			{"id":"dns-1","name":"DNS","type":"dns","status":"healthy","position":{"x":160,"y":80},"config":{"ip":"10.0.1.53","records":[{"name":"api.netquest.local","type":"A","value":"` + dnsValue + `","ttl":300}]}},
			{"id":"router-1","name":"Router","type":"router","status":"healthy","position":{"x":250,"y":250},"config":{"ip":"10.0.1.1"}},
			{"id":"firewall-1","name":"Firewall","type":"firewall","status":"healthy","position":{"x":440,"y":250},"config":{"ip":"10.0.1.254","defaultPolicy":"deny","rules":[{"priority":100,"action":"` + firewallAction + `","protocol":"tcp","source":"10.0.1.0/24","destination":"10.0.2.10/32","port":443},{"priority":200,"action":"allow","protocol":"tcp","source":"0.0.0.0/0","destination":"0.0.0.0/0","port":443}]}},
			{"id":"lb-1","name":"Load Balancer","type":"load_balancer","status":"healthy","position":{"x":640,"y":250},"config":{"ip":"10.0.2.10","algorithm":"round_robin","autoDiscoverConnectedServers":false,"backends":` + backends + `}},
			{"id":"server-1","name":"Server 1","type":"server","status":"` + server1Status + `","position":{"x":850,"y":170},"config":{"ip":"10.0.2.21","port":443}},
			{"id":"server-2","name":"Server 2","type":"server","status":"` + server2Status + `","position":{"x":850,"y":330},"config":{"ip":"10.0.2.22","port":443}},
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
