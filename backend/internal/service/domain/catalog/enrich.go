package catalog

import "github.com/netquest/netquest/backend/internal/model"

// enrichQuestCatalog приводит каталог к виду, в котором его видит пользователь:
// тексты — к единой терминологии (normalizeHintText), подсказки — к прогрессивному
// формату, а недостающие объяснение решения, глоссарий и «зачем это в жизни»
// достраиваются по категории квеста.
func enrichQuestCatalog(quests []model.Quest) []model.Quest {
	enriched := make([]model.Quest, len(quests))
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

// normalizeProgressiveHints возвращает прогрессивные подсказки квеста: заданные
// явно — с нормализованными текстами, иначе собранные из обычных Hints
// (первая — «Где искать», последняя — «Конкретное действие»). Подсказок
// не меньше 4, у hard-квестов — не меньше 5: недостающие дополняются общими.
func normalizeProgressiveHints(quest model.Quest) []model.ProgressiveHint {
	minHints := 4
	if quest.Difficulty == model.DifficultyHard {
		minHints = 5
	}
	if len(quest.ProgressiveHints) > 0 {
		hints := make([]model.ProgressiveHint, 0, maxInt(minHints, len(quest.ProgressiveHints)))
		for _, hint := range quest.ProgressiveHints {
			hint.Body = normalizeHintText(hint.Body)
			hint.Actions = normalizeHintActions(hint.Actions)
			hints = append(hints, hint)
		}
		return ensureMinimumProgressiveHints(quest, hints, minHints)
	}
	hints := []model.ProgressiveHint{
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
		hints = append(hints, model.ProgressiveHint{
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

// defaultAfterSolution — объяснение после решения по умолчанию.
func defaultAfterSolution(quest model.Quest) string {
	return "После исправления «" + quest.Title + "» NetQuest подтверждает решение не по картинке на рабочем поле, а по поведению сети: сервер запускает симуляцию, проверяет ожидаемые условия и сравнивает фактический путь, статусы и решения узлов с целью упражнения."
}

// defaultGlossaryTerms — глоссарий по умолчанию: три общих термина и один
// термин категории.
func defaultGlossaryTerms(category string) []model.GlossaryTerm {
	terms := []model.GlossaryTerm{
		{Term: "Узел", Definition: "Элемент топологии: client, server, DNS, router, firewall или load balancer."},
		{Term: "Канал связи", Definition: "Связь между узлами. Состояние, задержка и потеря пакетов влияют на маршрут и виртуальное время."},
		{Term: "Timeline", Definition: "Список событий виртуального пакета: DNS, route, firewall, TCP/TLS, выбор сервера и delivery."},
	}
	switch {
	case containsFold(category, "DNS"):
		terms = append(terms, model.GlossaryTerm{Term: "A-запись", Definition: "DNS-запись, которая сопоставляет доменное имя с IPv4-адресом."})
	case containsFold(category, "Load Balancer"):
		terms = append(terms, model.GlossaryTerm{Term: "Пул серверов", Definition: "Список серверов, из которых Load Balancer выбирает доступную цель."})
	case containsFold(category, "Firewall"):
		terms = append(terms, model.GlossaryTerm{Term: "Правило firewall", Definition: "Правило allow/deny для протокола, порта, источника и назначения."})
	case containsFold(category, "Routing"):
		terms = append(terms, model.GlossaryTerm{Term: "Маршрут", Definition: "Путь пакета от исходного узла до цели через активные каналы связи."})
	case containsFold(category, "Latency"):
		terms = append(terms, model.GlossaryTerm{Term: "Задержка", Definition: "Виртуальное время канала связи или этапа, из которого складывается totalLatencyMs."})
	}
	return terms
}

// defaultRealWorldImportance — «почему это важно в реальной инфраструктуре»
// по категории квеста.
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
