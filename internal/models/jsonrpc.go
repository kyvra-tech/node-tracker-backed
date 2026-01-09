package models

import "time"

// JsonRPCNodeResponse represents a node with its status
type JsonRPCNodeResponse struct {
	Name         string       `json:"name"`
	Address      string       `json:"address"`
	Network      string       `json:"network"`
	Email        string       `json:"email"`
	Website      string       `json:"website"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
	Country      string       `json:"country"`
	City         string       `json:"city"`
	Latitude     float64      `json:"latitude"`
	Longitude    float64      `json:"longitude"`
}

// StatusResponse represents a status check response
type StatusResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// CountResponse represents a count response
type CountResponse struct {
	Total     int       `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// SyncResponse represents a sync response
type SyncResponse struct {
	Message      string    `json:"message"`
	TotalServers int       `json:"total_servers"`
	Timestamp    time.Time `json:"timestamp"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}
