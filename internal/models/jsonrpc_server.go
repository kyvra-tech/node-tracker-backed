package models

import "time"

// JSONRPCServer represents a JSON-RPC public server
type JSONRPCServer struct {
	ID           int       `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Address      string    `json:"address" db:"address"`
	Network      string    `json:"network" db:"network"`
	Email        string    `json:"email" db:"email"`
	Website      string    `json:"website" db:"website"`
	Country      string    `json:"country" db:"country"`
	CountryCode  string    `json:"countryCode" db:"country_code"`
	City         string    `json:"city" db:"city"`
	Latitude     float64   `json:"latitude" db:"latitude"`
	Longitude    float64   `json:"longitude" db:"longitude"`
	OverallScore float64   `json:"overallScore" db:"overall_score"`
	IsActive     bool      `json:"isActive" db:"is_active"`
	IsVerified   bool      `json:"isVerified" db:"is_verified"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

// JSONRPCDailyStatus represents daily status for a JSON-RPC server
type JSONRPCDailyStatus struct {
	ID               int       `json:"id" db:"id"`
	ServerID         int       `json:"serverId" db:"server_id"`
	Date             time.Time `json:"date" db:"date"`
	Color            int       `json:"color" db:"color"`
	Attempts         int       `json:"attempts" db:"attempts"`
	Success          bool      `json:"success" db:"success"`
	ResponseTimeMs   int       `json:"responseTimeMs" db:"response_time_ms"`
	ErrorMsg         string    `json:"errorMsg" db:"error_msg"`
	BlockchainHeight int64     `json:"blockchainHeight" db:"blockchain_height"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
}

// JSONRPCServerResponse is the API response format for JSON-RPC servers
type JSONRPCServerResponse struct {
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	Address      string       `json:"address"`
	Network      string       `json:"network"`
	Email        string       `json:"email"`
	Website      string       `json:"website"`
	Country      string       `json:"country"`
	City         string       `json:"city"`
	Latitude     float64      `json:"latitude"`
	Longitude    float64      `json:"longitude"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
}
