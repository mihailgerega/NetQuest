package catalog

import "github.com/netquest/netquest/backend/internal/model"

// ensureMinimumProgressiveHints дополняет подсказки общими до minHints.
func ensureMinimumProgressiveHints(quest model.Quest, hints []model.ProgressiveHint, minHints int) []model.ProgressiveHint {
	for len(hints) < minHints {
		hints = append(hints, fallbackProgressiveHint(quest, len(hints), minHints))
	}
	return hints
}

// fallbackProgressiveHint — общая подсказка для позиции index: первая учит
// искать проверку с ошибкой, вторая — определить слой, последняя — почти решение,
// остальные — что изменить.
func fallbackProgressiveHint(quest model.Quest, index, minHints int) model.ProgressiveHint {
	switch index {
	case 0:
		return model.ProgressiveHint{
			Title: "Первый шаг",
			Body:  "Найдите проверку с ошибкой, затем проверьте состояние связанных узлов, состояние канала связи и настройки выбранного элемента.",
			Level: "guided",
			Actions: []string{
				"Откройте Timeline.",
				"Сравните первое событие с ошибкой с целью упражнения.",
			},
		}
	case 1:
		return model.ProgressiveHint{
			Title: "Определите слой",
			Body:  categoryLayerHint(quest.Category),
			Level: "guided",
			Actions: []string{
				"Откройте инспектор элемента из подсказки.",
				"Проверьте поля, которые влияют на этот слой.",
			},
		}
	case minHints - 1:
		return model.ProgressiveHint{
			Title: "Почти решение",
			Body:  finalActionHint(quest),
			Level: "action",
			Actions: []string{
				"Внесите изменение в topology.",
				"Запустите симуляцию и нажмите «Проверить решение».",
			},
		}
	default:
		return model.ProgressiveHint{
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

// categoryLayerHint — на каком слое сети сосредоточиться, по категории квеста.
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

// checkBasedHint — подсказка по первой проверке квеста: её собственная
// подсказка или общий совет для типа проверки.
func checkBasedHint(quest model.Quest) string {
	for _, check := range quest.ExpectedChecks {
		if check.Hint != "" {
			return normalizeHintText(check.Hint)
		}
		switch check.Type {
		case model.CheckDNS:
			return "Проверьте DNS-запись, resolver и ожидаемый IP-адрес из условия упражнения."
		case model.CheckRoute, model.CheckReachability:
			return "Проверьте, что исходный узел исправен, каналы связи активны, а route включает ожидаемые узлы."
		case model.CheckFirewall, model.CheckSecurity:
			return "Проверьте firewall rules: нужный traffic должен быть разрешён, а запрещённый путь должен блокироваться."
		case model.CheckLB, model.CheckFailover:
			return "Проверьте пул серверов Load Balancer: выключенные и устаревшие ссылки на серверы не должны участвовать в выборе."
		case model.CheckLatency:
			return "Проверьте задержку на каналах связи и убедитесь, что итоговое время укладывается в цель упражнения."
		}
	}
	return "Сравните цель упражнения с текущими настройками связанных узлов и каналов связи."
}

// finalActionHint — последняя подсказка: повторить цель квеста дословно.
func finalActionHint(quest model.Quest) string {
	if quest.Goal != "" {
		return "Сделайте ровно то, что описано в цели упражнения: " + normalizeHintText(quest.Goal)
	}
	return checkBasedHint(quest)
}
