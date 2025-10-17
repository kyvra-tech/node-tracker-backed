package models

import (
	"time"
)

type GRPCServer struct {
	ID           int       `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Address      string    `json:"address" db:"address"`
	Network      string    `json:"network" db:"network"` // mainnet or testnet
	OverallScore float64   `json:"overallScore" db:"overall_score"`
	IsActive     bool      `json:"isActive" db:"is_active"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type GRPCDailyStatus struct {
	ID             int       `json:"id" db:"id"`
	ServerID       int       `json:"serverId" db:"server_id"`
	Date           time.Time `json:"date" db:"date"`
	Color          int       `json:"color" db:"color"` // 0 = grey, 1 = green
	Attempts       int       `json:"attempts" db:"attempts"`
	Success        bool      `json:"success" db:"success"`
	ErrorMsg       string    `json:"errorMsg" db:"error_msg"`
	ResponseTimeMs int       `json:"responseTimeMs" db:"response_time_ms"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

type GRPCServerResponse struct {
	Name         string       `json:"name"`
	Address      string       `json:"address"`
	Network      string       `json:"network"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
}
