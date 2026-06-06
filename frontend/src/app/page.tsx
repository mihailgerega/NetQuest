import { NetworkPreview } from "@/components/landing/network-preview";
import { Button } from "@/components/ui/button";
import { StatusPill } from "@/components/ui/status-pill";

const capabilities = [
  {
    title: "Собирать топологии",
    copy: "Добавляйте клиентов, серверы, DNS, маршрутизаторы, межсетевые экраны, балансировщики и каналы связи прямо на рабочем поле."
  },
  {
    title: "Запускать сценарии",
    copy: "Проверяйте DNS lookup, Ping, HTTPS-запросы и отказоустойчивость без доступа к настоящей инфраструктуре."
  },
  {
    title: "Диагностировать сбои",
    copy: "Timeline, инспектор пакета, протокольный разбор и Advisor показывают, где именно ломается путь запроса."
  },
  {
    title: "Учиться через упражнения",
    copy: "Quest Mode загружает сломанную сеть, выдаёт постепенные подсказки и проверяет решение на сервере."
  },
  {
    title: "Объяснять задержки",
    copy: "NetQuest считает виртуальное время по задержкам каналов, потерям пакетов, маршруту и обработке на узлах."
  }
];

const httpsSteps = [
  "Клиент выбирает домен и источник запроса.",
  "DNS-сервер возвращает IP-адрес назначения.",
  "Симулятор ищет активный путь от клиента до цели.",
  "Маршрутизатор выбирает следующий участок сети.",
  "Межсетевой экран проверяет протокол, порт и направление.",
  "TCP-модель добавляет время установления соединения.",
  "TLS-модель проверяет имя сертификата и добавляет задержку рукопожатия.",
  "Балансировщик выбирает доступный сервер из пула.",
  "Пакет проходит по выбранному пути до сервера.",
  "Инспекторы показывают итог, ошибки и вклад каждого этапа во время доставки."
];

const elementCards = [
  {
    title: "Client",
    what: "Источник виртуального запроса.",
    change: "IP-адрес, подсеть, шлюз по умолчанию и состояние узла.",
    impact: "Если клиент выключен или не имеет маршрута, пакет не сможет стартовать.",
    example: "Выберите Client-2 в верхней панели и запустите Ping от его имени."
  },
  {
    title: "Server",
    what: "Узел, который принимает Ping или HTTPS-запрос.",
    change: "IP-адрес, порт сервиса, имя сертификата и состояние.",
    impact: "Сервер должен быть доступен по активному пути и соответствовать ожидаемому порту.",
    example: "Сломайте Server-1 и проверьте, выберет ли балансировщик запасной сервер."
  },
  {
    title: "DNS",
    what: "Справочник доменных имён.",
    change: "A-записи, IP-адрес DNS-узла и состояние.",
    impact: "Неверная запись отправит HTTPS-запрос не к тому узлу или остановит сценарий.",
    example: "Добавьте запись api.netquest.local -> 10.0.2.10 для балансировщика."
  },
  {
    title: "Router",
    what: "Узел, который соединяет подсети и выбирает маршрут.",
    change: "Таблицу маршрутизации, шлюзы, стоимость маршрутов и состояние.",
    impact: "Ошибочный шлюз или выключенный канал приводит к событию route.not_found.",
    example: "Добавьте маршрут 0.0.0.0/0 через доступный gateway."
  },
  {
    title: "Firewall",
    what: "Контроль доступа между участками сети.",
    change: "Правила allow/deny, протоколы, порты, источники и назначения.",
    impact: "Правило deny выше allow может заблокировать HTTPS даже при рабочем маршруте.",
    example: "Разрешите tcp/443 от клиента к балансировщику."
  },
  {
    title: "Load Balancer",
    what: "Балансировщик, который выбирает сервер приложения.",
    change: "Алгоритм выбора, пул серверов, веса, включение серверов и автообнаружение.",
    impact: "Выключенные, удалённые и недостижимые серверы исключаются из выбора.",
    example: "Добавьте Server-3 в пул и убедитесь, что он участвует в failover."
  },
  {
    title: "Link",
    what: "Канал связи между двумя узлами.",
    change: "Состояние, задержку в миллисекундах и процент потери пакетов.",
    impact: "Задержка увеличивает timeline, а потеря пакетов может вызвать retry или ошибку.",
    example: "Поставьте 200 ms на канал к Server-2 и сравните totalLatencyMs."
  },
  {
    title: "Canvas",
    what: "Рабочее поле, где собирается схема сети.",
    change: "Положение узлов, связи, масштаб и центрирование.",
    impact: "Сохранённая топология уходит на backend и именно она проверяется симулятором.",
    example: "Нажмите «По размеру», чтобы увидеть все узлы после загрузки упражнения."
  }
];

