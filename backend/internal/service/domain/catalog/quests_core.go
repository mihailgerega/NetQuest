package catalog

import "github.com/netquest/netquest/backend/internal/model"

// Базовые квесты: по одному на ключевой слой сети — DNS, ICMP, firewall,
// Load Balancer, маршрутизация, задержка, failover и безопасность.

// dnsLookupQuest — «Почини DNS Lookup» (quest-dns-lookup).
func dnsLookupQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-dns-lookup",
		Slug:               "pochini-dns-lookup",
		Title:              "Почини DNS Lookup",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "dns_record_exists", Type: model.CheckDNS, Title: "DNS-запись настроена", SourceNodeID: "client-1", Hostname: apiHostname, ExpectedIP: "10.0.2.21", Hint: "Добавьте A-запись api.netquest.local -> 10.0.2.21."},
			{ID: "dns_lookup_completed", Type: model.CheckReachability, Title: "DNS Lookup проходит", SourceNodeID: "client-1", ScenarioType: "dns_lookup", Target: apiHostname, ExpectedStatus: "completed"},
		},
		Hints:            []string{"Откройте DNS-узел в инспекторе.", "Добавьте A-запись для api.netquest.local.", "Значение должно указывать на IP нужного Server."},
		SuccessMessage:   "DNS Lookup починен: hostname резолвится в правильный IP.",
		FailureMessage:   "DNS всё ещё не возвращает правильный A record.",
		EstimatedMinutes: 6,
	}
}

// allowPingQuest — «Разреши Ping» (quest-allow-ping).
func allowPingQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-allow-ping",
		Slug:               "razreshi-ping",
		Title:              "Разреши Ping",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "ping_completed", Type: model.CheckReachability, Title: "Ping completed", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed"},
			{ID: "route_includes_router", Type: model.CheckRoute, Title: "Route проходит через Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}},
		},
		Hints:            []string{"Проверьте status у links.", "Убедитесь, что Router соединён с Server.", "Ping использует RTT: путь туда и обратно."},
		SuccessMessage:   "Ping проходит, route найден.",
		FailureMessage:   "Ping всё ещё не проходит.",
		EstimatedMinutes: 5,
	}
}

// firewallHTTPSQuest — «Открой HTTPS на Firewall» (quest-firewall-https).
func firewallHTTPSQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-firewall-https",
		Slug:               "otkroy-https-na-firewall",
		Title:              "Открой HTTPS на Firewall",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "firewall_allows_https", Type: model.CheckFirewall, Title: "Firewall разрешает tcp/443", NodeID: "firewall-1", ExpectedAction: "allow", Protocol: "tcp", Port: 443, ExpectedIP: "10.0.2.21"},
			{ID: "https_completed", Type: model.CheckReachability, Title: "HTTPS completed", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
		},
		Hints:            []string{"Firewall проверяет protocol, source, destination и port.", "Для HTTPS нужен tcp/443.", "Добавьте allow rule выше deny/default policy."},
		SuccessMessage:   "HTTPS проходит через Firewall.",
		FailureMessage:   "Firewall всё ещё блокирует HTTPS.",
		EstimatedMinutes: 8,
	}
}

// lbBackendPoolQuest — «Настрой пул серверов Load Balancer» (quest-lb-backend-pool).
func lbBackendPoolQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-lb-backend-pool",
		Slug:               "nastroj-lb-backend-pool",
		Title:              "Настрой пул серверов Load Balancer",
		Difficulty:         model.DifficultyMedium,
		Category:           "Load Balancer",
		Description:        "DNS указывает на Load Balancer, но пул серверов пустой.",
		Goal:               "Добавьте Server-1 и Server-2 в пул серверов.",
		LearningObjectives: []string{"Понять пул серверов", "Понять решение Load Balancer"},
		InitialTopology:    lbTopology(`[]`, "healthy", "allow", "10.0.2.10"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "lb_pool_contains_servers", Type: model.CheckLB, Title: "Backend pool содержит Server-1/Server-2", NodeID: "lb-1", RequiredBackends: []string{"server-1", "server-2"}},
			{ID: "https_completed_via_lb", Type: model.CheckReachability, Title: "HTTPS проходит через LB", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
		},
		Hints:            []string{"Выберите Load Balancer.", "В инспекторе добавьте серверы в пул.", "Сервер должен быть исправен и достижим."},
		SuccessMessage:   "Load Balancer выбирает исправный сервер.",
		FailureMessage:   "Backend pool ещё настроен неверно.",
		EstimatedMinutes: 10,
	}
}

// defaultRouteQuest — «Почини default route» (quest-default-route).
func defaultRouteQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-default-route",
		Slug:               "pochini-default-route",
		Title:              "Почини default route",
		Difficulty:         model.DifficultyMedium,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "ping_uses_router", Type: model.CheckRoute, Title: "Route использует Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}},
		},
		Hints:            []string{"Проверьте gateway у Client.", "Проверьте routing table на Router.", "Default route обычно 0.0.0.0/0."},
		SuccessMessage:   "Routing между subnet работает.",
		FailureMessage:   "Route между subnet всё ещё не найден.",
		EstimatedMinutes: 12,
	}
}

