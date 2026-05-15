DROP TABLE IF EXISTS logs;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS ports;
DROP TABLE IF EXISTS nodes_info;

CREATE TABLE logs (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL CHECK (status IN ('completed', 'processing', 'failed')),
    nodes_count INTEGER DEFAULT 0,
    ports_count INTEGER DEFAULT 0,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    log_id INTEGER NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    node_type VARCHAR(16) NOT NULL CHECK (node_type IN ('host', 'switch')),
    node_guid VARCHAR(64) NOT NULL,
    num_ports INTEGER DEFAULT 0,
    UNIQUE(log_id, node_guid)
);

CREATE TABLE ports (
    id SERIAL PRIMARY KEY,
    node_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    port_guid VARCHAR(64) NOT NULL,
    port_num INTEGER NOT NULL,
    port_state INTEGER,
    port_phy_state INTEGER,
    UNIQUE (node_id, port_num)
);

CREATE TABLE nodes_info (
    id SERIAL PRIMARY KEY,
    node_id INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    system_info JSONB,
    sharp_info  JSONB
);

CREATE INDEX idx_nodes_log_id ON nodes(log_id);
CREATE INDEX idx_ports_node_id ON ports(node_id);