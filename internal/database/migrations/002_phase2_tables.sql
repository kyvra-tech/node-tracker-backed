-- Phase 2: Reachable Nodes - Database Migrations
-- File: 002_phase2_tables.sql

-- ============================================
-- NEW TABLES
-- ============================================

-- Reachable peers discovered by the crawler
CREATE TABLE IF NOT EXISTS reachable_peers (
    id SERIAL PRIMARY KEY,
    peer_id VARCHAR(255) NOT NULL UNIQUE,
    address TEXT NOT NULL,
    protocol VARCHAR(50),
    user_agent VARCHAR(255),
    last_seen TIMESTAMP WITH TIME ZONE,
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Geographic data
    ip_address VARCHAR(45),
    country VARCHAR(100),
    country_code VARCHAR(2),
    city VARCHAR(100),
    latitude DECIMAL(10, 6),
    longitude DECIMAL(10, 6),
    timezone VARCHAR(50),
    asn VARCHAR(50),
    organization VARCHAR(255),
    
    -- Status
    is_reachable BOOLEAN DEFAULT true,
    connection_attempts INTEGER DEFAULT 0,
    successful_connections INTEGER DEFAULT 0,
    overall_score DECIMAL(5, 2) DEFAULT 0.00,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Daily status for reachable peers
CREATE TABLE IF NOT EXISTS peer_daily_status (
    id SERIAL PRIMARY KEY,
    peer_id INTEGER NOT NULL REFERENCES reachable_peers(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    color INTEGER NOT NULL CHECK (color IN (0, 1, 2)),
    attempts INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT false,
    response_time_ms INTEGER,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(peer_id, date)
);

-- JSON-RPC public servers
CREATE TABLE IF NOT EXISTS jsonrpc_servers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL UNIQUE,
    network VARCHAR(20) NOT NULL DEFAULT 'mainnet',
    email VARCHAR(255),
    website VARCHAR(255),
    
    -- Geographic data
    country VARCHAR(100),
    country_code VARCHAR(2),
    city VARCHAR(100),
    latitude DECIMAL(10, 6),
    longitude DECIMAL(10, 6),
    
    -- Status
    overall_score DECIMAL(5, 2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Daily status for JSON-RPC servers
CREATE TABLE IF NOT EXISTS jsonrpc_daily_status (
    id SERIAL PRIMARY KEY,
    server_id INTEGER NOT NULL REFERENCES jsonrpc_servers(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    color INTEGER NOT NULL CHECK (color IN (0, 1, 2)),
    attempts INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT false,
    response_time_ms INTEGER,
    error_msg TEXT,
    blockchain_height BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(server_id, date)
);

-- Network snapshots (like BitNodes)
CREATE TABLE IF NOT EXISTS network_snapshots (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_nodes INTEGER NOT NULL,
    reachable_nodes INTEGER NOT NULL,
    countries_count INTEGER,
    grpc_nodes INTEGER DEFAULT 0,
    jsonrpc_nodes INTEGER DEFAULT 0,
    bootstrap_nodes INTEGER DEFAULT 0,
    snapshot_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Node registration requests (pending approval)
CREATE TABLE IF NOT EXISTS node_registrations (
    id SERIAL PRIMARY KEY,
    node_type VARCHAR(20) NOT NULL CHECK (node_type IN ('grpc', 'jsonrpc')),
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    network VARCHAR(20) NOT NULL DEFAULT 'mainnet',
    email VARCHAR(255) NOT NULL,
    website VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by VARCHAR(255)
);

-- ============================================
-- ADD GEOGRAPHIC FIELDS TO EXISTING TABLES
-- ============================================

-- Add geographic fields to grpc_servers
ALTER TABLE grpc_servers ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE grpc_servers ADD COLUMN IF NOT EXISTS country_code VARCHAR(2);
ALTER TABLE grpc_servers ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE grpc_servers ADD COLUMN IF NOT EXISTS latitude DECIMAL(10, 6);
ALTER TABLE grpc_servers ADD COLUMN IF NOT EXISTS longitude DECIMAL(10, 6);

-- Add geographic fields to bootstrap_nodes
ALTER TABLE bootstrap_nodes ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE bootstrap_nodes ADD COLUMN IF NOT EXISTS country_code VARCHAR(2);
ALTER TABLE bootstrap_nodes ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE bootstrap_nodes ADD COLUMN IF NOT EXISTS latitude DECIMAL(10, 6);
ALTER TABLE bootstrap_nodes ADD COLUMN IF NOT EXISTS longitude DECIMAL(10, 6);

-- ============================================
-- INDEXES FOR PERFORMANCE
-- ============================================

CREATE INDEX IF NOT EXISTS idx_reachable_peers_is_reachable ON reachable_peers(is_reachable);
CREATE INDEX IF NOT EXISTS idx_reachable_peers_country ON reachable_peers(country_code);
CREATE INDEX IF NOT EXISTS idx_reachable_peers_last_seen ON reachable_peers(last_seen);
CREATE INDEX IF NOT EXISTS idx_peer_daily_status_date ON peer_daily_status(date);
CREATE INDEX IF NOT EXISTS idx_peer_daily_status_peer_date ON peer_daily_status(peer_id, date);
CREATE INDEX IF NOT EXISTS idx_jsonrpc_servers_network ON jsonrpc_servers(network);
CREATE INDEX IF NOT EXISTS idx_jsonrpc_servers_active ON jsonrpc_servers(is_active);
CREATE INDEX IF NOT EXISTS idx_jsonrpc_daily_status_date ON jsonrpc_daily_status(date);
CREATE INDEX IF NOT EXISTS idx_jsonrpc_daily_status_server_date ON jsonrpc_daily_status(server_id, date);
CREATE INDEX IF NOT EXISTS idx_network_snapshots_timestamp ON network_snapshots(timestamp);
CREATE INDEX IF NOT EXISTS idx_node_registrations_status ON node_registrations(status);
CREATE INDEX IF NOT EXISTS idx_node_registrations_type ON node_registrations(node_type);

-- ============================================
-- GRANT PERMISSIONS
-- ============================================

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO pactus_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO pactus_user;