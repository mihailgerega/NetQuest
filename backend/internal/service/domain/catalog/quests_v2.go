package catalog

import "github.com/netquest/netquest/backend/internal/model"

// Квесты второй волны (ID с префиксом quest-v2-): те же слои, но с более
// жизненными поломками — выключенный резолвер, потери пакетов, устаревшие
// ссылки в пуле серверов, несколько проблем сразу.

// v2ClientSourceQuest — «Запрос от другого Client» (quest-v2-client-source).
func v2ClientSourceQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-client-source",
		Slug:               "zapros-ot-zdorovogo-client",
		Title:              "Запрос от другого Client",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "client2_ping_completed", Type: model.CheckReachability, Title: "Client-2 отправляет Ping", SourceNodeID: "client-2", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", Hint: "Выберите Client-2 и верните status healthy."},
		},
		Hints:            []string{"Источник запроса задаётся в верхней панели simulator.", "Если исходный узел выключен, пакет даже не стартует.", "Верните Client-2 в healthy и повторите Ping."},
		SuccessMessage:   "Client-2 снова может отправлять traffic.",
		FailureMessage:   "Ping от Client-2 всё ещё не проходит.",
		EstimatedMinutes: 5,
	}
}

// v2DNSResolverDownQuest — «Восстанови DNS-сервер» (quest-v2-dns-resolver-down).
func v2DNSResolverDownQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-dns-resolver-down",
		Slug:               "vosstanovi-dns-resolver",
		Title:              "Восстанови DNS-сервер",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "dns_lookup_completed", Type: model.CheckReachability, Title: "DNS Lookup проходит", SourceNodeID: "client-1", ScenarioType: "dns_lookup", Target: apiHostname, ExpectedStatus: "completed", Hint: "Проверьте status DNS-узла: он должен быть healthy."},
		},
		Hints:            []string{"Запись может быть правильной, но DNS-сервер всё равно недоступен.", "Откройте DNS-узел в инспекторе.", "Status DNS должен быть healthy."},
		SuccessMessage:   "DNS-сервер отвечает и возвращает A-запись.",
		FailureMessage:   "DNS Lookup всё ещё не завершается успешно.",
		EstimatedMinutes: 5,
	}
}

// v2PacketLossQuest — «Убери packet loss» (quest-v2-packet-loss).
func v2PacketLossQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-packet-loss",
		Slug:               "uberi-packet-loss",
		Title:              "Убери packet loss",
		Difficulty:         model.DifficultyEasy,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "ping_without_loss", Type: model.CheckReachability, Title: "Ping проходит без drop", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", Hint: "Откройте lossy-link и поставьте packetLossPercent = 0."},
		},
		Hints:            []string{"Active link не означает надёжный канал связи.", "Потеря пакетов на 100% гарантированно ломает симуляцию.", "Поставьте packetLossPercent в 0 и запустите Ping ещё раз."},
		SuccessMessage:   "Потеря пакетов устранена, Ping проходит.",
		FailureMessage:   "Traffic всё ещё теряется на пути.",
		EstimatedMinutes: 6,
	}
}

// v2LBAddFallbackQuest — «Добавь резервный сервер» (quest-v2-lb-add-fallback).
func v2LBAddFallbackQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-lb-add-fallback",
		Slug:               "dobav-fallback-backend",
		Title:              "Добавь резервный сервер",
		Difficulty:         model.DifficultyMedium,
		Category:           "Load Balancer",
		Description:        "Server-1 выключен, а Load Balancer знает только о нём. Рядом есть исправные Server-2 и Server-3.",
		Goal:               "Добавьте Server-2 или Server-3 в пул серверов, чтобы HTTPS-запрос прошёл через failover.",
		LearningObjectives: []string{"Настраивать пул серверов", "Понимать пропущенные серверы"},
		InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "allow", "10.0.2.10"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "fallback_backend_present", Type: model.CheckLB, Title: "В пуле есть резервный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}, Hint: "В инспекторе Load Balancer добавьте Server-2 или Server-3."},
			{ID: "https_failover_completed", Type: model.CheckFailover, Title: "Failover HTTPS проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
		},
		Hints:            []string{"Load Balancer выбирает только из пула серверов.", "Выключенный сервер должен попасть в список пропущенных серверов.", "Добавьте исправный сервер, у которого есть активный путь от LB."},
		SuccessMessage:   "Load Balancer использует исправный резервный сервер.",
		FailureMessage:   "Failover backend ещё не настроен.",
		EstimatedMinutes: 11,
	}
}

