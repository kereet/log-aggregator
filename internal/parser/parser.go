package parser

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"log"
	"log-aggregator/internal/models"
	"strconv"
	"strings"
)

type ParseResult struct {
	Nodes     []models.Node
	Ports     []models.Port
	NodeInfos []models.NodeInfo
}

func ParseZip(path string) (*ParseResult, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	result := &ParseResult{
		Nodes:     []models.Node{},
		Ports:     []models.Port{},
		NodeInfos: []models.NodeInfo{},
	}

	nodeByGUID := make(map[string]*models.Node)
	portsByNodeGUID := make(map[string][]models.Port)
	systemInfoByGUID := make(map[string]map[string]any)

	for _, file := range r.File {
		if file.FileInfo().IsDir() {
			continue
		}

		err := func() error {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			switch {
			case strings.HasSuffix(file.Name, ".db_csv"):
				nodes, ports, sysInfo, err := parseDBCSV(content)
				if err != nil {
					return err
				}
				for i := range nodes {
					nodeByGUID[nodes[i].NodeGUID] = &nodes[i]
				}
				for guid, ports := range ports {
					portsByNodeGUID[guid] = ports
				}
				for guid, info := range sysInfo {
					systemInfoByGUID[guid] = info
				}
				log.Println()

			case strings.HasSuffix(file.Name, ".sharp_an_info"):
				sharpInfoByGUID, err := parseSharpInfo(content)
				if err != nil {
					return err
				}
				for guid, sharpInfo := range sharpInfoByGUID {
					if systemInfoByGUID[guid] == nil {
						systemInfoByGUID[guid] = make(map[string]any)
					}
					for k, v := range sharpInfo {
						systemInfoByGUID[guid][k] = v
					}
				}
			}
			return nil
		}()

		if err != nil {
			return nil, err
		}
	}

	result.Nodes = make([]models.Node, 0, len(nodeByGUID))
	for _, node := range nodeByGUID {
		result.Nodes = append(result.Nodes, *node)
	}

	result.Ports = make([]models.Port, 0)
	for _, ports := range portsByNodeGUID {
		result.Ports = append(result.Ports, ports...)
	}

	for guid, info := range systemInfoByGUID {
		var nodeID int
		for _, node := range result.Nodes {
			if node.NodeGUID == guid {
				nodeID = node.ID
				break
			}
		}
		if nodeID != 0 {
			result.NodeInfos = append(result.NodeInfos, models.NodeInfo{
				NodeID:     nodeID,
				SystemInfo: info,
				SharpInfo:  nil,
			})
		}
	}

	return result, nil
}

func parseDBCSV(content []byte) ([]models.Node, map[string][]models.Port, map[string]map[string]any, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var nodes []models.Node
	portsByGUID := make(map[string][]models.Port)
	systemInfoByGUID := make(map[string]map[string]any)

	var inBlock bool
	var currentBlockLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" && !inBlock {
			continue
		}

		if strings.HasPrefix(line, "START_") {
			inBlock = true
			currentBlockLines = []string{}
			continue
		}

		if strings.HasPrefix(line, "END_") && inBlock {
			inBlock = false
			blockType := strings.TrimPrefix(line, "END_")

			switch blockType {
			case "NODES":
				var err error
				nodes, err = parseNodesBlock(currentBlockLines)
				if err != nil {
					return nil, nil, nil, err
				}
			case "PORTS":
				var err error
				portsByGUID, err = parsePortsBlock(currentBlockLines)
				if err != nil {
					return nil, nil, nil, err
				}
			case "SYSTEM_GENERAL_INFORMATION":
				var err error
				systemInfoByGUID, err = parseSystemInfoBlock(currentBlockLines)
				if err != nil {
					return nil, nil, nil, err
				}
			}
			continue
		}

		if inBlock {
			currentBlockLines = append(currentBlockLines, line)
		}
	}

	return nodes, portsByGUID, systemInfoByGUID, nil
}

func parseNodesBlock(lines []string) ([]models.Node, error) {
	if len(lines) < 2 {
		return []models.Node{}, nil
	}

	var nodes []models.Node

	for i, line := range lines[1:] {
		row := parseCSVLine(line)
		if len(row) < 8 {
			return nil, NewParseError("NODES", i+2, "expected at least 8 columns")
		}

		nodeType := "host"
		if row[2] == "2" {
			nodeType = "switch"
		}

		numPorts, _ := strconv.Atoi(row[1])

		node := models.Node{
			Name:     strings.Trim(row[0], `"`),
			NodeType: nodeType,
			NodeGUID: row[6],
			NumPorts: numPorts,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func parsePortsBlock(lines []string) (map[string][]models.Port, error) {
	//if len(lines) < 2 {
	//	return make(map[string][]models.Port), nil
	//}

	portsByGUID := make(map[string][]models.Port)

	for i, line := range lines[1:] {
		row := parseCSVLine(line)
		if len(row) < 26 {
			return nil, NewParseError("PORTS", i+2, "expected at least 26 columns")
		}

		nodeGUID := row[0]
		portNum, _ := strconv.Atoi(row[2])
		portState, _ := strconv.Atoi(row[25])
		portPhyState, _ := strconv.Atoi(row[24])

		port := models.Port{
			PortGUID:     row[1],
			PortNum:      portNum,
			PortState:    portState,
			PortPhyState: portPhyState,
		}

		portsByGUID[nodeGUID] = append(portsByGUID[nodeGUID], port)
	}

	return portsByGUID, nil
}

func parseSystemInfoBlock(lines []string) (map[string]map[string]any, error) {
	if len(lines) < 2 {
		return make(map[string]map[string]any), nil
	}

	result := make(map[string]map[string]any)
	headers := strings.Split(lines[0], ",")

	for _, line := range lines[1:] {
		row := parseCSVLine(line)
		if len(row) < 2 {
			continue
		}
		nodeGUID := row[0]

		info := make(map[string]any)
		for j, val := range row {
			if j >= len(headers) {
				break
			}
			key := headers[j]
			if key == "NodeGuid" {
				continue
			}
			info[key] = strings.Trim(val, `"`)
		}
		result[nodeGUID] = info
	}

	return result, nil
}

func parseSharpInfo(content []byte) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var currentGUID string
	var currentParams map[string]any

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "---") {
			if currentGUID != "" && currentParams != nil {
				result[currentGUID] = currentParams
			}
			currentGUID = ""
			currentParams = nil
			continue
		}

		if strings.HasPrefix(line, "SW_GUID=") {
			currentGUID = strings.TrimPrefix(line, "SW_GUID=")
			currentParams = make(map[string]any)
			continue
		}

		if currentGUID != "" && strings.Contains(line, "=") && !strings.HasPrefix(line, "SW_GUID=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if intVal, err := strconv.Atoi(val); err == nil {
					currentParams[key] = intVal
				} else {
					currentParams[key] = val
				}
			}
		}
	}

	if currentGUID != "" && currentParams != nil {
		result[currentGUID] = currentParams
	}

	return result, nil
}

func parseCSVLine(line string) []string {
	r := csv.NewReader(strings.NewReader(line))
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	record, _ := r.Read()
	if len(record) == 0 {
		return strings.Split(line, ",")
	}
	return record
}

func NewParseError(section string, lineNum int, msg string) error {
	return &ParseError{
		Section: section,
		LineNum: lineNum,
		Message: msg,
	}
}

type ParseError struct {
	Section string
	LineNum int
	Message string
}

func (e *ParseError) Error() string {
	return "parse error in " + e.Section + " at line " + strconv.Itoa(e.LineNum) + ": " + e.Message
}
