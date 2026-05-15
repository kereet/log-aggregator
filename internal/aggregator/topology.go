package aggregator

import "log-aggregator/internal/models"

type Topology struct {
	LogID int                   `json:"log_id"`
	Nodes []models.Node         `json:"nodes"`
	Ports map[int][]models.Port `json:"ports"`
}

func BuildTopology(logID int, nodes []models.Node, portsByNode map[int][]models.Port) *Topology {
	return &Topology{
		LogID: logID,
		Nodes: nodes,
		Ports: portsByNode,
	}
}