// v2DefaultGatewayQuest — «Настрой default gateway» (quest-v2-default-gateway).
func v2DefaultGatewayQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-default-gateway",
		Slug:               "nastroj-default-gateway",
		Title:              "Настрой default gateway",
		Difficulty:         model.DifficultyMedium,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "route_uses_gateway", Type: model.CheckRoute, Title: "Route идёт через Router", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-1"}, Hint: "Добавьте defaultGateway 10.0.1.1 в config Client."},
		},
		Hints:            []string{"Если destination не в subnet Client, нужен gateway.", "Gateway должен быть reachable через active link.", "Для этого topology gateway равен 10.0.1.1."},
		SuccessMessage:   "Шлюз по умолчанию настроен, маршрут найден.",
		FailureMessage:   "Client всё ещё не знает путь в subnet Server.",
		EstimatedMinutes: 10,
	}
}

// v2StaleLBBackendQuest — «Удали stale backend» (quest-v2-stale-lb-backend).
func v2StaleLBBackendQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-stale-lb-backend",
		Slug:               "udal-stale-backend",
		Title:              "Удали stale backend",
		Difficulty:         model.DifficultyMedium,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "lb_pool_has_server1", Type: model.CheckLB, Title: "Пул содержит Server-1", NodeID: "lb-1", RequiredBackends: []string{"server-1"}, Hint: "Удалите nodeId=server-old из пула."},
			{ID: "https_without_stale", Type: model.CheckReachability, Title: "HTTPS проходит без stale reference", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
		},
		Hints:            []string{"Stale backend — это ссылка на node, которого уже нет на canvas.", "Validator не должен silently ignore такие ссылки.", "Удалите server-old и оставьте существующий server-1."},
		SuccessMessage:   "Backend pool больше не содержит stale references.",
		FailureMessage:   "В пуле серверов всё ещё есть проблемная ссылка.",
		EstimatedMinutes: 12,
	}
}

// v2LatencyThresholdQuest — «Снизь latency до SLA» (quest-v2-latency-threshold).
func v2LatencyThresholdQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-latency-threshold",
		Slug:               "sniz-latency-do-sla",
		Title:              "Снизь latency до SLA",
		Difficulty:         model.DifficultyMedium,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "https_under_sla", Type: model.CheckLatency, Title: "HTTPS latency ниже 300ms", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed", MaxTotalLatencyMs: 300, MustExcludePath: []string{"router-slow"}, Hint: "Активируйте fast-link или уменьшите latency slow-link."},
		},
		Hints:            []string{"Смотрите разбор задержки в инспекторе пакета.", "Алгоритм пути по графу выбирает активный путь с наименьшей задержкой.", "Быстрый канал сейчас down, поэтому путь вынужден идти через медленный маршрутизатор."},
		SuccessMessage:   "Path укладывается в SLA.",
		FailureMessage:   "Latency всё ещё выше заданного порога.",
		EstimatedMinutes: 12,
	}
}

// v2BackupRouteHardQuest — «Резервный route после аварии» (quest-v2-backup-route-hard).
func v2BackupRouteHardQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-backup-route-hard",
		Slug:               "rezervnyj-route-hard",
		Title:              "Резервный route после аварии",
		Difficulty:         model.DifficultyHard,
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
		ExpectedChecks: []model.CheckSpec{
			{ID: "backup_route_uses_router_c", Type: model.CheckRoute, Title: "Backup route использует Router-C", SourceNodeID: "client-1", ScenarioType: "icmp_ping", Target: "server-1", ExpectedStatus: "completed", MustIncludePath: []string{"router-c"}, MustExcludePath: []string{"router-a"}, Hint: "Включите backup-link Client -> Router-C, primary-link оставьте down."},
		},
		Hints:            []string{"Primary route уже считается аварийным.", "Не надо чинить всё сразу: нужен рабочий backup path.", "Проверьте, что route.selected содержит router-c."},
		SuccessMessage:   "Backup route восстановлен.",
		FailureMessage:   "Backup route всё ещё не выбран.",
		EstimatedMinutes: 16,
	}
}

