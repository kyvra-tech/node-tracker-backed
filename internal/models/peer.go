package models

import "time"

// ReachablePeer represents a discovered network peer
type ReachablePeer struct {
	ID                    int       `json:"id" db:"id"`
	PeerID                string    `json:"peerId" db:"peer_id"`
	Address               string    `json:"address" db:"address"`
	Protocol              string    `json:"protocol" db:"protocol"`
	UserAgent             string    `json:"userAgent" db:"user_agent"`
	LastSeen              time.Time `json:"lastSeen" db:"last_seen"`
	FirstSeen             time.Time `json:"firstSeen" db:"first_seen"`

	// Geographic
	IPAddress    string  `json:"ipAddress" db:"ip_address"`
	Country      string  `json:"country" db:"country"`
	CountryCode  string  `json:"countryCode" db:"country_code"`
	City         string  `json:"city" db:"city"`
	Latitude     float64 `json:"latitude" db:"latitude"`
	Longitude    float64 `json:"longitude" db:"longitude"`
	Timezone     string  `json:"timezone" db:"timezone"`
	ASN          string  `json:"asn" db:"asn"`
	Organization string  `json:"organization" db:"organization"`

	// Status
	IsReachable           bool    `json:"isReachable" db:"is_reachable"`
	ConnectionAttempts    int     `json:"connectionAttempts" db:"connection_attempts"`
	SuccessfulConnections int     `json:"successfulConnections" db:"successful_connections"`
	OverallScore          float64 `json:"overallScore" db:"overall_score"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// PeerDailyStatus represents daily status for a reachable peer
type PeerDailyStatus struct {
	ID             int       `json:"id" db:"id"`
	PeerID         int       `json:"peerId" db:"peer_id"`
	Date           time.Time `json:"date" db:"date"`
	Color          int       `json:"color" db:"color"`
	Attempts       int       `json:"attempts" db:"attempts"`
	Success        bool      `json:"success" db:"success"`
	ResponseTimeMs int       `json:"responseTimeMs" db:"response_time_ms"`
	ErrorMsg       string    `json:"errorMsg" db:"error_msg"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

// PeerResponse is the API response format for peers
type PeerResponse struct {
	ID           int          `json:"id"`
	PeerID       string       `json:"peerId"`
	Address      string       `json:"address"`
	Country      string       `json:"country"`
	CountryCode  string       `json:"countryCode"`
	City         string       `json:"city"`
	Latitude     float64      `json:"latitude"`
	Longitude    float64      `json:"longitude"`
	LastSeen     string       `json:"lastSeen"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
}

// NetworkStats represents network statistics
type NetworkStats struct {
	TotalNodes     int            `json:"totalNodes"`
	ReachableNodes int            `json:"reachableNodes"`
	CountriesCount int            `json:"countriesCount"`
	AvgUptime      float64        `json:"avgUptime"`
	TopCountries   []CountryStats `json:"topCountries"`
	GRPCNodes      int            `json:"grpcNodes"`
	JSONRPCNodes   int            `json:"jsonrpcNodes"`
	BootstrapNodes int            `json:"bootstrapNodes"`
}

// CountryStats represents statistics per country
type CountryStats struct {
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Count       int    `json:"count"`
}

// NetworkSnapshot represents a point-in-time snapshot of the network
type NetworkSnapshot struct {
	ID             int       `json:"id" db:"id"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	TotalNodes     int       `json:"totalNodes" db:"total_nodes"`
	ReachableNodes int       `json:"reachableNodes" db:"reachable_nodes"`
	CountriesCount int       `json:"countriesCount" db:"countries_count"`
	GRPCNodes      int       `json:"grpcNodes" db:"grpc_nodes"`
	JSONRPCNodes   int       `json:"jsonrpcNodes" db:"jsonrpc_nodes"`
	BootstrapNodes int       `json:"bootstrapNodes" db:"bootstrap_nodes"`
	SnapshotData   []byte    `json:"snapshotData" db:"snapshot_data"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

// MapNode represents a node for map display
type MapNode struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // bootstrap, grpc, jsonrpc, peer
	Coordinates []float64 `json:"coordinates"`
	Status      string    `json:"status"` // online, offline, unknown
	Country     string    `json:"country"`
	City        string    `json:"city,omitempty"`
}
