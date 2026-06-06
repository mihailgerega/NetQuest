package topology

import (
	"encoding/json"
	"time"
)

const (
	MaxNodes = 100
	MaxLinks = 200
)

type NodeType string

const (
	NodeTypeClient       NodeType = "client"
	NodeTypeServer       NodeType = "server"
	NodeTypeRouter       NodeType = "router"
	NodeTypeSwitch       NodeType = "switch"
	NodeTypeDNS          NodeType = "dns"
	NodeTypeFirewall     NodeType = "firewall"
	NodeTypeLoadBalancer NodeType = "load_balancer"
	NodeTypeProxy        NodeType = "proxy"
	NodeTypeNATGateway   NodeType = "nat_gateway"
	NodeTypeVPNGateway   NodeType = "vpn_gateway"
	NodeTypeDatabase     NodeType = "database"
	NodeTypeInternet     NodeType = "internet"
)

type Document struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Type     NodeType       `json:"type"`
	Status   string         `json:"status,omitempty"`
	Position *Position      `json:"position,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Link struct {
	ID           string         `json:"id"`
	SourceNodeID string         `json:"sourceNodeId"`
	TargetNodeID string         `json:"targetNodeId"`
	Type         string         `json:"type,omitempty"`
	Status       string         `json:"status,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Topology struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectId"`
	Version   int             `json:"version"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	DeletedAt *time.Time      `json:"deletedAt,omitempty"`
	CreatedBy *string         `json:"createdBy,omitempty"`
}

type CreateRequest struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}