// v2SecureLBBoundaryQuest — «Закрой direct server access» (quest-v2-secure-lb-boundary).
func v2SecureLBBoundaryQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-secure-lb-boundary",
		Slug:               "zakroj-direct-server-hard",
		Title:              "Закрой direct server access",
		Difficulty:         model.DifficultyHard,
		Category:           "Security / Firewall",
		Description:        "Публичный HTTPS должен идти через Load Balancer. Direct access к Server-1 не должен проходить.",
		Goal:               "Сохраните HTTPS через LB, но запретите direct request к Server-1.",
		LearningObjectives: []string{"Строить security boundary", "Понимать порядок firewall rules"},
		InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true},{"nodeId":"server-2","enabled":true}]`, "healthy", "allow", "10.0.2.10"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "https_via_lb_still_works", Type: model.CheckReachability, Title: "HTTPS через LB проходит", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, ExpectedStatus: "completed"},
			{ID: "direct_server_blocked", Type: model.CheckSecurity, Title: "Direct Server-1 access заблокирован", SourceNodeID: "client-1", Target: "https://10.0.2.21/users", ExpectedStatus: "failed", ForbiddenTarget: "server-1", Hint: "Добавьте deny rule для Client -> Server-1 tcp/443 выше широкого allow."},
		},
		Hints:            []string{"Проверяйте два сценария: нормальный URL и direct IP.", "Правило deny должно матчить Server-1.", "Не сломайте path Client -> Load Balancer."},
		SuccessMessage:   "Security boundary настроен корректно.",
		FailureMessage:   "Direct access к backend всё ещё открыт или LB path сломан.",
		EstimatedMinutes: 18,
	}
}

// v2ProductionMultiIssueQuest — «Production API: несколько причин отказа» (quest-v2-production-multi-issue).
func v2ProductionMultiIssueQuest() model.Quest {
	return model.Quest{
		ID:                 "quest-v2-production-multi-issue",
		Slug:               "production-multi-issue",
		Title:              "Production API: несколько причин отказа",
		Difficulty:         model.DifficultyHard,
		Category:           "DNS + Firewall + Load Balancer",
		Description:        "Production API не открывается из-за неправильного DNS, deny rule и отсутствия исправного резервного сервера.",
		Goal:               "Почините DNS на LB, разрешите tcp/443 к LB и настройте failover на Server-2 или Server-3.",
		LearningObjectives: []string{"Диагностировать несколько слоёв", "Проверять решение серверной проверкой"},
		InitialTopology:    lbTopology(`[{"nodeId":"server-1","enabled":true}]`, "down", "deny", "10.0.2.99"),
		ExpectedChecks: []model.CheckSpec{
			{ID: "dns_points_to_lb", Type: model.CheckDNS, Title: "DNS указывает на LB", SourceNodeID: "client-1", Hostname: apiHostname, ExpectedIP: "10.0.2.10", Hint: "A record api.netquest.local должен быть 10.0.2.10."},
			{ID: "fallback_backend_exists", Type: model.CheckLB, Title: "Есть исправный резервный сервер", NodeID: "lb-1", AnyOfBackends: []string{"server-2", "server-3"}, Hint: "Server-1 выключен, добавьте Server-2 или Server-3 в пул."},
			{ID: "production_https_completed", Type: model.CheckFailover, Title: "HTTPS проходит с failover", SourceNodeID: "client-1", ScenarioType: "https_request", Target: apiURL, DownBackendID: "server-1", ExpectedStatus: "completed"},
		},
		Hints:            []string{"Идите по слоям: DNS -> Firewall -> Load Balancer.", "После каждой правки запускайте HTTPS и смотрите первый failed event.", "Решение должно выбрать backend не server-1."},
		SuccessMessage:   "Production API восстановлен на всех нужных слоях.",
		FailureMessage:   "Один из слоёв всё ещё блокирует request.",
		EstimatedMinutes: 24,
	}
}
