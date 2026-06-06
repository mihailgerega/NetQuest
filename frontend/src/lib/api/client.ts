import type {
  AuthResponse,
  AdvisorIssue,
  HealthReport,
  Project,
  Quest,
  QuestAttempt,
  QuestCheckResult,
  SimulationEvent,
  SimulationSummary,
  StartSimulationRequest,
  Topology,
  TopologyDocument,
  TopologyValidationResult,
  User
} from "./types";

type ClientOptions = {
  baseUrl?: string;
  getToken?: () => string | null;
};

type RequestOptions = RequestInit & {
  authenticated?: boolean;
};

export class ApiError extends Error {
  status: number;
  code: string;
  details: unknown;

  constructor(status: number, code: string, message: string, details: unknown) {
    super(message);
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

export function createApiClient(options: ClientOptions = {}) {
  const baseUrl = options.baseUrl ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

  async function request<T>(path: string, init: RequestOptions = {}): Promise<T> {
    const headers = new Headers(init.headers);
    headers.set("Accept", "application/json");
    if (init.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    const token = options.getToken?.();
    if (init.authenticated !== false && token) {
      headers.set("Authorization", `Bearer ${token}`);
    }

    const response = await fetch(`${baseUrl}${path}`, {
      ...init,
      headers,
      credentials: init.credentials ?? "include",
      cache: "no-store"
    });

    const text = await response.text();
    const payload = text ? JSON.parse(text) : null;
    if (!response.ok) {
      const error = payload?.error;
      throw new ApiError(response.status, error?.code ?? "request_failed", error?.message ?? "Request failed", error?.details);
    }

    return payload as T;
  }

  return {
    health: () => request<HealthReport>("/health/ready", { authenticated: false }),
    register: (input: { email: string; password: string; displayName: string }) =>
      request<AuthResponse>("/api/v1/auth/register", { method: "POST", body: JSON.stringify(input), authenticated: false }),
    login: (input: { email: string; password: string }) =>
      request<AuthResponse>("/api/v1/auth/login", { method: "POST", body: JSON.stringify(input), authenticated: false }),
    demoLogin: () => request<AuthResponse>("/api/v1/auth/demo", { method: "POST", authenticated: false }),
    refresh: (refreshToken?: string) =>
      request<AuthResponse>("/api/v1/auth/refresh", { method: "POST", body: JSON.stringify({ refreshToken }), authenticated: false }),
    logout: (refreshToken?: string) =>
      request<{ ok: boolean }>("/api/v1/auth/logout", { method: "POST", body: JSON.stringify({ refreshToken }), authenticated: false }),
    me: () => request<{ user: User }>("/api/v1/auth/me"),
    listProjects: () => request<{ projects: Project[] }>("/api/v1/projects"),
    createProject: (input: { name: string; description?: string; visibility?: Project["visibility"] }) =>
      request<{ project: Project }>("/api/v1/projects", {
        method: "POST",
        body: JSON.stringify(input)
      }),
    validateTopology: (topologyId: string) =>
      request<TopologyValidationResult>(`/api/v1/topologies/${topologyId}/validate`, {
        method: "POST"
      }),
    createTopology: (projectId: string, input: { name: string; data: TopologyDocument }) =>
      request<{ topology: Topology; validation: TopologyValidationResult }>(`/api/v1/projects/${projectId}/topologies`, {
        method: "POST",
        body: JSON.stringify(input)
      }),
    listTopologies: (projectId: string) => request<{ topologies: Topology[] }>(`/api/v1/projects/${projectId}/topologies`),
    getTopology: (topologyId: string) => request<{ topology: Topology }>(`/api/v1/topologies/${topologyId}`),
    listQuests: () => request<{ quests: Quest[] }>("/api/v1/quests"),
    getQuest: (questId: string) => request<{ quest: Quest }>(`/api/v1/quests/${encodeURIComponent(questId)}`),
    startQuest: (questId: string) =>
      request<{ quest: Quest; attempt: QuestAttempt }>(`/api/v1/quests/${encodeURIComponent(questId)}/start`, { method: "POST" }),
    getQuestAttempt: (attemptId: string) => request<{ attempt: QuestAttempt }>(`/api/v1/quest-attempts/${encodeURIComponent(attemptId)}`),
    checkQuestAttempt: (attemptId: string, input: { topology: TopologyDocument; seed?: number }) =>
      request<{ attempt: QuestAttempt; result: QuestCheckResult }>(`/api/v1/quest-attempts/${encodeURIComponent(attemptId)}/check`, {
        method: "POST",
        body: JSON.stringify(input)
      }),
    resetQuestAttempt: (attemptId: string) =>
      request<{ quest: Quest; attempt: QuestAttempt }>(`/api/v1/quest-attempts/${encodeURIComponent(attemptId)}/reset`, { method: "POST" }),
    revealQuestHint: (attemptId: string, input: { revealedHintsCount: number }) =>
      request<{ attempt: QuestAttempt }>(`/api/v1/quest-attempts/${encodeURIComponent(attemptId)}/reveal-hint`, {
        method: "POST",
        body: JSON.stringify(input)
      }),
    analyzeTopology: (input: { topology: TopologyDocument; scenario?: StartSimulationRequest["scenario"] }) =>
      request<{ issues: AdvisorIssue[] }>("/api/v1/topologies/analyze", {
        method: "POST",
        body: JSON.stringify(input)
      }),
    analyzeStoredTopology: (topologyId: string, input: { scenario?: StartSimulationRequest["scenario"] } = {}) =>
      request<{ issues: AdvisorIssue[] }>(`/api/v1/topologies/${encodeURIComponent(topologyId)}/analyze`, {
        method: "POST",
        body: JSON.stringify(input)
      }),
    startSimulation: (input: StartSimulationRequest) =>
      request<{ simulation: unknown; events: SimulationEvent[]; summary: SimulationSummary }>("/api/v1/simulations", {
        method: "POST",
        body: JSON.stringify(input)
      }),
    simulationEvents: (simulationId: string) => request<{ events: SimulationEvent[] }>(`/api/v1/simulations/${simulationId}/events`),
    wsUrl: (simulationId: string, token: string) => {
      const wsBase = baseUrl.replace(/^http:/, "ws:").replace(/^https:/, "wss:");
      return `${wsBase}/api/v1/ws?simulationId=${encodeURIComponent(simulationId)}&token=${encodeURIComponent(token)}`;
    }
  };
}