// latencyDiagnosticsQuest — «Убери лишнюю latency» (quest-latency-diagnostics).
func latencyDiagnosticsQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-latency-diagnostics",
		Slug:               "uberi-lishnyuyu-latency",
		Title:              "Убери лишнюю latency",
		Difficulty:         model.DifficultyMedium,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "https_latency_below_300", Type: model.CheckLatency, Title: "HTTPS latency ниже 300ms", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MaxTotalLatencyMs: 300, MustExcludePath: []string{"router-a"}},
			{ID: "no_slow_selected_path", Type: model.CheckAdvisor, Title: "Advisor не видит slow path issue", ForbiddenIssueCode: "HIGH_LATENCY_LINK"},
		},
		Hints:            []string{"Откройте инспектор пакета и посмотрите расчёт задержки.", "Найдите канал связи с высокой задержкой.", "Включите быстрый канал или уменьшите задержку."},
		SuccessMessage:   "Запрос идёт быстрым path.",
		FailureMessage:   "Latency всё ещё слишком высокая.",
		EstimatedMinutes: 12,
	}
}

// backendFailoverQuest — «Failover backend после падения Server» (quest-backend-failover).
func backendFailoverQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-backend-failover",
		Slug:               "failover-backend-posle-padeniya-server",
		Title:              "Failover backend после падения Server",
		Difficulty:         model.DifficultyHard,
		Category:           "Failover / Load Balancer",
		Description:        "Server-1 выключен, но Load Balancer должен выбрать исправный сервер.",
		Goal:               "Настройте LB pool так, чтобы request уходил на Server-2 или Server-3.",
		LearningObjectives: []string{"Понять skippedBackends", "Понять failover"},
		InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "allow", "10.0.2.10"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "down_backend_skipped", Type: model.CheckFailover, Title: "Выключенный сервер пропущен", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
			{ID: "lb_has_healthy_pool", Type: model.CheckLB, Title: "Пул содержит исправный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}},
		},
		Hints:            []string{"Проверьте пул серверов.", "Выключенный сервер должен быть исключён из выбора.", "Добавьте Server-2 или Server-3 как исправный сервер."},
		SuccessMessage:   "Failover работает: down backend пропускается.",
		FailureMessage:   "Load Balancer всё ещё не имеет healthy failover backend.",
		EstimatedMinutes: 15,
	}
}

// denyDirectServerQuest — «Запрети direct access к Server» (quest-deny-direct-server).
func denyDirectServerQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-deny-direct-server",
		Slug:               "zapreti-direct-access-k-server",
		Title:              "Запрети direct access к Server",
		Difficulty:         model.DifficultyHard,
		Category:           "Security / Firewall / Load Balancer",
		Description:        "Client должен получать HTTPS только через Load Balancer, direct access к Server запрещён.",
		Goal:               "Разрешите Client -> Load Balancer tcp/443 и запретите Client -> Server tcp/443.",
		LearningObjectives: []string{"Понять security boundary", "Понять Firewall rule order"},
		InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]`, "healthy", "allow", "10.0.2.10"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "normal_https_completed", Type: model.CheckReachability, Title: "HTTPS через LB проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			{ID: "direct_server_denied", Type: model.CheckSecurity, Title: "Direct access к Server запрещён", SourceNodeID: "client-1", Target: "https://10.0.2.21/users", ExpectedStatus: "failed", ForbiddenTarget: "server-1"},
			{ID: "normal_path_includes_lb", Type: model.CheckRoute, Title: "Normal path включает Load Balancer", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MustIncludePath: []string{"lb-1"}},
		},
		Hints:            []string{"Разделяйте доступ к Load Balancer и Server.", "Firewall rule order имеет значение.", "LB должен иметь доступ к backend, Client — нет."},
		SuccessMessage:   "Direct access закрыт, доступ через LB работает.",
		FailureMessage:   "Security policy ещё неверная.",
		EstimatedMinutes: 18,
	}
}

// backupRouteQuest — «Резервный route после падения link» (quest-backup-route).
func backupRouteQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-backup-route",
		Slug:               "rezervnyj-route-posle-padeniya-link",
		Title:              "Резервный route после падения link",
		Difficulty:         model.DifficultyHard,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "backup_route_works", Type: model.CheckRoute, Title: "Backup route использует Router-C", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-c"}, MustExcludePath: []string{"router-a"}},
		},
		Hints:            []string{"Проверьте status link.", "Проверьте route cost.", "Резервный path должен быть reachable."},
		SuccessMessage:   "Backup route работает.",
		FailureMessage:   "Request всё ещё не использует резервный path.",
		EstimatedMinutes: 16,
	}
}

// productionAPIDiagnosticsQuest — «Комплексная диагностика Production API» (quest-production-api-diagnostics).
func productionAPIDiagnosticsQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-production-api-diagnostics",
		Slug:               "kompleksnaya-diagnostika-production-api",
		Title:              "Комплексная диагностика Production API",
		Difficulty:         model.DifficultyHard,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "client2_https_completed", Type: model.CheckReachability, Title: "Client-2 HTTPS completed", SourceNodeID: "client-2", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			{ID: "dns_to_lb", Type: model.CheckDNS, Title: "DNS указывает на Load Balancer", SourceNodeID: "client-2", Hostname: apiHostname, ExpectedIP: "10.0.2.10"},
			{ID: "server1_skipped", Type: model.CheckFailover, Title: "Server-1 выключен и пропущен", SourceNodeID: "client-2", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
		},
		Hints:            []string{"Начните с DNS.", "Затем проверьте Firewall.", "Потом проверьте имя TLS-сертификата.", "Последним проверьте пул серверов и health."},
		SuccessMessage:   "Production API восстановлен.",
		FailureMessage:   "В topology ещё есть блокирующая проблема.",
		EstimatedMinutes: 25,
	}
}
