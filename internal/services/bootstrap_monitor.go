package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/database"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
)

type BootstrapMonitor struct {
	db          *database.DB
	nodeChecker *NodeChecker
	logger      *logrus.Logger
}

func NewBootstrapMonitor(db *database.DB, nodeChecker *NodeChecker, logger *logrus.Logger) *BootstrapMonitor {
	return &BootstrapMonitor{
		db:          db,
		nodeChecker: nodeChecker,
		logger:      logger,
	}
}

func (bm *BootstrapMonitor) CheckAllNodes(ctx context.Context) error {
	nodes, err := bm.getActiveNodes()
	if err != nil {
		return fmt.Errorf("failed to get active nodes: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)

	for _, node := range nodes {
		if err := bm.checkSingleNode(ctx, node, today); err != nil {
			bm.logger.WithError(err).WithField("node_id", node.ID).Error("Failed to check node")
			continue
		}
	}

	// Update overall scores after checking all nodes
	if err := bm.updateOverallScores(); err != nil {
		bm.logger.WithError(err).Error("Failed to update overall scores")
	}

	return nil
}

func (bm *BootstrapMonitor) checkSingleNode(ctx context.Context, node *models.BootstrapNode, date time.Time) error {
	// Check if we already have a record for today
	exists, err := bm.hasStatusForDate(node.ID, date)
	if err != nil {
		return err
	}

	if exists {
		bm.logger.WithFields(logrus.Fields{
			"node_id": node.ID,
			"date":    date.Format("2006-01-02"),
		}).Info("Status already recorded for today")
		return nil
	}

	// Check the node
	result := bm.nodeChecker.CheckNode(ctx, node.Address)

	// Determine color based on success
	color := 0 // red/gray for failure
	if result.Success {
		color = 1 // green for success
	}

	// Save the result
	status := &models.DailyStatus{
		NodeID:   node.ID,
		Date:     date,
		Color:    color,
		Attempts: result.Attempts,
		Success:  result.Success,
		ErrorMsg: result.ErrorMsg,
	}

	return bm.saveDailyStatus(status)
}

func (bm *BootstrapMonitor) getActiveNodes() ([]*models.BootstrapNode, error) {
	query := `
        SELECT id, name, email, website, address, overall_score, is_active, created_at, updated_at
        FROM bootstrap_nodes 
        WHERE is_active = true
        ORDER BY id
    `

	rows, err := bm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*models.BootstrapNode
	for rows.Next() {
		node := &models.BootstrapNode{}
		err := rows.Scan(
			&node.ID, &node.Name, &node.Email, &node.Website, &node.Address,
			&node.OverallScore, &node.IsActive, &node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (bm *BootstrapMonitor) hasStatusForDate(nodeID int, date time.Time) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM daily_status WHERE node_id = $1 AND date = $2)`

	var exists bool
	err := bm.db.QueryRow(query, nodeID, date).Scan(&exists)
	return exists, err
}

func (bm *BootstrapMonitor) saveDailyStatus(status *models.DailyStatus) error {
	query := `
        INSERT INTO daily_status (node_id, date, color, attempts, success, error_msg)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (node_id, date) 
        DO UPDATE SET 
            color = EXCLUDED.color,
            attempts = EXCLUDED.attempts,
            success = EXCLUDED.success,
            error_msg = EXCLUDED.error_msg,
            created_at = NOW()
    `

	_, err := bm.db.Exec(query,
		status.NodeID, status.Date, status.Color,
		status.Attempts, status.Success, status.ErrorMsg,
	)

	return err
}

func (bm *BootstrapMonitor) updateOverallScores() error {
	query := `
        UPDATE bootstrap_nodes 
        SET overall_score = (
            SELECT COALESCE(
                ROUND(
                    (COUNT(CASE WHEN success = true THEN 1 END) * 100.0 / COUNT(*))::numeric, 2
                ), 0
            )
            FROM daily_status 
            WHERE node_id = bootstrap_nodes.id 
            AND date >= CURRENT_DATE - INTERVAL '30 days'
        ),
        updated_at = NOW()
        WHERE is_active = true
    `

	_, err := bm.db.Exec(query)
	return err
}

func (bm *BootstrapMonitor) GetBootstrapNodesWithStatus() ([]*models.BootstrapNodeResponse, error) {
	nodes, err := bm.getActiveNodes()
	if err != nil {
		return nil, err
	}

	var response []*models.BootstrapNodeResponse

	for _, node := range nodes {
		statuses, err := bm.getRecentStatuses(node.ID, 30) // Last 30 days
		if err != nil {
			bm.logger.WithError(err).WithField("node_id", node.ID).Error("Failed to get statuses")
			continue
		}

		nodeResponse := &models.BootstrapNodeResponse{
			Name:         node.Name,
			Email:        node.Email,
			Website:      node.Website,
			Address:      node.Address,
			Status:       statuses,
			OverallScore: node.OverallScore,
		}

		response = append(response, nodeResponse)
	}

	return response, nil
}

func (bm *BootstrapMonitor) getRecentStatuses(nodeID int, days int) ([]models.StatusItem, error) {
	query := `
        SELECT color, date
        FROM daily_status
        WHERE node_id = $1 AND date >= CURRENT_DATE - INTERVAL '%d days'
        ORDER BY date DESC
    `

	rows, err := bm.db.Query(fmt.Sprintf(query, days), nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.StatusItem
	for rows.Next() {
		var color int
		var date time.Time

		if err := rows.Scan(&color, &date); err != nil {
			return nil, err
		}

		status := models.StatusItem{
			Color: color,
			Date:  date.Format("2006-01-02"),
		}
		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}
