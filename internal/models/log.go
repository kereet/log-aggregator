package models

import "time"

type Log struct {
	ID         int       `json:"id" db:"id"`
	FilePath   string    `json:"file_path" db:"file_path"`
	Status     string    `json:"status" db:"status"`
	NodesCount int       `json:"nodes_count" db:"nodes_count"`
	PortsCount int       `json:"ports_count" db:"ports_count"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}

type ParseLogInput struct {
	FilePath string `json:"file_path"`
}

type ParseLogResponse struct {
	LogID int `json:"log_id"`
}

type Node struct {
	ID       int    `json:"id" db:"id"`
	LogID    int    `json:"log_id" db:"log_id"`
	Name     string `json:"name" db:"name"`
	NodeType string `json:"node_type" db:"node_type"`
	NodeGUID string `json:"node_guid" db:"node_guid"`
	NumPorts int    `json:"num_ports" db:"num_ports"`
}

type Port struct {
	ID           int    `json:"id" db:"id"`
	NodeID       int    `json:"node_id" db:"node_id"`
	NodeGUID     string `json:"-" db:"-"`
	PortGUID     string `json:"port_guid" db:"port_guid"`
	PortNum      int    `json:"port_num" db:"port_num"`
	PortState    int    `json:"port_state,omitempty" db:"port_state"`
	PortPhyState int    `json:"port_phy_state,omitempty" db:"port_phy_state"`
}

type NodeInfo struct {
	ID         int            `json:"id" db:"id"`
	NodeID     int            `json:"node_id" db:"node_id"`
	NodeGUID   string         `json:"-" db:"-"`
	SystemInfo map[string]any `json:"system_info,omitempty" db:"system_info"`
	SharpInfo  map[string]any `json:"sharp_info,omitempty" db:"sharp_info"`
}