const startSteps = [
  "Откройте демо или создайте локальный аккаунт.",
  "Загрузите demo topology или выберите упражнение.",
  "Проверьте источник запроса, домен и URL в верхней панели.",
  "Запустите DNS, Ping или HTTPS-сценарий.",
  "Откройте Timeline и инспекторы, чтобы увидеть причину результата.",
  "Измените узлы, каналы, DNS, firewall или пул серверов.",
  "Сохраните проект или отправьте решение упражнения на проверку."
];

export default function HomePage() {
  return (
    <main className="min-h-screen overflow-hidden bg-ink-950 text-white">
      <section className="relative min-h-[88vh] border-b border-white/10 px-6 pb-16 pt-6">
        <NetworkPreview />
        <div className="relative z-10 mx-auto flex w-full max-w-6xl flex-col gap-16">
          <nav className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-white/10 bg-ink-950/90 px-4 py-3 shadow-2xl shadow-black/30 backdrop-blur">
            <span className="font-bold text-white">NetQuest</span>
            <div className="flex flex-wrap items-center gap-2">
              <Button href="/login" variant="ghost" className="min-h-9 px-3">
                Войти
              </Button>
              <Button href="/dashboard" variant="secondary" className="min-h-9 px-3">
                Проекты
              </Button>
              <Button href="/quests" variant="secondary" className="min-h-9 px-3">
                Упражнения
              </Button>
            </div>
          </nav>

          <div className="max-w-4xl pt-12">
            <StatusPill status="running" label="учебный сетевой симулятор" />
            <h1 className="mt-7 max-w-4xl text-5xl font-black leading-[1.03] text-white md:text-7xl">
              NetQuest — визуальная лаборатория компьютерных сетей
            </h1>
            <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
              Собирайте сеть из узлов и каналов, запускайте виртуальные запросы и смотрите, почему пакет дошёл, задержался или завершился ошибкой. NetQuest не трогает настоящую сеть: все решения и задержки рассчитываются внутри модели.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Button href="/login">Открыть демо</Button>
              <Button href="/quests" variant="secondary">
                Перейти к упражнениям
              </Button>
              <Button href="/docs" variant="secondary">
                Документация
              </Button>
            </div>
          </div>
        </div>
      </section>

      <section className="bg-ink-900 px-6 py-16">
        <div className="mx-auto max-w-6xl">
          <p className="text-sm font-semibold text-signal-cyan">Что можно делать в NetQuest</p>
          <h2 className="mt-2 text-2xl font-bold">От схемы сети к объяснённому результату</h2>
          <div className="mt-8 grid gap-4 md:grid-cols-5">
            {capabilities.map((item) => (
              <article className="rounded-md border border-white/10 bg-white/[0.04] p-5" key={item.title}>
                <h3 className="font-bold">{item.title}</h3>
                <p className="mt-3 text-sm leading-6 text-slate-300">{item.copy}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-ink-950 px-6 py-16">
        <div className="mx-auto grid max-w-6xl gap-8 md:grid-cols-[0.85fr_1.15fr]">
          <div>
            <p className="text-sm font-semibold text-signal-cyan">Как проходит HTTPS-запрос</p>
            <h2 className="mt-2 text-2xl font-bold">Один запуск показывает весь путь по слоям</h2>
            <p className="mt-3 leading-7 text-slate-300">
              Timeline не просто перечисляет события. Он показывает, какой узел принял решение, сколько виртуального времени добавил этап и почему был выбран конкретный сервер.
            </p>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {httpsSteps.map((item, index) => (
              <div className="rounded-md border border-white/10 bg-white/[0.04] px-4 py-3 text-sm leading-6 text-slate-300" key={item}>
                <span className="mr-2 text-signal-cyan">{index + 1}.</span>
                {item}
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-ink-900 px-6 py-16">
        <div className="mx-auto max-w-6xl">
          <p className="text-sm font-semibold text-signal-cyan">Элементы сети</p>
          <h2 className="mt-2 text-2xl font-bold">Что делает каждый узел и как он влияет на симуляцию</h2>
          <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {elementCards.map((item) => (
              <article className="rounded-md border border-white/10 bg-white/[0.04] p-5" key={item.title}>
                <h3 className="text-base font-bold">{item.title}</h3>
                <dl className="mt-4 space-y-3 text-sm leading-6 text-slate-300">
                  <div>
                    <dt className="font-semibold text-white">Что это</dt>
                    <dd>{item.what}</dd>
                  </div>
                  <div>
                    <dt className="font-semibold text-white">Что можно менять</dt>
                    <dd>{item.change}</dd>
                  </div>
                  <div>
                    <dt className="font-semibold text-white">Как влияет</dt>
                    <dd>{item.impact}</dd>
                  </div>
                  <div>
                    <dt className="font-semibold text-white">Пример</dt>
                    <dd>{item.example}</dd>
                  </div>
                </dl>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-ink-950 px-6 py-16">
        <div className="mx-auto grid max-w-6xl gap-6 md:grid-cols-2">
          <article className="rounded-md border border-white/10 bg-white/[0.04] p-6">
            <p className="text-sm font-semibold text-signal-cyan">Как считается время</p>
            <h2 className="mt-2 text-2xl font-bold">Задержка складывается из пути и обработки</h2>
            <p className="mt-4 leading-7 text-slate-300">
              Каждый канал связи добавляет свою задержку. DNS, маршрутизация, firewall, TCP, TLS и балансировщик добавляют небольшое время обработки. Если на канале есть потеря пакетов, seed делает результат повторяемым: при одинаковой схеме и seed вы получите тот же исход.
            </p>
            <p className="mt-4 rounded-md border border-signal-cyan/30 bg-signal-cyan/10 px-4 py-3 text-sm leading-6 text-slate-200">
              Пример: путь Client → Router → Firewall → Load Balancer → Server с задержками 5 + 10 + 7 + 20 ms даёт примерно 42 ms в одну сторону, а HTTPS добавляет DNS, TCP/TLS и выбор сервера.
            </p>
          </article>
          <article className="rounded-md border border-white/10 bg-white/[0.04] p-6">
            <p className="text-sm font-semibold text-signal-green">Безопасно для настоящей сети</p>
            <h2 className="mt-2 text-2xl font-bold">NetQuest ничего не отправляет наружу</h2>
            <p className="mt-4 leading-7 text-slate-300">
              DNS lookup, Ping и HTTPS в NetQuest — это учебные сценарии. Они не создают реальные ICMP-пакеты, не открывают TCP-соединения и не обращаются к указанным доменам. Backend получает JSON-топологию, проверяет её и возвращает рассчитанные события.
            </p>
            <p className="mt-4 rounded-md border border-white/10 bg-ink-950 px-4 py-3 text-sm leading-6 text-slate-300">
              Поэтому можно спокойно ломать каналы, выключать серверы и менять правила доступа: изменения существуют только внутри проекта.
            </p>
          </article>
        </div>
      </section>

      <section className="bg-ink-900 px-6 py-16">
        <div className="mx-auto grid max-w-6xl gap-8 md:grid-cols-[0.85fr_1.15fr]">
          <div>
            <p className="text-sm font-semibold text-signal-cyan">Как начать</p>
            <h2 className="mt-2 text-2xl font-bold">Семь шагов до первой диагностики</h2>
            <p className="mt-3 leading-7 text-slate-300">
              Локальный аккаунт нужен для сохранения проектов и прогресса упражнений. Демо-вход подходит, если нужно быстро посмотреть приложение без настройки пользователя.
            </p>
          </div>
          <div className="space-y-3">
            {startSteps.map((item, index) => (
              <div className="rounded-md border border-white/10 bg-white/[0.04] px-4 py-3 text-sm leading-6 text-slate-300" key={item}>
                <span className="mr-2 text-signal-cyan">{index + 1}.</span>
                {item}
              </div>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}
