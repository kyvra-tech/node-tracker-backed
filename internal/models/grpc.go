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
	Email        string    `json:"email" db:"email"`
	Website      string    `json:"website" db:"website"`
	// Geographic fields (Phase 2)
	Country     string  `json:"country" db:"country"`
	CountryCode string  `json:"countryCode" db:"country_code"`
	City        string  `json:"city" db:"city"`
	Latitude    float64 `json:"latitude" db:"latitude"`
	Longitude   float64 `json:"longitude" db:"longitude"`
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
	Email        string       `json:"email"`
	Website      string       `json:"website"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
	// Geographic fields (Phase 2)
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}
