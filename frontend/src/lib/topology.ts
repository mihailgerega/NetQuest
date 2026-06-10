import { z } from "zod";
import type { TopologyDocument } from "@/lib/api/types";

export const topologyNodeSchema = z.object({
  id: z.string().min(1),
  name: z.string().optional(),
  type: z.enum(["client", "dns", "router", "firewall", "load_balancer", "server"]),
  status: z.enum(["healthy", "degraded", "down"]).optional(),
  position: z.object({ x: z.number(), y: z.number() }).optional(),
  config: z.record(z.unknown()).optional()
});

export const topologyLinkSchema = z.object({
  id: z.string().min(1),
  sourceNodeId: z.string().min(1),
  targetNodeId: z.string().min(1),
  status: z.enum(["active", "degraded", "down"]).optional(),
  config: z.record(z.unknown()).optional()
});

export const topologySchema = z.object({
  nodes: z.array(topologyNodeSchema).max(100),
  links: z.array(topologyLinkSchema).max(200)
});

export function demoTopology(): TopologyDocument {
  return {
    nodes: [
      { id: "client-1", name: "Client", type: "client", status: "healthy", position: { x: 80, y: 220 }, config: { ip: "10.0.1.10", hostname: "client-1" } },
      {
        id: "dns-1",
        name: "DNS",
        type: "dns",
        status: "healthy",
        position: { x: 180, y: 70 },
        config: { ip: "10.0.1.53", records: [{ name: "api.netquest.local", type: "A", value: "10.0.2.10", ttl: 300 }] }
      },
      { id: "router-1", name: "Router", type: "router", status: "healthy", position: { x: 270, y: 220 }, config: { ip: "10.0.1.1" } },
      {
        id: "firewall-1",
        name: "Firewall",
        type: "firewall",
        status: "healthy",
        position: { x: 470, y: 220 },
        config: {
          ip: "10.0.1.254",
          defaultPolicy: "deny",
          rules: [{ priority: 100, action: "allow", protocol: "tcp", source: "10.0.1.0/24", destination: "10.0.2.10/32", port: 443 }]
        }
      },
      {
        id: "lb-1",
        name: "Load Balancer",
        type: "load_balancer",
        status: "healthy",
        position: { x: 680, y: 220 },
        config: {
          ip: "10.0.2.10",
          algorithm: "round_robin",
          autoDiscoverConnectedServers: true,
          healthCheckEnabled: true,
          backends: [{ nodeId: "server-1", enabled: true, weight: 1 }, { nodeId: "server-2", enabled: true, weight: 1 }]
        }
      },
      { id: "server-1", name: "Server 1", type: "server", status: "healthy", position: { x: 900, y: 135 }, config: { ip: "10.0.2.21", serviceName: "api-1", port: 443, openPorts: [{ protocol: "tcp", port: 443, service: "HTTPS", status: "open" }] } },
      { id: "server-2", name: "Server 2", type: "server", status: "healthy", position: { x: 900, y: 305 }, config: { ip: "10.0.2.22", serviceName: "api-2", port: 443, openPorts: [{ protocol: "tcp", port: 443, service: "HTTPS", status: "open" }] } }
    ],
    links: [
      { id: "link-client-router", sourceNodeId: "client-1", targetNodeId: "router-1", status: "active", config: { latencyMs: 5, packetLossPercent: 0 } },
      { id: "link-client-dns", sourceNodeId: "client-1", targetNodeId: "dns-1", status: "active", config: { latencyMs: 2, packetLossPercent: 0 } },
      { id: "link-router-firewall", sourceNodeId: "router-1", targetNodeId: "firewall-1", status: "active", config: { latencyMs: 8, packetLossPercent: 0 } },
      { id: "link-firewall-lb", sourceNodeId: "firewall-1", targetNodeId: "lb-1", status: "active", config: { latencyMs: 12, packetLossPercent: 0 } },
      { id: "link-lb-server-1", sourceNodeId: "lb-1", targetNodeId: "server-1", status: "active", config: { latencyMs: 4, packetLossPercent: 0 } },
      { id: "link-lb-server-2", sourceNodeId: "lb-1", targetNodeId: "server-2", status: "active", config: { latencyMs: 6, packetLossPercent: 0 } }
    ]
  };
}
