package model

import (
	"encoding/json"
	"time"
)

// Лимиты размера топологии по умолчанию. Они защищают движок симуляции:
// маршрутизация перебирает узлы и каналы, и огромный документ превратил бы
// один запрос в долгую работу CPU.
const (
	MaxNodes = 100
	MaxLinks = 200
)

// NodeType — тип сетевого узла на рабочем поле.
type NodeType string

// Типы узлов, которые понимает движок симуляции.
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

// Document — топология как JSON-документ: узлы и каналы связи.
//
// Config узлов и каналов — свободная map: у каждого типа узла свои настройки
// (DNS-записи, правила firewall, пул серверов Load Balancer, маршруты), и новые
// поля появляются без миграций. Читают их движок, валидатор и советник —
// каждый только нужные ключи.
type Document struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

// Node — узел топологии.
type Node struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Type     NodeType       `json:"type"`
	Status   string         `json:"status,omitempty"`
	Position *Position      `json:"position,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Position — координаты узла на рабочем поле. Движку не нужны, хранятся для фронтенда.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Link — канал связи между двумя узлами. Направления у канала нет:
// движок ходит по нему в обе стороны.
type Link struct {
	ID           string         `json:"id"`
	SourceNodeID string         `json:"sourceNodeId"`
	TargetNodeID string         `json:"targetNodeId"`
	Type         string         `json:"type,omitempty"`
	Status       string         `json:"status,omitempty"`
	Config       map[string]any `json:"config,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Topology — сохранённая версия топологии проекта.
//
// Версии неизменяемы: каждое сохранение рабочего поля создаёт новую строку
// с Version+1, а симуляция запускается по конкретному ID версии. Data — исходный
// JSON-документ без перекодирования: клиент получает его байт в байт.
type Topology struct {
	ID        string
	ProjectID string
	Version   int
	Name      string
	Data      json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil — версия не удалена
	CreatedBy *string    // nil — автор неизвестен (строки до появления колонки)
}

// ValidationResult — итог валидации топологии. Уходит клиенту как есть:
// в ответе POST /topologies/{id}/validate и в details ошибки 422.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

// ValidationError — одна найденная проблема: Path указывает на место в документе
// в стиле JSONPath ("nodes[2].config.backends[0].nodeId").
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}
