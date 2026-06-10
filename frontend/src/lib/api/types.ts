export type HealthStatus = "ok" | "degraded" | "error";

export type HealthReport = {
  status: HealthStatus;
  checks: Record<string, { status: HealthStatus; latencyMs: number; error?: string }>;
  timestamp: string;
};

export type Project = {
  id: string;
  ownerId: string;
  name: string;
  description?: string;
  visibility: "private" | "public" | "unlisted";
  createdAt: string;
  updatedAt: string;
};

export type User = {
  id: string;
  email: string;
  displayName: string;
  role: string;
};

export type AuthResponse = {
  user: User;
  accessToken: string;
  refreshToken?: string;
  tokenType: "Bearer";
  expiresIn: number;
};

export type TopologyNode = {
  id: string;
  name?: string;
  type: "client" | "dns" | "router" | "firewall" | "load_balancer" | "server";
  status?: "healthy" | "degraded" | "down";
  position?: { x: number; y: number };
  config?: Record<string, unknown>;
};

export type OpenPort = {
  protocol: "tcp" | "udp";
  port: number;
  service?: string;
  status?: "open" | "closed" | "filtered";
};

export type LoadBalancerBackend = {
  nodeId: string;
  weight?: number;
  enabled?: boolean;
  status?: string;
  healthy?: boolean;
  activeConnections?: number;
};

export type TopologyLink = {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  status?: "active" | "degraded" | "down";
  config?: Record<string, unknown>;
};

export type TopologyDocument = {
  nodes: TopologyNode[];
  links: TopologyLink[];
};

export type Topology = {
  id: string;
  projectId: string;
  version: number;
  name: string;
  data: TopologyDocument;
  createdAt: string;
  updatedAt: string;
};

export type TopologyValidationError = {
  path: string;
  message: string;
};

export type TopologyValidationResult = {
  valid: boolean;
  errors: TopologyValidationError[];
};

export type QuestDifficulty = "easy" | "medium" | "hard";
export type QuestAttemptStatus = "not_started" | "in_progress" | "completed" | "failed";

export type QuestCheckSpec = {
  id: string;
  type: string;
  title: string;
  hint?: string;
  requiredBackends?: string[];
  anyOfBackends?: string[];
};

export type ProgressiveHint = {
  title: string;
  body: string;
  level?: string;
  relatedCheckId?: string;
  actions?: string[];
};

export type GlossaryTerm = {
  term: string;
  definition: string;
};

export type Quest = {
  id: string;
  slug: string;
  title: string;
  difficulty: QuestDifficulty;
  category: string;
  description: string;
  goal: string;
  learningObjectives: string[];
  initialTopology: TopologyDocument;
  expectedChecks: QuestCheckSpec[];
  hints: string[];
  progressiveHints: ProgressiveHint[];
  afterSolutionExplanation: string;
  glossaryTerms: GlossaryTerm[];
  realWorldImportance?: string;
  successMessage: string;
  failureMessage: string;
  estimatedMinutes: number;
  attemptStatus?: QuestAttemptStatus;
};

export type QuestAttempt = {
  id: string;
  questId: string;
  status: QuestAttemptStatus;
  attemptsCount: number;
  revealedHintsCount: number;
  lastCheckResult?: QuestCheckResult;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type QuestCheckItem = {
  id: string;
  passed: boolean;
  message: string;
  details?: Record<string, unknown>;
};

export type QuestCheckResult = {
  passed: boolean;
  score: number;
  checks: QuestCheckItem[];
  hints: string[];
  afterSolutionExplanation?: string;
};

export type AdvisorIssue = {
  severity: "info" | "warning" | "error";
  category: string;
  code: string;
  title: string;
  message: string;
  affectedNodeId?: string;
  affectedLinkId?: string;
  suggestedFix: string;
};

export type StartSimulationRequest = {
  projectId: string;
  topologyId: string;
  seed?: number;
  scenario: {
    type: string;
    sourceNodeId?: string;
    target?: string;
    method?: string;
    metadata?: Record<string, unknown>;
  };
};

export type SimulationEvent = {
  id: string;
  simulationId: string;
  sequenceNumber: number;
  type: string;
  timestampMs: number;
  sourceNodeId?: string;
  targetNodeId?: string;
  packetId?: string;
  severity: "info" | "warn" | "error";
  message: string;
  details: Record<string, unknown>;
};

export type SimulationSummary = {
  packetId?: string;
  scenario: string;
  status: "pending" | "running" | "completed" | "failed";
  seed?: number;
  source?: string;
  sourceNodeId?: string;
  destination?: string;
  resolvedIp?: string;
  selectedBackend?: string;
  selectedBackendNodeId?: string;
  selectedBackendName?: string;
  healthyBackends?: string[];
  skippedBackends?: Array<{ nodeId: string; name?: string; reason: string }>;
  failover?: boolean;
  totalLatencyMs: number;
  latencyBreakdown?: Array<{ stage: string; label: string; durationMs: number; details?: Record<string, unknown> }>;
  latencyFormula?: string;
  protocolDetails?: Record<string, unknown>;
  path: string[];
  decisions: string[];
  errors: string[];
  metadata?: Record<string, unknown>;
};
