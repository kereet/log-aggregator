package database

import (
	"log-aggregator/internal/models"

	"github.com/jmoiron/sqlx"
)

type LogStore struct {
	db *sqlx.DB
}

func NewLogStore(db *sqlx.DB) *LogStore {
	return &LogStore{db: db}
}

func (s *LogStore) CreateLog(filePath string) (int, error) {
	var id int
	err := s.db.QueryRow(`
		INSERT INTO logs (file_path, status, nodes_count, ports_count)
		VALUES ($1, 'processing', 0, 0)
		RETURNING id
	`, filePath).Scan(&id)
	return id, err
}

func (s *LogStore) GetLogByID(id int) (*models.Log, error) {
	var log models.Log
	err := s.db.Get(&log, `
		SELECT id, file_path, status, nodes_count, ports_count, uploaded_at
		FROM logs WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (s *LogStore) UpdateLogStatus(id int, status string, nodesCount, portsCount int) error {
	_, err := s.db.Exec(`
		UPDATE logs 
		SET status = $1, nodes_count = $2, ports_count = $3
		WHERE id = $4
	`, status, nodesCount, portsCount, id)
	return err
}

func (s *LogStore) GetNodeByID(id int) (*models.Node, error) {
	var node models.Node
	err := s.db.Get(&node, `
		SELECT id, log_id, name, node_type, node_guid, num_ports
		FROM nodes WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *LogStore) GetNodesByLogID(logID int) ([]models.Node, error) {
	var nodes []models.Node
	err := s.db.Select(&nodes, `
		SELECT id, log_id, name, node_type, node_guid, num_ports
		FROM nodes WHERE log_id = $1
	`, logID)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (s *LogStore) InsertNode(logID int, node *models.Node) error {
	return s.db.QueryRowx(`
		INSERT INTO nodes (log_id, name, node_type, node_guid, num_ports)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, logID, node.Name, node.NodeType, node.NodeGUID, node.NumPorts).Scan(&node.ID)
}

func (s *LogStore) GetPortsByNodeID(nodeID int) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.Select(&ports, `
		SELECT id, node_id, port_guid, port_num, port_state, port_phy_state
		FROM ports WHERE node_id = $1
	`, nodeID)
	if err != nil {
		return nil, err
	}
	return ports, nil
}

func (s *LogStore) InsertPort(nodeID int, port *models.Port) error {
	return s.db.QueryRowx(`
		INSERT INTO ports (node_id, port_guid, port_num, port_state, port_phy_state)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, nodeID, port.PortGUID, port.PortNum, port.PortState, port.PortPhyState).Scan(&port.ID)
}

func (s *LogStore) GetNodeInfoByNodeID(nodeID int) (*models.NodeInfo, error) {
	var info models.NodeInfo
	err := s.db.Get(&info, `
		SELECT id, node_id, system_info, sharp_info
		FROM nodes_info WHERE node_id = $1
	`, nodeID)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *LogStore) InsertNodeInfo(nodeID int, systemInfo, sharpInfo map[string]any) error {
	_, err := s.db.Exec(`
		INSERT INTO nodes_info (node_id, system_info, sharp_info)
		VALUES ($1, $2, $3)
	`, nodeID, systemInfo, sharpInfo)
	return err
}
