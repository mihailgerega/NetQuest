"use client";

import { MouseEvent, WheelEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { StatusPill } from "@/components/ui/status-pill";
import type { AdvisorIssue, LoadBalancerBackend, Quest, QuestAttempt, QuestCheckResult, SimulationEvent, SimulationSummary, TopologyDocument, TopologyLink, TopologyNode } from "@/lib/api/types";
import { authedClient, getAccessToken, getStoredUser } from "@/lib/auth";
import { demoTopology, topologySchema } from "@/lib/topology";

const nodeTypes: TopologyNode["type"][] = ["client", "dns", "router", "firewall", "load_balancer", "server"];
const nodeTypeLabels: Record<TopologyNode["type"], string> = {
  client: "Client",
  dns: "DNS",
  router: "Router",
  firewall: "Firewall",
  load_balancer: "Load Balancer",
  server: "Server"
};

const eventTypeLabels: Record<string, string> = {
  "simulation.started": "симуляция запущена",
  "topology.validated": "топология проверена",
  "packet.created": "пакет создан",
  "dns.query": "DNS запрос",
  "dns.response": "DNS ответ",
  "dns.error": "DNS ошибка",
  "route.selected": "маршрут выбран",
  "route.not_found": "маршрут не найден",
  "firewall.decision": "решение Firewall",
  "firewall.denied": "Firewall отклонил пакет",
  "tcp.handshake.start": "TCP handshake начат",
  "tcp.syn": "TCP SYN",
  "tcp.syn_ack": "TCP SYN-ACK",
  "tcp.ack": "TCP ACK",
  "tcp.handshake.done": "TCP handshake завершён",
  "tls.handshake.start": "TLS handshake начат",
  "tls.client_hello": "TLS ClientHello",
  "tls.server_hello": "TLS ServerHello",
  "tls.certificate.validated": "TLS certificate проверен",
  "tls.handshake.done": "TLS handshake завершён",
  "lb.backend.discovered": "Load Balancer обнаружил сервер",
  "lb.backend.selected": "Load Balancer выбрал сервер",
  "lb.backend.unhealthy": "сервер недоступен",
  "packet.forwarded": "пакет переслан",
  "packet.dropped": "пакет потерян",
  "packet.delivered": "пакет доставлен",
  "failover.triggered": "failover запущен",
  "failover.route_changed": "маршрут пересчитан",
  "simulation.completed": "симуляция завершена",
  "simulation.failed": "симуляция завершилась ошибкой"
};

const DEFAULT_DNS_HOSTNAME = "api.netquest.local";
const DEFAULT_HTTPS_URL = "https://api.netquest.local/users";

const paletteGroups: Array<{ id: string; title: string; items: Array<{ type: TopologyNode["type"]; description: string }> }> = [
  { id: "endpoint", title: "Конечные узлы", items: [{ type: "client", description: "Источник DNS, Ping или HTTPS-запроса" }, { type: "server", description: "Принимает Ping или HTTPS-запрос" }] },
  { id: "network", title: "Маршрутизация", items: [{ type: "router", description: "Выбирает маршрут между подсетями" }] },
  { id: "security", title: "Безопасность", items: [{ type: "firewall", description: "Разрешает или блокирует пакет" }] },
  { id: "delivery", title: "Балансировка", items: [{ type: "load_balancer", description: "Выбирает сервер для запроса" }] },
  { id: "infrastructure", title: "Инфраструктура", items: [{ type: "dns", description: "Преобразует доменное имя в IP" }] }
];

type ScenarioType = "dns_lookup" | "icmp_ping" | "https_request" | "failover_demo";
type SaveState = "unsaved" | "saving" | "saved" | "running";
type ProtocolTab = "summary" | "dns" | "routing" | "firewall" | "tcp" | "tls" | "loadBalancer" | "errors";

type LayoutState = {
  left: boolean;
  right: boolean;
  timeline: boolean;
  packet: boolean;
  protocol: boolean;
  advisor: boolean;
  quest: boolean;
};

type CanvasViewport = {
  x: number;
  y: number;
  zoom: number;
};

const defaultLayout: LayoutState = { left: true, right: true, timeline: true, packet: true, protocol: true, advisor: true, quest: true };

function loadLayoutState(): LayoutState {
  if (typeof window === "undefined") return defaultLayout;
  try {
    const raw = window.localStorage.getItem("netquest.simulator.layout");
    if (!raw) return defaultLayout;
    const parsed = JSON.parse(raw) as Partial<LayoutState>;
    return { ...defaultLayout, ...parsed };
  } catch {
    return defaultLayout;
  }
}

function loadCanvasViewport(): CanvasViewport {
  if (typeof window === "undefined") return { x: 0, y: 0, zoom: 1 };
  try {
    const raw = window.localStorage.getItem("netquest.simulator.viewport");
    if (!raw) return { x: 0, y: 0, zoom: 1 };
    const parsed = JSON.parse(raw) as Partial<CanvasViewport>;
    return {
      x: typeof parsed.x === "number" ? parsed.x : 0,
      y: typeof parsed.y === "number" ? parsed.y : 0,
      zoom: typeof parsed.zoom === "number" ? clamp(parsed.zoom, 0.35, 2.2) : 1
    };
  } catch {
    return { x: 0, y: 0, zoom: 1 };
  }
}

export default function SimulatorPage() {
  const router = useRouter();
  const api = useMemo(() => authedClient(), []);
  const canvasRef = useRef<HTMLElement | null>(null);
  const initialFitDoneRef = useRef(false);
  const [projectId, setProjectId] = useState("");
  const [topologyId, setTopologyId] = useState("");
  const [topologyVersion, setTopologyVersion] = useState<number | null>(null);
  const [saveState, setSaveState] = useState<SaveState>("unsaved");
  const [topology, setTopology] = useState<TopologyDocument>(() => demoTopology());
  const [selectedNodeId, setSelectedNodeId] = useState("client-1");
  const [selectedLinkId, setSelectedLinkId] = useState("");
  const [selectedSourceNodeId, setSelectedSourceNodeId] = useState("client-1");
  const [pingTargetNodeId, setPingTargetNodeId] = useState("server-1");
  const [dnsHostname, setDnsHostname] = useState(DEFAULT_DNS_HOSTNAME);
  const [httpsUrl, setHttpsUrl] = useState(DEFAULT_HTTPS_URL);
  const [seed, setSeed] = useState(7);
  const [fixedSeed, setFixedSeed] = useState(true);
  const [showLatencyHelp, setShowLatencyHelp] = useState(false);
  const [connectFrom, setConnectFrom] = useState("");
  const [backendToAdd, setBackendToAdd] = useState("");
  const [selectedBackendId, setSelectedBackendId] = useState("");
  const [dragging, setDragging] = useState<{ id: string; dx: number; dy: number } | null>(null);
  const [panning, setPanning] = useState<{ startX: number; startY: number; originX: number; originY: number } | null>(null);
  const [canvasViewport, setCanvasViewport] = useState<CanvasViewport>(() => loadCanvasViewport());
  const [events, setEvents] = useState<SimulationEvent[]>([]);
  const [summary, setSummary] = useState<SimulationSummary | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [wsStatus, setWsStatus] = useState("idle");
  const [quest, setQuest] = useState<Quest | null>(null);
  const [questAttempt, setQuestAttempt] = useState<QuestAttempt | null>(null);
  const [questResult, setQuestResult] = useState<QuestCheckResult | null>(null);
  const [revealedQuestHintsCount, setRevealedQuestHintsCount] = useState(0);
  const [advisorIssues, setAdvisorIssues] = useState<AdvisorIssue[]>([]);
  const [activeProtocolTab, setActiveProtocolTab] = useState<ProtocolTab>("summary");
  const [layout, setLayout] = useState<LayoutState>(() => loadLayoutState());
  const [paletteSearch, setPaletteSearch] = useState("");
  const [collapsedPaletteGroups, setCollapsedPaletteGroups] = useState<Record<string, boolean>>({});

  const fitCanvasToDocument = useCallback((document: TopologyDocument) => {
    const rect = canvasRef.current?.getBoundingClientRect();
    if (!rect || document.nodes.length === 0) {
      setCanvasViewport({ x: 0, y: 0, zoom: 1 });
      return;
    }
    const nodeWidth = 112;
    const nodeHeight = 56;
    const padding = 72;
    const xs = document.nodes.map((node) => node.position?.x ?? 0);
    const ys = document.nodes.map((node) => node.position?.y ?? 0);
    const minX = Math.min(...xs);
    const minY = Math.min(...ys);
    const maxX = Math.max(...xs) + nodeWidth;
    const maxY = Math.max(...ys) + nodeHeight;
    const width = Math.max(maxX - minX, nodeWidth);
    const height = Math.max(maxY - minY, nodeHeight);
    const zoom = clamp(Math.min((rect.width - padding * 2) / width, (rect.height - padding * 2) / height), 0.35, 1.35);
    setCanvasViewport({
      zoom,
      x: (rect.width - width * zoom) / 2 - minX * zoom,
      y: (rect.height - height * zoom) / 2 - minY * zoom
    });
  }, []);

  const loadQuestContext = useCallback(async (questId: string, attemptId: string) => {
    try {
      const cached = window.localStorage.getItem(`netquest.quest.${attemptId}`);
      if (cached) {
        const parsed = JSON.parse(cached) as { quest?: Quest; attempt?: QuestAttempt };
        if (parsed.quest) {
          setQuest(parsed.quest);
          setTopology(parsed.quest.initialTopology);
          requestAnimationFrame(() => fitCanvasToDocument(parsed.quest!.initialTopology));
          setSelectedNodeId(parsed.quest.initialTopology.nodes[0]?.id ?? "");
          const firstClient = parsed.quest.initialTopology.nodes.find((node) => node.type === "client");
          if (firstClient) setSelectedSourceNodeId(firstClient.id);
        }
        if (parsed.attempt) {
          setQuestAttempt(parsed.attempt);
          setRevealedQuestHintsCount(Math.max(parsed.attempt.revealedHintsCount ?? 0, storedQuestHintCount(attemptId)));
        }
        setSaveState("unsaved");
        setMessage("Топология упражнения загружена. Исправьте её и нажмите «Проверить решение».");
        return;
      }
      const [questRes, attemptRes] = await Promise.all([api.getQuest(questId), api.getQuestAttempt(attemptId)]);
      setQuest(questRes.quest);
      setQuestAttempt(attemptRes.attempt);
      setRevealedQuestHintsCount(Math.max(attemptRes.attempt.revealedHintsCount ?? 0, storedQuestHintCount(attemptId)));
      setTopology(questRes.quest.initialTopology);
      requestAnimationFrame(() => fitCanvasToDocument(questRes.quest.initialTopology));
      const firstClient = questRes.quest.initialTopology.nodes.find((node) => node.type === "client");
      if (firstClient) setSelectedSourceNodeId(firstClient.id);
      setSaveState("unsaved");
    } catch (error) {
      setMessage(userFacingError(error, "Не удалось загрузить упражнение."));
    }
  }, [api, fitCanvasToDocument]);

  useEffect(() => {
    if (!getStoredUser()) {
      router.push("/login");
      return;
    }
    const params = new URLSearchParams(window.location.search);
    setProjectId(params.get("projectId") ?? "");
    const questId = params.get("questId");
    const attemptId = params.get("attemptId");
    if (questId && attemptId) {
      void loadQuestContext(questId, attemptId);
    }
  }, [loadQuestContext, router]);

  useEffect(() => {
    window.localStorage.setItem("netquest.simulator.layout", JSON.stringify(layout));
  }, [layout]);

  useEffect(() => {
    window.localStorage.setItem("netquest.simulator.viewport", JSON.stringify(canvasViewport));
  }, [canvasViewport]);

  useEffect(() => {
    if (initialFitDoneRef.current) return;
    initialFitDoneRef.current = true;
    requestAnimationFrame(() => fitCanvasToDocument(topology));
  }, [fitCanvasToDocument, topology]);

  const clientNodes = useMemo(() => topology.nodes.filter((node) => node.type === "client"), [topology.nodes]);
  const selectedSourceNode = clientNodes.find((node) => node.id === selectedSourceNodeId);
  const selectedNode = topology.nodes.find((node) => node.id === selectedNodeId);
  const selectedLink = topology.links.find((link) => link.id === selectedLinkId);
  const pingTargetOptions = useMemo(() => topology.nodes.filter((node) => node.id !== selectedSourceNodeId), [topology.nodes, selectedSourceNodeId]);
  const selectedPingTargetNode = topology.nodes.find((node) => node.id === pingTargetNodeId);
  const activePath = summary?.path ?? [];
  const selectedBackendNodeId = summary?.selectedBackendNodeId ?? summary?.selectedBackend;
  const skippedBackendIds = useMemo(() => new Set((summary?.skippedBackends ?? []).map((backend) => backend.nodeId)), [summary?.skippedBackends]);
  const selectedLBBackends = useMemo(() => (selectedNode?.type === "load_balancer" ? getBackends(selectedNode) : []), [selectedNode]);
  const selectedLBBackendIds = useMemo(() => new Set(selectedLBBackends.map((backend) => backend.nodeId)), [selectedLBBackends]);
  const protocolDetails = (summary?.protocolDetails ?? {}) as Record<string, unknown>;
  const filteredPaletteGroups = useMemo(() => {
    const query = paletteSearch.trim().toLowerCase();
    return paletteGroups
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => {
          const label = nodeTypeLabels[item.type].toLowerCase();
          return !query || label.includes(query) || item.type.includes(query) || item.description.toLowerCase().includes(query);
        })
      }))
      .filter((group) => group.items.length > 0);
  }, [paletteSearch]);
  const addableBackends =
    selectedNode?.type === "load_balancer"
      ? topology.nodes.filter((node) => node.type === "server" && !selectedLBBackendIds.has(node.id))
      : [];
  const sourceIsDown = Boolean(selectedSourceNode && getNodeStatus(selectedSourceNode) === "down");
  const sourceProblem =
    clientNodes.length === 0
      ? "Добавьте хотя бы один Client, чтобы запустить симуляцию."
      : !selectedSourceNode
        ? "Выберите Client, от которого нужно отправить запрос."
        : sourceIsDown
          ? "Выбранный Client недоступен. Восстановите его или выберите другой источник."
          : "";
  const sourceNodeForSummary = summary?.sourceNodeId ? topology.nodes.find((node) => node.id === summary.sourceNodeId) : undefined;

  useEffect(() => {
    setSelectedSourceNodeId((current) => {
      if (clientNodes.length === 0) return "";
      if (clientNodes.some((node) => node.id === current)) return current;
      return (clientNodes.find((node) => getNodeStatus(node) !== "down") ?? clientNodes[0]).id;
    });
  }, [clientNodes]);

  useEffect(() => {
    setPingTargetNodeId((current) => {
      const options = topology.nodes.filter((node) => node.id !== selectedSourceNodeId);
      if (options.length === 0) return "";
      if (options.some((node) => node.id === current)) return current;
      return (options.find((node) => node.type === "server") ?? options[0]).id;
    });
  }, [selectedSourceNodeId, topology.nodes]);

  useEffect(() => {
    if (selectedNode?.type !== "load_balancer") {
      setSelectedBackendId("");
      setBackendToAdd("");
      return;
    }
    if (selectedBackendId && !selectedLBBackendIds.has(selectedBackendId)) {
      setSelectedBackendId("");
    }
  }, [selectedBackendId, selectedLBBackendIds, selectedNode?.type]);

  async function ensureProject() {
    if (projectId) return projectId;
    const res = await api.createProject({ name: "NetQuest Demo Lab", visibility: "private" });
    setProjectId(res.project.id);
    window.history.replaceState(null, "", `/simulator?projectId=${res.project.id}`);
    return res.project.id;
  }

  async function saveTopology() {
    setMessage(null);
    setSaveState("saving");
    try {
      const parsed = topologySchema.safeParse(topology);
      if (!parsed.success) {
        setSaveState("unsaved");
        setMessage(parsed.error.issues[0]?.message ?? "Topology не прошла локальную проверку.");
        return null;
      }
      const pid = await ensureProject();
      const res = await api.createTopology(pid, { name: "Демо-топология", data: topology });
      setTopologyId(res.topology.id);
      setTopologyVersion(res.topology.version);
      setSaveState("saved");
      setMessage(`Топология сохранена как версия ${res.topology.version}`);
      return { id: res.topology.id, version: res.topology.version };
    } catch (error) {
      setSaveState("unsaved");
      setMessage(userFacingError(error, "Не удалось сохранить топологию."));
      return null;
    }
  }

  async function runScenario(type: ScenarioType) {
    const validationMessage = scenarioValidationMessage(type, selectedSourceNode, selectedPingTargetNode, dnsHostname, httpsUrl);
    if (validationMessage) {
      setMessage(validationMessage);
      return;
    }

    setBusy(true);
    setMessage(null);
    try {
      const pid = await ensureProject();
      const saved = await saveTopology();
      if (!saved) return;

      const runSeed = fixedSeed ? normalizeSeed(seed) : randomSeed();
      if (!fixedSeed) setSeed(runSeed);
      const target = scenarioTarget(type, dnsHostname, pingTargetNodeId, httpsUrl);
      setSaveState("running");
      setEvents([]);
      setSummary(null);
      const res = await api.startSimulation({
        projectId: pid,
        topologyId: saved.id,
        scenario: {
          type,
          sourceNodeId: selectedSourceNode!.id,
          target,
          method: "GET"
        },
        seed: runSeed
      });
      setEvents(res.events);
      setSummary(res.summary);
      setActiveProtocolTab("summary");
      openWebSocket(String((res.simulation as { id?: string }).id ?? ""));
      setMessage(res.summary.status === "failed" ? translateSimulationError(res.summary.errors[0]) : "Simulation завершена.");
      setSaveState("saved");
    } catch (error) {
      setMessage(userFacingError(error, "Simulation завершилась ошибкой."));
      setSaveState(topologyId ? "saved" : "unsaved");
    } finally {
      setBusy(false);
    }
  }

  async function checkQuestSolution() {
    if (!questAttempt) {
      setMessage("Сначала откройте упражнение из Quest Mode.");
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.checkQuestAttempt(questAttempt.id, { topology, seed: normalizeSeed(seed) });
      setQuestAttempt(res.attempt);
      setQuestResult(res.result);
      if (quest) {
        cacheQuestContext(res.attempt.id, quest, res.attempt);
      }
      setMessage(res.result.passed ? quest?.successMessage ?? "Упражнение выполнено." : quest?.failureMessage ?? "Условия ещё не выполнены.");
    } catch (error) {
      setMessage(userFacingError(error, "Не удалось проверить решение."));
    } finally {
      setBusy(false);
    }
  }

  async function resetQuest() {
    if (!questAttempt) return;
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.resetQuestAttempt(questAttempt.id);
      setQuest(res.quest);
      setQuestAttempt(res.attempt);
      setQuestResult(null);
      setRevealedQuestHintsCount(0);
      clearStoredQuestHintCount(res.attempt.id);
      cacheQuestContext(res.attempt.id, res.quest, res.attempt);
      setTopology(res.quest.initialTopology);
      requestAnimationFrame(() => fitCanvasToDocument(res.quest.initialTopology));
      const firstClient = res.quest.initialTopology.nodes.find((node) => node.type === "client");
      if (firstClient) setSelectedSourceNodeId(firstClient.id);
      setSaveState("unsaved");
      setMessage("Упражнение сброшено.");
    } catch (error) {
      setMessage(userFacingError(error, "Не удалось сбросить упражнение."));
    } finally {
      setBusy(false);
    }
  }

  async function revealNextQuestHint() {
    if (!quest || !questAttempt) return;
    const totalHints = quest.progressiveHints?.length ?? 0;
    if (totalHints === 0 || revealedQuestHintsCount >= totalHints) return;
    const nextCount = Math.min(revealedQuestHintsCount + 1, totalHints);
    setRevealedQuestHintsCount(nextCount);
    storeQuestHintCount(questAttempt.id, nextCount);
    try {
      const res = await api.revealQuestHint(questAttempt.id, { revealedHintsCount: nextCount });
      setQuestAttempt(res.attempt);
      setRevealedQuestHintsCount(res.attempt.revealedHintsCount ?? nextCount);
      storeQuestHintCount(res.attempt.id, res.attempt.revealedHintsCount ?? nextCount);
      cacheQuestContext(res.attempt.id, quest, res.attempt);
    } catch (error) {
      setMessage(userFacingError(error, "Подсказка открыта локально, но backend не сохранил это состояние."));
    }
  }

  async function analyzeCurrentTopology() {
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.analyzeTopology({
        topology,
        scenario: selectedSourceNode ? { type: "https_request", sourceNodeId: selectedSourceNode.id, target: httpsUrl, method: "GET" } : undefined
      });
      setAdvisorIssues(res.issues);
      setLayout((current) => ({ ...current, right: true, advisor: true }));
      setMessage(res.issues.length ? `Validation Advisor нашёл ${res.issues.length} issue.` : "Validation Advisor не нашёл проблем.");
    } catch (error) {
      setMessage(userFacingError(error, "Не удалось проверить топологию."));
    } finally {
      setBusy(false);
    }
  }

  function toggleLayout(key: keyof LayoutState) {
    setLayout((current) => ({ ...current, [key]: !current[key] }));
  }

  function focusAdvisorIssue(issue: AdvisorIssue) {
    if (issue.affectedNodeId) {
      setSelectedNodeId(issue.affectedNodeId);
      setSelectedLinkId("");
    }
    if (issue.affectedLinkId) {
      setSelectedLinkId(issue.affectedLinkId);
      setSelectedNodeId("");
    }
  }

  function openWebSocket(simulationId: string) {
    const token = getAccessToken();
    if (!simulationId || !token) return;
    try {
      const ws = new WebSocket(api.wsUrl(simulationId, token));
      setWsStatus("connecting");
      ws.onopen = () => setWsStatus("streaming");
      ws.onmessage = (event) => {
        const payload = JSON.parse(event.data) as { event?: SimulationEvent };
        if (payload.event) {
          setEvents((items) => (items.some((item) => item.id === payload.event?.id) ? items : [...items, payload.event as SimulationEvent]));
        }
      };
      ws.onerror = () => setWsStatus("polling fallback");
      ws.onclose = () => setWsStatus("idle");
    } catch {
      setWsStatus("polling fallback");
    }
  }

  function mutateTopology(updater: (current: TopologyDocument) => TopologyDocument, statusMessage?: string) {
    setTopology((current) => updater(current));
    setSaveState("unsaved");
    if (statusMessage) setMessage(statusMessage);
  }

  function resetDemo() {
    const nextTopology = demoTopology();
    setTopology(nextTopology);
    requestAnimationFrame(() => fitCanvasToDocument(nextTopology));
    setTopologyId("");
    setTopologyVersion(null);
    setSaveState("unsaved");
    setSelectedNodeId("client-1");
    setSelectedSourceNodeId("client-1");
    setPingTargetNodeId("server-1");
    setSelectedLinkId("");
    setSummary(null);
    setEvents([]);
    setQuest(null);
    setQuestAttempt(null);
    setQuestResult(null);
    setRevealedQuestHintsCount(0);
    setAdvisorIssues([]);
    setMessage("Демо-топология загружена заново.");
  }

  function addNode(type: TopologyNode["type"]) {
    const id = nextNodeId(topology, type);
    const node: TopologyNode = {
      id,
      type,
      name: nodeTypeLabels[type],
      status: "healthy",
      position: { x: 160 + topology.nodes.length * 28, y: 120 + topology.nodes.length * 18 },
      config: defaultNodeConfig(type, topology.nodes.length, id)
    };
    mutateTopology((current) => ({ ...current, nodes: [...current.nodes, node] }));
    setSelectedNodeId(id);
    setSelectedLinkId("");
  }

  function updateNode(id: string, patch: Partial<TopologyNode>) {
    mutateTopology((current) => ({ ...current, nodes: current.nodes.map((node) => (node.id === id ? { ...node, ...patch } : node)) }));
  }

  function updateNodeConfig(id: string, key: string, value: unknown) {
    const node = topology.nodes.find((item) => item.id === id);
    if (!node) return;
    updateNode(id, { config: { ...(node.config ?? {}), [key]: value } });
  }

  function updateLink(id: string, patch: Partial<TopologyLink>) {
    mutateTopology((current) => ({ ...current, links: current.links.map((link) => (link.id === id ? { ...link, ...patch } : link)) }));
  }

  function updateRoute(nodeId: string, index: number, key: string, value: unknown) {
    const node = topology.nodes.find((item) => item.id === nodeId);
    if (!node) return;
    const routes = Array.isArray(node.config?.routes) ? [...(node.config?.routes as Record<string, unknown>[])] : [];
    routes[index] = { ...(routes[index] ?? {}), [key]: value };
    updateNodeConfig(nodeId, "routes", routes);
  }

  function addRoute(nodeId: string) {
    const node = topology.nodes.find((item) => item.id === nodeId);
    if (!node) return;
    const routes = Array.isArray(node.config?.routes) ? [...(node.config?.routes as Record<string, unknown>[])] : [];
    routes.push({ destination: "0.0.0.0/0", gateway: "", interface: "eth0", metric: 100 });
    updateNodeConfig(nodeId, "routes", routes);
  }

  function removeRoute(nodeId: string, index: number) {
    const node = topology.nodes.find((item) => item.id === nodeId);
    if (!node) return;
    const routes = Array.isArray(node.config?.routes) ? [...(node.config?.routes as Record<string, unknown>[])] : [];
    routes.splice(index, 1);
    updateNodeConfig(nodeId, "routes", routes);
  }

  function connectTo(nodeId: string) {
    if (!connectFrom || connectFrom === nodeId) return;
    const id = nextLinkId(topology, connectFrom, nodeId);
    const link: TopologyLink = { id, sourceNodeId: connectFrom, targetNodeId: nodeId, status: "active", config: { latencyMs: 10, packetLossPercent: 0 } };
    mutateTopology((current) => autoAddConnectedBackend({ ...current, links: [...current.links, link] }, link), "Канал связи создан.");
    setConnectFrom("");
  }

  function deleteSelected() {
    if (selectedNodeId) {
      const deletedName = selectedNode?.name ?? selectedNodeId;
      mutateTopology((current) => removeNodeAndReferences(current, selectedNodeId), `${deletedName} удалён.`);
      setSelectedNodeId("");
      setSelectedBackendId("");
    } else if (selectedLinkId) {
      const link = topology.links.find((item) => item.id === selectedLinkId);
      mutateTopology((current) => removeLinkAndMaybeBackend(current, selectedLinkId), link ? `Канал ${link.sourceNodeId} → ${link.targetNodeId} удалён.` : "Канал связи удалён.");
      setSelectedLinkId("");
    }
  }

  function setNodeStatus(id: string, status: TopologyNode["status"]) {
    const node = topology.nodes.find((item) => item.id === id);
    if (!node) return;
    updateNode(id, { status, config: { ...(node.config ?? {}), status } });
  }

  function setLinkStatus(id: string, status: TopologyLink["status"]) {
    const link = topology.links.find((item) => item.id === id);
    if (!link) return;
    updateLink(id, { status, config: { ...(link.config ?? {}), status } });
  }

  function screenToWorld(clientX: number, clientY: number) {
    const rect = canvasRef.current?.getBoundingClientRect();
    if (!rect) return { x: 0, y: 0 };
    return {
      x: (clientX - rect.left - canvasViewport.x) / canvasViewport.zoom,
      y: (clientY - rect.top - canvasViewport.y) / canvasViewport.zoom
    };
  }

  function canvasMouseMove(event: MouseEvent<HTMLDivElement>) {
    if (dragging) {
      const pos = screenToWorld(event.clientX, event.clientY);
      updateNode(dragging.id, { position: { x: pos.x - dragging.dx, y: pos.y - dragging.dy } });
      return;
    }
    if (panning) {
      setCanvasViewport((current) => ({
        ...current,
        x: panning.originX + event.clientX - panning.startX,
        y: panning.originY + event.clientY - panning.startY
      }));
    }
  }

  function startCanvasPan(event: MouseEvent<HTMLElement>) {
    if (event.button !== 0) return;
    const target = event.target as HTMLElement;
    if (target.closest("[data-canvas-node], [data-canvas-control], [data-canvas-link]")) return;
    setPanning({ startX: event.clientX, startY: event.clientY, originX: canvasViewport.x, originY: canvasViewport.y });
  }

  function canvasWheel(event: WheelEvent<HTMLElement>) {
    event.preventDefault();
    const rect = canvasRef.current?.getBoundingClientRect();
    if (!rect) return;
    const nextZoom = clamp(canvasViewport.zoom * (event.deltaY > 0 ? 0.9 : 1.1), 0.35, 2.2);
    const pointerX = event.clientX - rect.left;
    const pointerY = event.clientY - rect.top;
    const worldX = (pointerX - canvasViewport.x) / canvasViewport.zoom;
    const worldY = (pointerY - canvasViewport.y) / canvasViewport.zoom;
    setCanvasViewport({
      zoom: nextZoom,
      x: pointerX - worldX * nextZoom,
      y: pointerY - worldY * nextZoom
    });
  }

  function zoomCanvas(multiplier: number) {
    const rect = canvasRef.current?.getBoundingClientRect();
    if (!rect) {
      setCanvasViewport((current) => ({ ...current, zoom: clamp(current.zoom * multiplier, 0.35, 2.2) }));
      return;
    }
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;
    setCanvasViewport((current) => {
      const nextZoom = clamp(current.zoom * multiplier, 0.35, 2.2);
      const worldX = (centerX - current.x) / current.zoom;
      const worldY = (centerY - current.y) / current.zoom;
      return { zoom: nextZoom, x: centerX - worldX * nextZoom, y: centerY - worldY * nextZoom };
    });
  }

  function resetCanvasView() {
    setCanvasViewport({ x: 0, y: 0, zoom: 1 });
  }

  function addBackendToSelectedLB(nodeId: string) {
    if (!selectedNode || selectedNode.type !== "load_balancer" || !nodeId) return;
    updateLoadBalancerBackends(selectedNode.id, (backends) => [...backends, { nodeId, enabled: true, weight: 1 }]);
    setBackendToAdd("");
    setSelectedBackendId(nodeId);
  }

  function updateLoadBalancerBackends(lbId: string, updater: (backends: LoadBalancerBackend[]) => LoadBalancerBackend[]) {
    const lb = topology.nodes.find((node) => node.id === lbId);
    if (!lb) return;
    updateNode(lbId, { config: { ...(lb.config ?? {}), backends: updater(getBackends(lb)) } });
  }

  function updateBackend(lbId: string, backendNodeId: string, patch: Partial<LoadBalancerBackend>) {
    updateLoadBalancerBackends(lbId, (backends) => backends.map((backend) => (backend.nodeId === backendNodeId ? { ...backend, ...patch } : backend)));
  }

  function removeBackend(lbId: string, backendNodeId: string) {
    updateLoadBalancerBackends(lbId, (backends) => backends.filter((backend) => backend.nodeId !== backendNodeId));
    if (selectedBackendId === backendNodeId) setSelectedBackendId("");
  }

  const saveLabel = saveState === "saved" ? `Сохранено${topologyVersion ? ` v${topologyVersion}` : ""}` : saveState === "saving" ? "Сохранение..." : saveState === "running" ? "Симуляция выполняется..." : "Есть несохранённые изменения";
  const savePillStatus = saveState === "saved" ? "healthy" : saveState === "unsaved" ? "degraded" : "running";
  const runBaseDisabled = busy || saveState === "saving" || Boolean(sourceProblem);

  return (
    <main className="grid h-screen overflow-hidden bg-ink-950 text-white" style={{ gridTemplateRows: layout.timeline ? "auto minmax(0,1fr) 250px" : "auto minmax(0,1fr) 0px" }}>
      <header className="border-b border-white/10 bg-ink-900 px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="font-bold">NetQuest</span>
            <span className="hidden truncate text-sm text-slate-400 md:block">{projectId ? `Проект ${projectId.slice(0, 8)}` : "Демо-проект"}</span>
            <StatusPill status={savePillStatus} label={saveLabel} />
            <StatusPill status={wsStatus === "streaming" ? "running" : "idle"} label={websocketLabel(wsStatus)} />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("left")}>Палитра</Button>
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("right")}>Инспектор</Button>
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("timeline")}>Timeline</Button>
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("packet")}>Пакет</Button>
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("advisor")}>Advisor</Button>
            <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("protocol")}>Протокол</Button>
            {quest && <Button variant="ghost" className="min-h-9 px-2" onClick={() => toggleLayout("quest")}>Упражнение</Button>}
            <Button variant="secondary" className="min-h-9 px-3" onClick={resetDemo} disabled={busy || saveState === "saving"}>
              Загрузить демо-топологию
            </Button>
            <Button variant="secondary" className="min-h-9 px-3" onClick={() => void saveTopology()} disabled={busy || saveState === "saving"}>
              Сохранить
            </Button>
            <Button className="min-h-9 px-3" onClick={() => void runScenario("https_request")} disabled={runBaseDisabled || !httpsUrl.trim()}>
              Запустить HTTPS
            </Button>
          </div>
        </div>

        <div className="mt-3 text-xs font-semibold text-slate-300">Параметры симуляции</div>
        <div className="mt-2 grid gap-3 text-xs lg:grid-cols-[260px_220px_260px_1fr_260px]">
          <label className="block">
            <span className="font-semibold text-slate-200" title="Client, от имени которого будет отправлен DNS Lookup, Ping или HTTPS-запрос.">Источник запроса</span>
            <select className="mt-1 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-2 text-slate-100" value={selectedSourceNodeId} onChange={(event) => setSelectedSourceNodeId(event.target.value)} disabled={clientNodes.length === 0}>
              {clientNodes.length === 0 && <option value="">Нет узлов Client</option>}
              {clientNodes.map((client) => (
                <option key={client.id} value={client.id}>
                  {formatNodeOption(client)}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="font-semibold text-slate-200">Назначение Ping</span>
            <select className="mt-1 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-2 text-slate-100" value={pingTargetNodeId} onChange={(event) => setPingTargetNodeId(event.target.value)} disabled={pingTargetOptions.length === 0}>
              {pingTargetOptions.length === 0 && <option value="">Нет узлов</option>}
              {pingTargetOptions.map((node) => (
                <option key={node.id} value={node.id}>
                  {formatNodeOption(node)}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="font-semibold text-slate-200">Домен для DNS Lookup</span>
            <input className="mt-1 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-2 text-slate-100" value={dnsHostname} onChange={(event) => setDnsHostname(event.target.value)} />
            <span className="mt-1 block text-[11px] text-slate-500">DNS-сервер: автоматически</span>
          </label>
          <label className="block">
            <span className="font-semibold text-slate-200">URL запроса</span>
            <input className="mt-1 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-2 text-slate-100" value={httpsUrl} onChange={(event) => setHttpsUrl(event.target.value)} />
          </label>
          <div className="grid grid-cols-[84px_1fr] gap-2">
            <label className="block">
            <span className="font-semibold text-slate-200" title="Seed делает симуляцию повторяемой. При одинаковой топологии и seed результат будет одинаковым.">Seed</span>
              <input className="mt-1 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-2 text-slate-100" type="number" value={seed} onChange={(event) => setSeed(normalizeSeed(Number(event.target.value)))} />
            </label>
            <div className="pt-5">
              <label className="flex items-center gap-2 text-slate-300">
                <input type="checkbox" checked={fixedSeed} onChange={(event) => setFixedSeed(event.target.checked)} />
                <span>фиксированный seed</span>
              </label>
              <button className="mt-1 text-signal-cyan hover:text-sky-300" onClick={() => setSeed(randomSeed())}>
                Случайный seed
              </button>
            </div>
          </div>
        </div>
        {sourceProblem && <p className="mt-2 text-xs text-signal-amber">{sourceProblem}</p>}
      </header>

      <div className="grid min-h-0" style={{ gridTemplateColumns: `${layout.left ? "260px" : "0px"} minmax(0,1fr) ${layout.right ? "400px" : "0px"}` }}>
        {layout.left && (
        <aside className="min-h-0 overflow-auto border-r border-white/10 bg-ink-900">
          <div className="sticky top-0 z-10 border-b border-white/10 bg-ink-900 px-4 py-3">
            <div className="text-sm font-semibold">Палитра узлов</div>
            <input className="mt-3 h-9 w-full rounded-md border border-white/10 bg-ink-950 px-3 text-sm outline-none focus:border-signal-cyan" placeholder="Найти узел..." value={paletteSearch} onChange={(event) => setPaletteSearch(event.target.value)} />
          </div>
          <div className="grid gap-3 p-3">
            {filteredPaletteGroups.map((group) => (
              <section className="rounded-md border border-white/10 bg-white/[0.03]" key={group.id}>
                <button className="flex w-full items-center justify-between px-3 py-2 text-left text-xs font-semibold text-slate-300" onClick={() => setCollapsedPaletteGroups((current) => ({ ...current, [group.id]: !current[group.id] }))}>
                  <span>{group.title}</span>
                  <span>{collapsedPaletteGroups[group.id] ? "+" : "-"}</span>
                </button>
                {!collapsedPaletteGroups[group.id] && (
                  <div className="grid gap-2 p-2 pt-0">
                    {group.items.map((item) => (
                      <button className="rounded-md border border-white/10 bg-ink-950 px-3 py-2 text-left text-sm hover:border-signal-cyan/60" key={item.type} onClick={() => addNode(item.type)}>
                        <span className="block font-semibold">{nodeTypeLabels[item.type]}</span>
                        <span className="block text-xs text-slate-400">{item.description}</span>
                      </button>
                    ))}
                  </div>
                )}
              </section>
            ))}
          </div>
          {quest && layout.quest && (
            <div className="space-y-3 border-t border-white/10 p-3">
              <h2 className="text-sm font-semibold">Упражнение</h2>
              <p className="text-sm font-bold">{quest.title}</p>
              <p className="text-xs leading-5 text-slate-400">{quest.goal}</p>
              <div className="rounded-md border border-white/10 bg-ink-950 p-3 text-xs">
                <p className="font-semibold text-slate-200">Критерии проверки</p>
                <div className="mt-2 space-y-1 text-slate-400">
                  {quest.expectedChecks.map((check) => (
                    <p key={check.id}>- {check.title}</p>
                  ))}
                </div>
              </div>
              <div className="grid gap-2">
                <Button className="w-full min-h-9 px-2" onClick={checkQuestSolution} disabled={busy}>
                  Проверить решение
                </Button>
                <Button className="w-full min-h-9 px-2" variant="secondary" onClick={revealNextQuestHint} disabled={busy || revealedQuestHintsCount >= (quest.progressiveHints?.length ?? 0)}>
                  {revealedQuestHintsCount < (quest.progressiveHints?.length ?? 0) ? `Показать подсказку ${revealedQuestHintsCount + 1}` : "Все подсказки открыты"}
                </Button>
                <Button className="w-full min-h-9 px-2" variant="secondary" onClick={resetQuest} disabled={busy}>
                  Сбросить упражнение
                </Button>
              </div>
              {(quest.progressiveHints?.length ?? 0) > 0 && (
                <div className="rounded-md border border-white/10 bg-ink-950 p-3 text-xs">
                  <div className="flex items-center justify-between gap-2">
                    <p className="font-semibold text-slate-200">Постепенные подсказки</p>
                    <span className="text-slate-500">{Math.min(revealedQuestHintsCount, quest.progressiveHints.length)}/{quest.progressiveHints.length}</span>
                  </div>
                  <div className="mt-3 space-y-2">
                    {quest.progressiveHints.slice(0, revealedQuestHintsCount).map((hint, index) => (
                      <div className="rounded-md border border-white/10 bg-white/[0.03] px-3 py-2" key={`${hint.title}-${index}`}>
                        <p className="font-semibold text-slate-100">{hint.title}</p>
                        <p className="mt-1 leading-5 text-slate-400">{hint.body}</p>
                        {(hint.actions ?? []).length > 0 && (
                          <div className="mt-2 space-y-1 text-slate-500">
                            {hint.actions?.map((action) => <p key={action}>- {action}</p>)}
                          </div>
                        )}
                      </div>
                    ))}
                    {revealedQuestHintsCount === 0 && <p className="text-slate-500">Сначала попробуйте сами: запустите симуляцию и найдите первое событие с ошибкой.</p>}
                  </div>
                </div>
              )}
              {questResult && (
                <div className="rounded-md border border-white/10 bg-ink-950 p-3 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold">Результат</span>
                    <StatusPill status={questResult.passed ? "healthy" : "degraded"} label={`${questResult.score}%`} />
                  </div>
                  <div className="mt-3 space-y-2">
                    {questResult.checks.map((check) => (
                      <p className={check.passed ? "text-signal-green" : "text-signal-amber"} key={check.id}>{check.passed ? "✓" : "!"} {check.message}</p>
                    ))}
                  </div>
                  {!questResult.passed && questResult.hints.length > 0 && (
                    <div className="mt-3 rounded-md border border-signal-amber/30 bg-signal-amber/10 p-2 text-signal-amber">
                      <p className="font-semibold">Подсказка по ошибочной проверке</p>
                      <p className="mt-1 leading-5">{questResult.hints[0]}</p>
                    </div>
                  )}
                  {questResult.passed && (questResult.afterSolutionExplanation || quest.afterSolutionExplanation) && (
                    <div className="mt-3 rounded-md border border-signal-green/30 bg-signal-green/10 p-2 text-slate-200">
                      <p className="font-semibold text-signal-green">После решения</p>
                      <p className="mt-1 leading-5">{questResult.afterSolutionExplanation || quest.afterSolutionExplanation}</p>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
          <div className="space-y-2 border-t border-white/10 p-3">
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => void runScenario("dns_lookup")} disabled={runBaseDisabled || !dnsHostname.trim()}>
              Запустить DNS
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => void runScenario("icmp_ping")} disabled={runBaseDisabled || !pingTargetNodeId}>
              Запустить Ping
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => void runScenario("failover_demo")} disabled={runBaseDisabled || !httpsUrl.trim()}>
              Запустить Failover
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => selectedNode && setNodeStatus(selectedNode.id, "down")} disabled={!selectedNode || selectedNode.status === "down"}>
              Отключить выбранный узел
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => selectedNode && setNodeStatus(selectedNode.id, "healthy")} disabled={!selectedNode || selectedNode.status !== "down"}>
              Восстановить выбранный узел
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => selectedLink && setLinkStatus(selectedLink.id, "down")} disabled={!selectedLink || selectedLink.status === "down"}>
              Отключить выбранный link
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={() => selectedLink && setLinkStatus(selectedLink.id, "active")} disabled={!selectedLink || selectedLink.status !== "down"}>
              Восстановить выбранный link
            </Button>
            <Button className="w-full min-h-9 px-2" variant="secondary" onClick={analyzeCurrentTopology} disabled={busy}>
              Проверить топологию
            </Button>
          </div>
        </aside>
        )}

        <section
          className={`network-grid relative min-h-0 overflow-hidden bg-ink-950 ${panning ? "cursor-grabbing" : "cursor-grab"}`}
          data-canvas-root
          onMouseDown={startCanvasPan}
          onMouseMove={canvasMouseMove}
          onMouseLeave={() => {
            setDragging(null);
            setPanning(null);
          }}
          onMouseUp={() => {
            setDragging(null);
            setPanning(null);
          }}
          onWheel={canvasWheel}
          ref={canvasRef}
        >
          <div className="absolute right-4 top-4 z-20 flex overflow-hidden rounded-md border border-white/10 bg-ink-900/90 shadow-2xl shadow-black/30 backdrop-blur" data-canvas-control>
            <button className="h-9 w-9 border-r border-white/10 text-sm font-bold text-slate-200 hover:bg-white/[0.08]" onClick={() => zoomCanvas(0.85)} title="Отдалить карту">
              -
            </button>
            <button className="h-9 w-9 border-r border-white/10 text-sm font-bold text-slate-200 hover:bg-white/[0.08]" onClick={() => zoomCanvas(1.18)} title="Приблизить карту">
              +
            </button>
            <button className="h-9 px-3 border-r border-white/10 text-xs font-semibold text-slate-200 hover:bg-white/[0.08]" onClick={() => fitCanvasToDocument(topology)} title="Показать все узлы">
              По размеру
            </button>
            <button className="h-9 px-3 text-xs font-semibold text-slate-200 hover:bg-white/[0.08]" onClick={resetCanvasView} title="Сбросить масштаб и сдвиг">
              Сброс
            </button>
          </div>
          <svg className="absolute inset-0 h-full w-full">
            <g transform={`translate(${canvasViewport.x} ${canvasViewport.y}) scale(${canvasViewport.zoom})`}>
              {topology.links.map((link) => {
                const source = topology.nodes.find((node) => node.id === link.sourceNodeId);
                const target = topology.nodes.find((node) => node.id === link.targetNodeId);
                if (!source?.position || !target?.position) return null;
                const active = linkInActivePath(link, activePath);
                return (
                  <line
                    data-canvas-link
                    key={link.id}
                    x1={source.position.x + 56}
                    y1={source.position.y + 28}
                    x2={target.position.x + 56}
                    y2={target.position.y + 28}
                    stroke={link.status === "down" ? "#ff6b76" : active ? "#35d3ff" : "rgba(255,255,255,0.28)"}
                    strokeWidth={(active ? 4 : 2) / canvasViewport.zoom}
                    onClick={(event) => {
                      event.stopPropagation();
                      setSelectedLinkId(link.id);
                      setSelectedNodeId("");
                    }}
                    onMouseDown={(event) => event.stopPropagation()}
                  />
                );
              })}
            </g>
          </svg>
          <div className="absolute inset-0" style={{ transform: `translate(${canvasViewport.x}px, ${canvasViewport.y}px) scale(${canvasViewport.zoom})`, transformOrigin: "0 0" }}>
            {topology.nodes.map((node) => {
              const selectedByLB = selectedBackendNodeId === node.id;
              const skipped = skippedBackendIds.has(node.id);
              return (
                <button
                  className={`absolute h-14 w-28 cursor-grab rounded-md border bg-ink-900 text-xs font-bold shadow-lg active:cursor-grabbing ${
                    selectedNodeId === node.id || selectedByLB
                      ? "border-signal-cyan text-signal-cyan"
                      : node.status === "down" || skipped
                        ? "border-signal-red text-signal-red"
                        : "border-white/20 text-white"
                  } ${activePath.includes(node.id) || selectedByLB ? "shadow-glow" : ""}`}
                  data-canvas-node
                  key={node.id}
                  style={{ left: node.position?.x ?? 0, top: node.position?.y ?? 0 }}
                  onClick={(event) => {
                    event.stopPropagation();
                    connectTo(node.id);
                    setSelectedNodeId(node.id);
                    setSelectedLinkId("");
                  }}
                  onMouseDown={(event) => {
                    event.stopPropagation();
                    const pos = node.position ?? { x: 0, y: 0 };
                    const world = screenToWorld(event.clientX, event.clientY);
                    setDragging({ id: node.id, dx: world.x - pos.x, dy: world.y - pos.y });
                  }}
                >
                  <span className="block truncate px-2">{node.name ?? node.id}</span>
                  <span className="block text-[10px] font-medium opacity-70">{node.type}</span>
                </button>
              );
            })}
          </div>
          {message && <p className="absolute bottom-4 left-4 max-w-xl rounded-md border border-white/10 bg-black/55 px-4 py-3 text-sm text-slate-200">{message}</p>}
        </section>

        {layout.right && (
        <aside className="min-h-0 overflow-auto border-l border-white/10 bg-ink-900 p-4">
          <h2 className="text-sm font-semibold">Инспектор</h2>
          {selectedNode && (
            <div className="mt-4 space-y-3 text-sm">
              <label className="block">
                <span className="text-xs text-slate-400">Имя узла</span>
                <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={selectedNode.name ?? ""} onChange={(event) => updateNode(selectedNode.id, { name: event.target.value })} />
              </label>
              <label className="block">
                <span className="text-xs text-slate-400">Status</span>
                <select className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={selectedNode.status ?? "healthy"} onChange={(event) => updateNode(selectedNode.id, { status: event.target.value as TopologyNode["status"] })}>
                  <option value="healthy">healthy</option>
                  <option value="degraded">degraded</option>
                  <option value="down">down</option>
                </select>
              </label>
              <label className="block">
                <span className="text-xs text-slate-400">IP-адрес</span>
                <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={String(selectedNode.config?.ip ?? "")} onChange={(event) => updateNodeConfig(selectedNode.id, "ip", event.target.value)} />
              </label>
              {(selectedNode.type === "client" || selectedNode.type === "server" || selectedNode.type === "router") && (
                <label className="block">
                  <span className="text-xs text-slate-400">Подсеть/CIDR</span>
                  <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" placeholder="10.0.1.10/24" value={String(selectedNode.config?.cidr ?? "")} onChange={(event) => updateNodeConfig(selectedNode.id, "cidr", event.target.value)} />
                </label>
              )}
              {selectedNode.type === "client" && (
                <label className="block">
                  <span className="text-xs text-slate-400">Шлюз по умолчанию</span>
                  <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" placeholder="10.0.1.1" value={String(selectedNode.config?.defaultGateway ?? "")} onChange={(event) => updateNodeConfig(selectedNode.id, "defaultGateway", event.target.value)} />
                </label>
              )}
              {selectedNode.type === "router" && (
                <div className="space-y-3 rounded-md border border-white/10 bg-white/[0.04] p-3">
                  <div className="flex items-center justify-between">
                    <h3 className="font-semibold">Routing table</h3>
                    <button className="text-xs text-signal-cyan" onClick={() => addRoute(selectedNode.id)}>add route</button>
                  </div>
                  {(Array.isArray(selectedNode.config?.routes) ? (selectedNode.config?.routes as Record<string, unknown>[]) : []).map((route, index) => (
                    <div className="grid gap-2 rounded-md border border-white/10 bg-ink-950 p-2" key={index}>
                      <input className="h-8 rounded-md border border-white/10 bg-ink-900 px-2 text-xs" placeholder="destination CIDR" value={String(route.destination ?? "")} onChange={(event) => updateRoute(selectedNode.id, index, "destination", event.target.value)} />
                      <input className="h-8 rounded-md border border-white/10 bg-ink-900 px-2 text-xs" placeholder="gateway" value={String(route.gateway ?? "")} onChange={(event) => updateRoute(selectedNode.id, index, "gateway", event.target.value)} />
                      <div className="grid grid-cols-[1fr_80px_60px] gap-2">
                        <input className="h-8 rounded-md border border-white/10 bg-ink-900 px-2 text-xs" placeholder="interface" value={String(route.interface ?? "")} onChange={(event) => updateRoute(selectedNode.id, index, "interface", event.target.value)} />
                        <input className="h-8 rounded-md border border-white/10 bg-ink-900 px-2 text-xs" type="number" value={String(route.metric ?? 100)} onChange={(event) => updateRoute(selectedNode.id, index, "metric", Number(event.target.value) || 100)} />
                        <button className="text-xs text-signal-red" onClick={() => removeRoute(selectedNode.id, index)}>del</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {selectedNode.type === "dns" && (
                <>
                  <label className="block">
                    <span className="text-xs text-slate-400">DNS-запись</span>
                    <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={String(((selectedNode.config?.records as any[])?.[0]?.name) ?? DEFAULT_DNS_HOSTNAME)} onChange={(event) => updateNodeConfig(selectedNode.id, "records", [{ name: event.target.value, type: "A", value: "10.0.2.10", ttl: 300 }])} />
                  </label>
                  <label className="block">
                    <span className="text-xs text-slate-400">A-значение</span>
                    <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={String(((selectedNode.config?.records as any[])?.[0]?.value) ?? "10.0.2.10")} onChange={(event) => updateNodeConfig(selectedNode.id, "records", [{ name: DEFAULT_DNS_HOSTNAME, type: "A", value: event.target.value, ttl: 300 }])} />
                  </label>
                </>
              )}
              {selectedNode.type === "firewall" && (
                <label className="block">
                  <span className="text-xs text-slate-400">Решение Firewall</span>
                  <select className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={String(((selectedNode.config?.rules as any[])?.[0]?.action) ?? "allow")} onChange={(event) => updateNodeConfig(selectedNode.id, "rules", [{ priority: 100, action: event.target.value, protocol: "tcp", source: "10.0.1.0/24", destination: "10.0.2.10/32", port: 443 }])}>
                    <option value="allow">allow tcp/443</option>
                    <option value="deny">deny tcp/443</option>
                  </select>
                </label>
              )}
              {selectedNode.type === "load_balancer" && (
                <div className="space-y-3 rounded-md border border-white/10 bg-white/[0.04] p-3">
                  <h3 className="font-semibold">Пул серверов Load Balancer</h3>
                  <label className="block">
                    <span className="text-xs text-slate-400">Алгоритм</span>
                    <select className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={String(selectedNode.config?.algorithm ?? "round_robin")} onChange={(event) => updateNodeConfig(selectedNode.id, "algorithm", event.target.value)}>
                      <option value="round_robin">round_robin</option>
                      <option value="least_connections">least_connections</option>
                    </select>
                  </label>
                  <label className="flex items-center gap-2 text-slate-300">
                    <input type="checkbox" checked={selectedNode.config?.autoDiscoverConnectedServers !== false} onChange={(event) => updateNodeConfig(selectedNode.id, "autoDiscoverConnectedServers", event.target.checked)} />
                    <span>Автоматически добавлять подключённые Server</span>
                  </label>
                  <div className="space-y-2">
                    {selectedLBBackends.length === 0 && <p className="rounded-md border border-signal-amber/30 bg-signal-amber/10 px-3 py-2 text-xs text-signal-amber">Пул серверов пуст.</p>}
                    {selectedLBBackends.map((backend) => {
                      const server = topology.nodes.find((node) => node.id === backend.nodeId);
                      return (
                        <div className="rounded-md border border-white/10 bg-ink-950 p-3" key={backend.nodeId}>
                          <div className="flex items-start justify-between gap-2">
                            <button className="text-left" onClick={() => setSelectedBackendId(backend.nodeId)}>
                              <span className="block font-semibold">{server?.name ?? backend.nodeId}</span>
                              <span className="block text-xs text-slate-400">{backend.nodeId} · {server?.status ?? "узел отсутствует"}</span>
                            </button>
                            <button className="text-xs text-signal-red" onClick={() => removeBackend(selectedNode.id, backend.nodeId)}>удалить</button>
                          </div>
                          <div className="mt-3 grid grid-cols-[1fr_84px] gap-2">
                            <label className="flex items-center gap-2 text-xs text-slate-300">
                              <input type="checkbox" checked={backend.enabled !== false} onChange={(event) => updateBackend(selectedNode.id, backend.nodeId, { enabled: event.target.checked })} />
                              включён
                            </label>
                            <input className="h-8 rounded-md border border-white/10 bg-ink-900 px-2 text-xs" type="number" min={1} value={backend.weight ?? 1} onChange={(event) => updateBackend(selectedNode.id, backend.nodeId, { weight: Math.max(1, Number(event.target.value) || 1) })} />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  <div className="flex gap-2">
                    <select className="h-10 min-w-0 flex-1 rounded-md border border-white/10 bg-ink-950 px-3" value={backendToAdd} onChange={(event) => setBackendToAdd(event.target.value)}>
                      <option value="">Добавить сервер</option>
                      {addableBackends.map((server) => (
                        <option key={server.id} value={server.id}>
                          {server.name ?? server.id} · {server.status ?? "healthy"}{isConnected(topology, selectedNode.id, server.id) ? " · подключён" : ""}
                        </option>
                      ))}
                    </select>
                    <Button className="min-h-10 px-3" variant="secondary" disabled={!backendToAdd} onClick={() => addBackendToSelectedLB(backendToAdd)}>Добавить</Button>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <Button className="min-h-9 px-2" variant="secondary" disabled={!selectedBackendId} onClick={() => selectedBackendId && setNodeStatus(selectedBackendId, "down")}>Отключить backend</Button>
                    <Button className="min-h-9 px-2" variant="secondary" disabled={!selectedBackendId} onClick={() => selectedBackendId && setNodeStatus(selectedBackendId, "healthy")}>Восстановить backend</Button>
                  </div>
                </div>
              )}
              <Button className="w-full min-h-9" variant="secondary" onClick={() => setConnectFrom(selectedNode.id)}>
                {connectFrom === selectedNode.id ? "Выберите целевой узел" : "Начать связь"}
              </Button>
              <Button className="w-full min-h-9" variant="secondary" onClick={deleteSelected}>
                Удалить
              </Button>
            </div>
          )}
          {selectedLink && (
            <div className="mt-4 space-y-3 text-sm">
              <p className="text-slate-300">{selectedLink.sourceNodeId} → {selectedLink.targetNodeId}</p>
              <label className="block">
                <span className="text-xs text-slate-400">Состояние связи</span>
                <select className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" value={selectedLink.status ?? "active"} onChange={(event) => updateLink(selectedLink.id, { status: event.target.value as TopologyLink["status"] })}>
                  <option value="active">active</option>
                  <option value="degraded">degraded</option>
                  <option value="down">down</option>
                </select>
              </label>
              <label className="block">
                <span className="text-xs text-slate-400" title="Задержка канала влияет на виртуальное время доставки пакета.">Задержка, мс</span>
                <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" type="number" min={0} value={String(selectedLink.config?.latencyMs ?? 10)} onChange={(event) => updateLink(selectedLink.id, { config: { ...(selectedLink.config ?? {}), latencyMs: Math.max(0, Number(event.target.value) || 0) } })} />
              </label>
              <label className="block">
                <span className="text-xs text-slate-400">Потеря пакетов, %</span>
                <input className="mt-1 h-10 w-full rounded-md border border-white/10 bg-ink-950 px-3" type="number" min={0} max={100} value={String(selectedLink.config?.packetLossPercent ?? 0)} onChange={(event) => updateLink(selectedLink.id, { config: { ...(selectedLink.config ?? {}), packetLossPercent: clamp(Number(event.target.value) || 0, 0, 100) } })} />
              </label>
              <Button className="w-full min-h-9" variant="secondary" onClick={deleteSelected}>
                Удалить link
              </Button>
            </div>
          )}
          {summary && layout.packet && (
            <div className="mt-6 rounded-md border border-white/10 bg-white/[0.04] p-3 text-sm">
              <h3 className="font-semibold">Инспектор пакета</h3>
              <p className="mt-2 text-xs text-slate-400">NetQuest показывает виртуальный путь пакета. Реальные сетевые пакеты не отправляются.</p>
              <div className="mt-3 space-y-1 text-slate-300">
                <p>Статус: {summary.status}</p>
                <p>Источник: {formatSummarySource(sourceNodeForSummary, summary)}</p>
                <p>Пакет: {summary.packetId}</p>
                <p>Resolved IP: {summary.resolvedIp || "n/a"}</p>
                <p>Backend-сервер: {summary.selectedBackendName || summary.selectedBackendNodeId || summary.selectedBackend || "n/a"}</p>
                <p>Итоговая задержка: {summary.totalLatencyMs}ms</p>
                <p>Seed: {summary.seed ?? seed}</p>
              </div>
              <p className="mt-2 text-xs text-slate-400">Путь: {summary.path.length ? summary.path.join(" → ") : "n/a"}</p>
              {summary.healthyBackends?.length ? <p className="mt-2 text-xs text-signal-green">Доступные серверы: {summary.healthyBackends.join(", ")}</p> : null}
              {summary.skippedBackends?.length ? (
                <div className="mt-2 space-y-1 text-xs text-signal-amber">
                  {summary.skippedBackends.map((backend) => <p key={`${backend.nodeId}-${backend.reason}`}>Пропущен {backend.name || backend.nodeId}: {translateSkipReason(backend.reason)}</p>)}
                </div>
              ) : null}
              {summary.failover && <p className="mt-2 text-xs text-signal-amber">Failover пересчитал route.</p>}
              <div className="mt-4 rounded-md border border-white/10 bg-ink-950 p-3">
                <h4 className="font-semibold">Расчёт времени</h4>
                <p className="mt-2 text-xs text-slate-400">Это виртуальное время симуляции. Оно рассчитывается по задержкам каналов, выбранному пути, потере пакетов, задержкам обработки и seed.</p>
                {summary.latencyFormula && <p className="mt-2 text-xs text-slate-300">{summary.latencyFormula}</p>}
                <div className="mt-3 space-y-2">
                  {(summary.latencyBreakdown ?? []).map((stage) => (
                    <div className="rounded-md border border-white/10 bg-white/[0.03] px-3 py-2" key={`${stage.stage}-${stage.label}`}>
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-semibold">{stage.label}</span>
                        <span className="text-signal-cyan">{stage.durationMs}ms</span>
                      </div>
                      <p className="mt-1 text-xs text-slate-400">{formatLatencyDetails(stage.details)}</p>
                    </div>
                  ))}
                  {!summary.latencyBreakdown?.length && <p className="text-xs text-slate-400">Backend не вернул детализацию времени для этой симуляции.</p>}
                </div>
              </div>
              <div className="mt-3 space-y-1 text-xs text-slate-400">
                {summary.decisions.map((decision) => <p key={decision}>{translateDecision(decision)}</p>)}
                {summary.errors.map((error) => <p className="text-signal-red" key={error}>{translateSimulationError(error)}</p>)}
              </div>
            </div>
          )}
          {summary && layout.protocol && (
            <div className="mt-6 rounded-md border border-white/10 bg-white/[0.04] p-3 text-sm">
              <div className="flex items-center justify-between gap-3">
                <h3 className="font-semibold">Протокольный разбор</h3>
                <button className="text-xs text-slate-400 hover:text-white" onClick={() => toggleLayout("protocol")}>Скрыть</button>
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                {(["summary", "dns", "routing", "firewall", "tcp", "tls", "loadBalancer", "errors"] as ProtocolTab[]).map((tab) => (
                  <button className={`rounded-md border px-2 py-1 text-xs ${activeProtocolTab === tab ? "border-signal-cyan text-signal-cyan" : "border-white/10 text-slate-300"}`} key={tab} onClick={() => setActiveProtocolTab(tab)}>
                    {protocolTabLabel(tab)}
                  </button>
                ))}
              </div>
              <div className="mt-3 max-h-56 overflow-auto rounded-md border border-white/10 bg-ink-950 p-3 text-xs text-slate-300">
                {formatProtocolDetails(protocolDetails, activeProtocolTab).map((line) => <p className="mb-1" key={line}>{line}</p>)}
              </div>
            </div>
          )}
          {layout.advisor && (
            <div className="mt-6 rounded-md border border-white/10 bg-white/[0.04] p-3 text-sm">
              <div className="flex items-center justify-between gap-3">
                <h3 className="font-semibold">Validation Advisor</h3>
                <div className="flex gap-2">
                  <button className="text-xs text-signal-cyan hover:text-sky-300" onClick={analyzeCurrentTopology}>Проверить</button>
                  <button className="text-xs text-slate-400 hover:text-white" onClick={() => setAdvisorIssues([])}>Очистить</button>
                </div>
              </div>
              <div className="mt-3 space-y-2">
                {advisorIssues.length === 0 && <p className="text-xs text-slate-400">Нажмите «Проверить», чтобы найти проблемы в топологии.</p>}
                {advisorIssues.map((issue) => (
                  <button className={`w-full rounded-md border px-3 py-2 text-left text-xs ${issue.severity === "error" ? "border-signal-red/35 bg-signal-red/10" : issue.severity === "warning" ? "border-signal-amber/35 bg-signal-amber/10" : "border-white/10 bg-ink-950"}`} key={`${issue.code}-${issue.affectedNodeId ?? issue.affectedLinkId ?? issue.title}`} onClick={() => focusAdvisorIssue(issue)}>
                    <span className="block font-semibold">{issue.title}</span>
                    <span className="mt-1 block text-slate-300">{issue.message}</span>
                    <span className="mt-1 block text-slate-500">{issue.suggestedFix}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </aside>
        )}
      </div>

      {layout.timeline && <footer className="border-t border-white/10 bg-ink-900">
        <div className="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3 text-sm font-semibold">
          <div className="flex items-center gap-3">
            <span>Timeline событий</span>
            <button className="rounded-md border border-white/10 px-2 py-1 text-xs text-signal-cyan hover:bg-white/[0.06]" onClick={() => setShowLatencyHelp(true)} title="Timestamp показывает виртуальное время симуляции, а не реальное время выполнения backend.">
              Как считается время?
            </button>
          </div>
          <span className="text-xs text-slate-400">{events.length} событий</span>
        </div>
        <div className="grid max-h-[190px] gap-2 overflow-auto p-3 font-mono text-xs text-slate-300 md:grid-cols-2">
          {events.map((event, index) => {
            const previous = index > 0 ? events[index - 1].timestampMs : 0;
            const delta = index === 0 ? event.timestampMs : event.timestampMs - previous;
            return (
              <div className={`rounded-md border px-3 py-2 ${event.severity === "error" ? "border-signal-red/35 bg-signal-red/10" : event.severity === "warn" ? "border-signal-amber/35 bg-signal-amber/10" : "border-white/10 bg-white/[0.04]"}`} key={event.id}>
                [{event.timestampMs}ms] {eventTypeLabels[event.type] ?? event.type} {event.sourceNodeId || event.targetNodeId ? `${event.sourceNodeId || "system"} → ${event.targetNodeId || "system"} ` : ""}
                {translateEventMessage(event.message)} {index > 0 && <span className="text-slate-500">(+{delta}ms)</span>}
              </div>
            );
          })}
        </div>
      </footer>}

      {showLatencyHelp && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4" onClick={() => setShowLatencyHelp(false)}>
          <div className="max-w-lg rounded-md border border-white/10 bg-ink-900 p-5 shadow-2xl" onClick={(event) => event.stopPropagation()}>
            <div className="flex items-start justify-between gap-4">
              <h2 className="text-lg font-bold">Как считается время?</h2>
              <button className="text-sm text-slate-400 hover:text-white" onClick={() => setShowLatencyHelp(false)}>Закрыть</button>
            </div>
            <div className="mt-4 space-y-3 text-sm leading-6 text-slate-300">
              <p>Время в Timeline — это виртуальное время симуляции. Оно не измеряется в реальной сети.</p>
              <p>Backend рассчитывает его на основе выбранного пути, задержек каналов, потери пакетов, задержек обработки и seed. Для Ping используется приближённый RTT: путь туда и обратно.</p>
              <p>Для HTTPS добавляются DNS Lookup, TCP handshake, TLS handshake, решение Firewall, выбор сервера Load Balancer и доставка до него.</p>
              <p>Если топология и seed одинаковые, результат будет повторяемым. Чтобы получить другой вариант потери пакетов или задержек обработки, измените seed.</p>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}

function getBackends(node: TopologyNode): LoadBalancerBackend[] {
  const backends = node.config?.backends;
  return Array.isArray(backends) ? (backends as LoadBalancerBackend[]).filter((backend) => backend && typeof backend.nodeId === "string") : [];
}

function defaultNodeConfig(type: TopologyNode["type"], index: number, id: string): Record<string, unknown> {
  if (type === "load_balancer") {
    return { ip: `10.0.${index}.10`, algorithm: "round_robin", autoDiscoverConnectedServers: true, healthCheckEnabled: true, backends: [] };
  }
  if (type === "server") {
    return { ip: `10.0.${index}.10`, serviceName: id, port: 443 };
  }
  if (type === "dns") {
    return { ip: `10.0.${index}.53`, records: [{ name: DEFAULT_DNS_HOSTNAME, type: "A", value: "10.0.2.10", ttl: 300 }] };
  }
  if (type === "firewall") {
    return { ip: `10.0.${index}.254`, defaultPolicy: "deny", rules: [{ priority: 100, action: "allow", protocol: "tcp", source: "10.0.1.0/24", destination: "10.0.2.10/32", port: 443 }] };
  }
  return { ip: `10.0.${index}.10` };
}

function nextNodeId(topology: TopologyDocument, type: TopologyNode["type"]) {
  let index = 1;
  while (topology.nodes.some((node) => node.id === `${type}-${index}`)) {
    index += 1;
  }
  return `${type}-${index}`;
}

function nextLinkId(topology: TopologyDocument, source: string, target: string) {
  const base = `link-${source}-${target}`;
  if (!topology.links.some((link) => link.id === base)) return base;
  let index = 2;
  while (topology.links.some((link) => link.id === `${base}-${index}`)) {
    index += 1;
  }
  return `${base}-${index}`;
}

function autoAddConnectedBackend(topology: TopologyDocument, link: TopologyLink): TopologyDocument {
  const source = topology.nodes.find((node) => node.id === link.sourceNodeId);
  const target = topology.nodes.find((node) => node.id === link.targetNodeId);
  const lb = source?.type === "load_balancer" ? source : target?.type === "load_balancer" ? target : null;
  const server = source?.type === "server" ? source : target?.type === "server" ? target : null;
  if (!lb || !server || lb.config?.autoDiscoverConnectedServers === false) return topology;
  return {
    ...topology,
    nodes: topology.nodes.map((node) => {
      if (node.id !== lb.id) return node;
      const backends = getBackends(node);
      if (backends.some((backend) => backend.nodeId === server.id)) return node;
      return { ...node, config: { ...(node.config ?? {}), backends: [...backends, { nodeId: server.id, enabled: true, weight: 1 }] } };
    })
  };
}

function removeNodeAndReferences(topology: TopologyDocument, nodeId: string): TopologyDocument {
  return {
    nodes: topology.nodes
      .filter((node) => node.id !== nodeId)
      .map((node) => node.type === "load_balancer" ? { ...node, config: { ...(node.config ?? {}), backends: getBackends(node).filter((backend) => backend.nodeId !== nodeId) } } : node),
    links: topology.links.filter((link) => link.sourceNodeId !== nodeId && link.targetNodeId !== nodeId)
  };
}

function removeLinkAndMaybeBackend(topology: TopologyDocument, linkId: string): TopologyDocument {
  const link = topology.links.find((item) => item.id === linkId);
  if (!link) return topology;
  const withoutLink = { ...topology, links: topology.links.filter((item) => item.id !== linkId) };
  const source = topology.nodes.find((node) => node.id === link.sourceNodeId);
  const target = topology.nodes.find((node) => node.id === link.targetNodeId);
  const lb = source?.type === "load_balancer" ? source : target?.type === "load_balancer" ? target : null;
  const server = source?.type === "server" ? source : target?.type === "server" ? target : null;
  if (!lb || !server || lb.config?.autoDiscoverConnectedServers === false) return withoutLink;
  return {
    ...withoutLink,
    nodes: withoutLink.nodes.map((node) => node.id === lb.id ? { ...node, config: { ...(node.config ?? {}), backends: getBackends(node).filter((backend) => backend.nodeId !== server.id) } } : node)
  };
}

function isConnected(topology: TopologyDocument, left: string, right: string) {
  return topology.links.some((link) => (link.sourceNodeId === left && link.targetNodeId === right) || (link.sourceNodeId === right && link.targetNodeId === left));
}

function linkInActivePath(link: TopologyLink, path: string[]) {
  for (let i = 0; i < path.length - 1; i += 1) {
    if ((link.sourceNodeId === path[i] && link.targetNodeId === path[i + 1]) || (link.targetNodeId === path[i] && link.sourceNodeId === path[i + 1])) {
      return true;
    }
  }
  return false;
}

function getNodeStatus(node?: TopologyNode) {
  return node?.status ?? "healthy";
}

function nodeIp(node?: TopologyNode) {
  return String(node?.config?.ip ?? "");
}

function formatNodeOption(node: TopologyNode) {
  const ip = nodeIp(node);
  return `${node.name ?? node.id} · ${ip || node.id} · ${getNodeStatus(node)}`;
}

function scenarioTarget(type: ScenarioType, dnsHostname: string, pingTargetNodeId: string, httpsUrl: string) {
  if (type === "dns_lookup") return dnsHostname.trim();
  if (type === "icmp_ping") return pingTargetNodeId;
  return httpsUrl.trim();
}

function scenarioValidationMessage(type: ScenarioType, source: TopologyNode | undefined, pingTarget: TopologyNode | undefined, dnsHostname: string, httpsUrl: string) {
  if (!source) return "Выберите Client, от которого нужно отправить запрос.";
  if (getNodeStatus(source) === "down") return "Выбранный Client недоступен. Восстановите его или выберите другой источник.";
  if (type === "dns_lookup" && !dnsHostname.trim()) return "Введите домен для DNS Lookup.";
  if (type === "icmp_ping" && !pingTarget) return "Выберите назначение для Ping.";
  if ((type === "https_request" || type === "failover_demo") && !httpsUrl.trim()) return "Введите URL для HTTPS-запроса.";
  return "";
}

function normalizeSeed(value: number) {
  if (!Number.isFinite(value) || value <= 0) return 1;
  return Math.floor(value);
}

function randomSeed() {
  return Math.floor(1 + Math.random() * 999_999);
}

function cacheQuestContext(attemptId: string, quest: Quest, attempt: QuestAttempt) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(`netquest.quest.${attemptId}`, JSON.stringify({ quest, attempt }));
}

function storedQuestHintCount(attemptId: string) {
  if (typeof window === "undefined") return 0;
  const raw = window.localStorage.getItem(`netquest.quest.${attemptId}.revealedHints`);
  const value = raw ? Number(raw) : 0;
  return Number.isFinite(value) && value > 0 ? Math.floor(value) : 0;
}

function storeQuestHintCount(attemptId: string, count: number) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(`netquest.quest.${attemptId}.revealedHints`, String(Math.max(0, Math.floor(count))));
}

function clearStoredQuestHintCount(attemptId: string) {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(`netquest.quest.${attemptId}.revealedHints`);
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value));
}

function websocketLabel(value: string) {
  if (value === "streaming") return "WebSocket stream";
  if (value === "connecting") return "WebSocket подключается";
  if (value === "polling fallback") return "polling fallback";
  return "stream idle";
}

function formatSummarySource(node: TopologyNode | undefined, summary: SimulationSummary) {
  if (!summary.sourceNodeId) return "n/a";
  if (!node) return summary.sourceNodeId;
  const ip = nodeIp(node);
  return `${node.name ?? node.id} · ${ip || node.id}`;
}

function formatLatencyDetails(details?: Record<string, unknown>) {
  if (!details) return "";
  const parts: string[] = [];
  const path = Array.isArray(details.path) ? details.path.join(" → ") : "";
  if (path) parts.push(`путь: ${path}`);
  for (const [key, label] of [
    ["oneWayLatencyMs", "путь в одну сторону"],
    ["rttMs", "RTT"],
    ["processingMs", "обработка"],
    ["retryDelayMs", "retry"],
    ["attempts", "попытки"],
    ["selectedBackendName", "сервер"]
  ]) {
    const value = details[key];
    if (value !== undefined && value !== null && value !== "") {
      parts.push(`${label}: ${String(value)}${key.endsWith("Ms") ? "ms" : ""}`);
    }
  }
  return parts.join(" · ") || "задержка обработки";
}

function protocolTabLabel(tab: ProtocolTab) {
  const labels: Record<ProtocolTab, string> = {
    summary: "Итог",
    dns: "DNS",
    routing: "Маршрут",
    firewall: "Firewall",
    tcp: "TCP",
    tls: "TLS",
    loadBalancer: "Load Balancer",
    errors: "Ошибки"
  };
  return labels[tab];
}

function formatProtocolDetails(details: Record<string, unknown>, tab: ProtocolTab) {
  const value = details[tab];
  if (value === undefined || value === null) return ["Нет данных для этого слоя протокола."];
  return flattenProtocolValue(value);
}

function flattenProtocolValue(value: unknown, prefix = ""): string[] {
  if (value === null || value === undefined) return [`${prefix || "значение"}: n/a`];
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return [`${prefix || "значение"}: ${String(value)}`];
  }
  if (Array.isArray(value)) {
    if (value.length === 0) return [`${prefix || "элементы"}: нет`];
    return value.flatMap((item, index) => flattenProtocolValue(item, `${prefix || "элемент"}[${index + 1}]`));
  }
  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    if (entries.length === 0) return [`${prefix || "object"}: none`];
    return entries.flatMap(([key, child]) => flattenProtocolValue(child, prefix ? `${prefix}.${key}` : key));
  }
  return [`${prefix || "value"}: ${String(value)}`];
}

function translateEventMessage(message: string) {
  const messages: Record<string, string> = {
    "simulation started": "симуляция запущена",
    "topology validated": "топология проверена",
    "packet created": "пакет создан",
    "DNS query": "DNS запрос",
    "DNS response": "DNS ответ",
    "route selected": "маршрут выбран",
    "packet dropped by deterministic packet loss": "пакет потерян по детерминированной потере пакетов",
    "ICMP echo reply delivered": "ICMP echo reply доставлен",
    "HTTPS request delivered": "HTTPS-запрос доставлен",
    "simulation completed": "симуляция завершена",
    "simulation failed": "симуляция завершилась ошибкой",
    "load balancer selected backend": "Load Balancer выбрал сервер",
    "load balancer skipped unavailable backend(s)": "Load Balancer пропустил недоступные серверы",
    "load balancer has no healthy backends available": "У Load Balancer нет доступных серверов."
  };
  return messages[message] ?? message;
}

function translateSkipReason(reason: string) {
  const reasons: Record<string, string> = {
    "node is down": "узел выключен",
    "backend is disabled": "сервер отключён в пуле",
    "backend node does not exist": "сервер из пула не существует",
    "backend node is not a server": "узел из пула не является Server",
    "no active path from load balancer": "нет активного пути от Load Balancer",
    "backend nodeId is empty": "в пуле есть пустой nodeId"
  };
  return reasons[reason] ?? reason;
}

function translateDecision(decision: string) {
  return decision
    .replace("DNS resolved", "DNS разрешил")
    .replace("Graph route selected", "Выбран маршрут по графу")
    .replace("Load balancer selected", "Load Balancer выбрал")
    .replace("No firewall on path", "Firewall на пути отсутствует");
}

function translateSimulationError(message?: string) {
  if (!message) return "Симуляция завершилась ошибкой.";
  const errors: Record<string, string> = {
    "sourceNodeId is required": "Выберите Client, от которого нужно отправить запрос.",
    "source node does not exist": "Выбранный узел-источник не существует.",
    "source node must be a client": "Узел-источник должен быть Client.",
    "source client is down": "Выбранный Client недоступен.",
    "Load balancer has no healthy backends available.": "У Load Balancer нет доступных серверов.",
    "DNS resolver is not available": "DNS-сервер недоступен."
  };
  if (errors[message]) return errors[message];
  if (message.startsWith("no route from")) return "Маршрут не найден.";
  if (message.startsWith("DNS NXDOMAIN")) return "DNS-запись не найдена.";
  return message;
}

function userFacingError(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    if (error.code === "validation_failed") return "Данные не прошли проверку. Проверьте параметры симуляции и топологию.";
    return translateSimulationError(error.message);
  }
  return error instanceof Error ? error.message : fallback;
}
