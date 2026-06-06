import { Button } from "@/components/ui/button";

const repositoryUrl = process.env.NEXT_PUBLIC_REPOSITORY_URL;

const quickStart = [
  "Нажмите «Открыть демо» на странице входа или создайте локальный аккаунт.",
  "Откройте проект и нажмите «Загрузить demo topology».",
  "Выберите источник запроса, домен, URL и seed в верхней панели симулятора.",
  "Запустите DNS, Ping или HTTPS.",
  "Прочитайте Timeline: первое красное событие обычно показывает слой, где возникла проблема.",
  "Исправьте DNS, маршрут, firewall, состояние узла, канал связи или пул серверов.",
  "Сохраните проект либо нажмите «Проверить решение» в упражнении."
];

const docs = [
  {
    title: "Локальный аккаунт и демо-вход",
    copy: "Локальный аккаунт создаётся внутри вашей базы NetQuest. Он нужен для проектов, сохранённых версий топологии и прогресса упражнений. Демо-вход создаёт готового пользователя для быстрого просмотра приложения. Refresh token хранится в HttpOnly cookie, а frontend работает с access token."
  },
  {
    title: "Как открыть Quest Mode",
    copy: "Перейдите в «Упражнения» из верхней навигации, landing page или dashboard. Выберите карточку упражнения, прочитайте цель, откройте подсказки при необходимости и нажмите «Начать упражнение». Симулятор загрузит сломанную топологию автоматически."
  },
  {
    title: "Как работает проверка упражнений",
    copy: "Кнопка «Проверить решение» отправляет текущую топологию на backend. Проверяется не картинка на экране, а поведение: DNS-ответ, доступность, маршрут, firewall, выбор сервера балансировщиком, задержки и ожидаемые ошибки."
  },
  {
    title: "Как читать Timeline",
    copy: "Каждое событие имеет виртуальное время, тип, сообщение и прирост задержки относительно прошлого события. Если симуляция упала, Timeline всё равно показывает частичный путь и причину остановки."
  },
  {
    title: "Как читать инспектор пакета",
    copy: "Инспектор показывает статус, итоговую задержку, выбранный путь, DNS-результат, решение firewall, выбранный сервер, исключённые серверы и breakdown времени по этапам."
  },
  {
    title: "Почему это безопасно",
    copy: "NetQuest не отправляет настоящие сетевые пакеты. DNS lookup, Ping и HTTPS — виртуальные сценарии, рассчитанные по JSON-топологии, состояниям узлов, каналам связи и seed."
  }
];

const checklist = [
  "Есть ли DNS-запись для нужного домена.",
  "Доступен ли клиент, от имени которого запускается запрос.",
  "Есть ли активный маршрут до назначения.",
  "Не блокирует ли firewall нужный протокол и порт.",
  "Есть ли у балансировщика доступный сервер в пуле.",
  "Не выключен ли канал связи на выбранном пути.",
  "Не слишком ли велика задержка или потеря пакетов на каналах."
];

export default function DocsPage() {
  return (
    <main className="min-h-screen bg-ink-950 px-6 py-12 text-white">
      <div className="mx-auto max-w-6xl space-y-10">
        <header className="border-b border-white/10 pb-8">
          <p className="text-sm font-semibold text-signal-cyan">Документация NetQuest</p>
          <h1 className="mt-3 text-4xl font-black">Как пользоваться учебным сетевым симулятором</h1>
          <p className="mt-4 max-w-3xl leading-7 text-slate-300">
            NetQuest помогает разбирать сетевые сбои без доступа к настоящей инфраструктуре. Вы создаёте проект, собираете топологию, запускаете виртуальную симуляцию и смотрите, какие решения принял каждый элемент сети.
          </p>
          <div className="mt-6 flex flex-wrap gap-3">
            <Button href="/login">Войти или открыть демо</Button>
            <Button href="/quests" variant="secondary">
              Упражнения
            </Button>
            {repositoryUrl && (
              <Button href={repositoryUrl} variant="secondary">
                Код проекта
              </Button>
            )}
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-3">
          {docs.map((item) => (
            <article className="rounded-md border border-white/10 bg-white/[0.04] p-5" key={item.title}>
              <h2 className="font-bold">{item.title}</h2>
              <p className="mt-3 text-sm leading-6 text-slate-300">{item.copy}</p>
            </article>
          ))}
        </section>

        <section className="grid gap-6 md:grid-cols-[0.85fr_1.15fr]">
          <article className="rounded-md border border-white/10 bg-white/[0.04] p-5">
            <h2 className="font-bold">Быстрый старт</h2>
            <div className="mt-4 space-y-3 text-sm leading-6 text-slate-300">
              {quickStart.map((item, index) => (
                <p key={item}>
                  <span className="mr-2 text-signal-cyan">{index + 1}.</span>
                  {item}
                </p>
              ))}
            </div>
          </article>
          <article className="rounded-md border border-white/10 bg-white/[0.04] p-5">
            <h2 className="font-bold">Что проверять при первой ошибке</h2>
            <div className="mt-4 grid gap-2 text-sm leading-6 text-slate-300 md:grid-cols-2">
              {checklist.map((item, index) => (
                <p key={item}>
                  <span className="mr-2 text-signal-cyan">{index + 1}.</span>
                  {item}
                </p>
              ))}
            </div>
          </article>
        </section>

        <section className="rounded-md border border-signal-cyan/25 bg-signal-cyan/10 p-5">
          <h2 className="font-bold">Как считается виртуальное время</h2>
          <p className="mt-3 text-sm leading-6 text-slate-200">
            Время в Timeline не является реальным измерением сети. Симулятор складывает задержки каналов на выбранном пути, небольшие задержки обработки на DNS, Router, Firewall и Load Balancer, время TCP/TLS-модели и возможную задержку retry при потере пакета. Seed делает такие сценарии повторяемыми.
          </p>
        </section>
      </div>
    </main>
  );
}
