package models

import (
	"time"
)

type BootstrapNode struct {
	ID           int       `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Email        string    `json:"email" db:"email"`
	Website      string    `json:"website" db:"website"`
	Address      string    `json:"address" db:"address"`
	OverallScore float64   `json:"overallScore" db:"overall_score"`
	IsActive     bool      `json:"isActive" db:"is_active"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type DailyStatus struct {
	ID        int       `json:"id" db:"id"`
	NodeID    int       `json:"nodeId" db:"node_id"`
	Date      time.Time `json:"date" db:"date"`
	Color     int       `json:"color" db:"color"` // 0 = red/gray, 1,2 = green
	Attempts  int       `json:"attempts" db:"attempts"`
	Success   bool      `json:"success" db:"success"`
	ErrorMsg  string    `json:"errorMsg" db:"error_msg"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type BootstrapNodeResponse struct {
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Website      string       `json:"website"`
	Address      string       `json:"address"`
	Status       []StatusItem `json:"status"`
	OverallScore float64      `json:"overallScore"`
}

type StatusItem struct {
	Color int    `json:"color"`
	Date  string `json:"date"`
}
