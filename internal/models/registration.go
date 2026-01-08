package models

import "time"

// NodeRegistration represents a node registration request
type NodeRegistration struct {
	ID              int        `json:"id" db:"id"`
	NodeType        string     `json:"nodeType" db:"node_type"` // grpc, jsonrpc
	Name            string     `json:"name" db:"name"`
	Address         string     `json:"address" db:"address"`
	Network         string     `json:"network" db:"network"`
	Email           string     `json:"email" db:"email"`
	Website         string     `json:"website" db:"website"`
	Status          string     `json:"status" db:"status"` // pending, approved, rejected
	RejectionReason string     `json:"rejectionReason" db:"rejection_reason"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	ReviewedAt      *time.Time `json:"reviewedAt" db:"reviewed_at"`
	ReviewedBy      string     `json:"reviewedBy" db:"reviewed_by"`
}

// RegistrationRequest is the API request for node registration
type RegistrationRequest struct {
	NodeType string `json:"nodeType" binding:"required,oneof=grpc jsonrpc"`
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Address  string `json:"address" binding:"required"`
	Network  string `json:"network" binding:"required,oneof=mainnet testnet"`
	Email    string `json:"email" binding:"required,email"`
	Website  string `json:"website"`
}

// RegistrationResponse is the API response for registration
type RegistrationResponse struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
