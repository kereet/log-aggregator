package handlers

import (
	"encoding/json"
	"log"
	"log-aggregator/internal/aggregator"
	"log-aggregator/internal/database"
	"log-aggregator/internal/models"
	"log-aggregator/internal/parser"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type Handlers struct {
	store *database.LogStore
}

func NewHandlers(store *database.LogStore) *Handlers {
	return &Handlers{store}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{"error": message})
}

func (h *Handlers) ParseLog(w http.ResponseWriter, r *http.Request) {
	var input models.ParseLogInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request data")
		return
	}

	if strings.TrimSpace(input.FilePath) == "" {
		respondWithError(w, http.StatusBadRequest, "File path is required")
		return
	}

	logID, err := h.store.CreateLog(input.FilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	parseResult, err := parser.ParseZip(input.FilePath)
	if err != nil {
		if err := h.store.UpdateLogStatus(logID, "failed", 0, 0); err != nil {
			slog.Error("failed to update log status", "error", err)
		}
		respondWithError(w, http.StatusBadRequest, "Parse error: "+err.Error())
		return
	}

	nodeIDByGUID := make(map[string]int)
	for i := range parseResult.Nodes {
		parseResult.Nodes[i].LogID = logID
		if err := h.store.InsertNode(logID, &parseResult.Nodes[i]); err != nil {
			if err := h.store.UpdateLogStatus(logID, "failed", 0, 0); err != nil {
				slog.Error("failed to update log status", "error", err)
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to save node: "+err.Error())
			return
		}
		nodeIDByGUID[parseResult.Nodes[i].NodeGUID] = parseResult.Nodes[i].ID
	}

	for i := range parseResult.Ports {
		nodeID, ok := nodeIDByGUID[parseResult.Ports[i].NodeGUID]
		if !ok {
			log.Printf("Node not found for port %s", parseResult.Ports[i].PortGUID)
			continue
		}
		parseResult.Ports[i].NodeID = nodeID
		if err := h.store.InsertPort(nodeID, &parseResult.Ports[i]); err != nil {
			if err := h.store.UpdateLogStatus(logID, "failed", 0, 0); err != nil {
				slog.Error("failed to update log status", "error", err)
			}
			respondWithError(w, http.StatusInternalServerError, "Failed to save port: "+err.Error())
			return
		}
	}

	for i := range parseResult.NodeInfos {
		nodeID, ok := nodeIDByGUID[parseResult.NodeInfos[i].NodeGUID]
		if !ok {
			slog.Error("Node not found for node info",
				slog.String("GUID", parseResult.NodeInfos[i].NodeGUID),
			)
			continue
		}
		parseResult.NodeInfos[i].NodeID = nodeID
		if err := h.store.InsertNodeInfo(&parseResult.NodeInfos[i]); err != nil {
			slog.Error("Failed to save node info",
				slog.Int("node_id", nodeID),
				slog.String("error", err.Error()),
			)
		}
	}

	nodesCount := len(parseResult.Nodes)
	portsCount := len(parseResult.Ports)

	if err := h.store.UpdateLogStatus(logID, "completed", nodesCount, portsCount); err != nil {
		slog.Error("failed to update log status", "error", err)
	}
	respondWithJSON(w, http.StatusCreated, models.ParseLogResponse{LogID: logID})
}

func (h *Handlers) GetTopology(w http.ResponseWriter, r *http.Request) {
	logID, err := strconv.Atoi(r.PathValue("log_id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid log_id")
	}
	nodes, err := h.store.GetNodesByLogID(logID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}
	portsByNode := make(map[int][]models.Port)
	for _, node := range nodes {
		ports, err := h.store.GetPortsByNodeID(node.ID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		portsByNode[node.ID] = ports
	}
	topology := aggregator.BuildTopology(logID, nodes, portsByNode)
	respondWithJSON(w, http.StatusOK, topology)
}

func (h *Handlers) GetNode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("node_id")
	if idStr == "" {
		respondWithError(w, http.StatusBadRequest, "Missing node_id")
		return
	}

	nodeID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid node_id")
		return
	}

	node, err := h.store.GetNodeByID(nodeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Node not found")
		return
	}

	ports, err := h.store.GetPortsByNodeID(nodeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}
	nodeInfo, err := h.store.GetNodeInfoByNodeID(nodeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	response := struct {
		ID         int            `json:"id"`
		Name       string         `json:"name"`
		NodeType   string         `json:"node_type"`
		NodeGUID   string         `json:"node_guid"`
		NumPorts   int            `json:"num_ports"`
		Ports      []models.Port  `json:"ports"`
		SystemInfo map[string]any `json:"system_info,omitempty"`
		SharpInfo  map[string]any `json:"sharp_info,omitempty"`
	}{
		ID:       node.ID,
		Name:     node.Name,
		NodeType: node.NodeType,
		NodeGUID: node.NodeGUID,
		NumPorts: node.NumPorts,
		Ports:    ports,
	}

	if nodeInfo != nil {
		response.SystemInfo = nodeInfo.SystemInfo
		response.SharpInfo = nodeInfo.SharpInfo
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) GetPorts(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("node_id")
	if idStr == "" {
		respondWithError(w, http.StatusBadRequest, "Missing node_id")
		return
	}

	nodeID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid node_id")
		return
	}

	ports, err := h.store.GetPortsByNodeID(nodeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get ports")
		return
	}

	if ports == nil {
		ports = []models.Port{}
	}

	respondWithJSON(w, http.StatusOK, ports)
}

func (h *Handlers) GetLogInfo(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("log_id")
	if idStr == "" {
		respondWithError(w, http.StatusBadRequest, "Missing log_id")
		return
	}

	logID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid log_id")
		return
	}

	log, err := h.store.GetLogByID(logID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Log not found")
		return
	}

	respondWithJSON(w, http.StatusOK, log)
}
