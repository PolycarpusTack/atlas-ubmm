// services/backlog-service/internal/adapters/db/metrics_repository.go

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/ubmm/backlog-service/internal/domain/model"
	"github.com/ubmm/backlog-service/internal/domain/repository"
)

// MetricsRepository implements the metrics repository interface
type MetricsRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *sqlx.DB, logger *zap.Logger) repository.MetricsRepository {
	return &MetricsRepository{
		db:     db,
		logger: logger,
	}
}

// GetBacklogSize retrieves the current backlog size metrics
func (r *MetricsRepository) GetBacklogSize(ctx context.Context) (map[model.ItemType]int, error) {
	query := `
		SELECT type, COUNT(*) as count
		FROM backlog_items
		WHERE status != $1
		GROUP BY type
	`

	rows, err := r.db.QueryContext(ctx, query, model.ItemStatusDone)
	if err != nil {
		return nil, fmt.Errorf("failed to query backlog size: %w", err)
	}
	defer rows.Close()

	result := make(map[model.ItemType]int)
	
	// Initialize with zeros for all types
	result[model.ItemTypeEpic] = 0
	result[model.ItemTypeFeature] = 0
	result[model.ItemTypeStory] = 0

	for rows.Next() {
		var (
			itemType model.ItemType
			count    int
		)

		err := rows.Scan(&itemType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backlog size: %w", err)
		}

		result[itemType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetItemAge retrieves age metrics for backlog items
func (r *MetricsRepository) GetItemAge(ctx context.Context, status model.ItemStatus) (map[model.ItemType]float64, error) {
	query := `
		SELECT 
			type, 
			AVG(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)) / 86400) as avg_age_days
		FROM backlog_items
		WHERE status = $1
		GROUP BY type
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query item age: %w", err)
	}
	defer rows.Close()

	result := make(map[model.ItemType]float64)
	
	// Initialize with zeros for all types
	result[model.ItemTypeEpic] = 0
	result[model.ItemTypeFeature] = 0
	result[model.ItemTypeStory] = 0

	for rows.Next() {
		var (
			itemType  model.ItemType
			avgAgeDays float64
		)

		err := rows.Scan(&itemType, &avgAgeDays)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item age: %w", err)
		}

		result[itemType] = avgAgeDays
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetWIPCounts retrieves work-in-progress counts
func (r *MetricsRepository) GetWIPCounts(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*) as wip_count
		FROM backlog_items
		WHERE status = $1
	`

	var wipCount int
	err := r.db.QueryRowContext(ctx, query, model.ItemStatusInProgress).Scan(&wipCount)
	if err != nil {
		return 0, fmt.Errorf("failed to query WIP count: %w", err)
	}

	return wipCount, nil
}

// GetLeadTime retrieves lead time metrics
func (r *MetricsRepository) GetLeadTime(ctx context.Context, timeWindowDays int) (float64, error) {
	// Lead time is calculated as the average time from creation to completion
	// for items completed in the last timeWindowDays days
	query := `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400) as avg_lead_time_days
		FROM backlog_items
		WHERE 
			status = $1 AND 
			updated_at >= NOW() - INTERVAL '1 day' * $2
	`

	var avgLeadTime float64
	err := r.db.QueryRowContext(ctx, query, model.ItemStatusDone, timeWindowDays).Scan(&avgLeadTime)
	if err != nil {
		return 0, fmt.Errorf("failed to query lead time: %w", err)
	}

	return avgLeadTime, nil
}

// GetThroughput retrieves throughput metrics
func (r *MetricsRepository) GetThroughput(ctx context.Context, timeWindowDays int) (int, error) {
	// Throughput is the number of items completed in the last timeWindowDays days
	query := `
		SELECT COUNT(*) as throughput
		FROM backlog_items
		WHERE 
			status = $1 AND 
			updated_at >= NOW() - INTERVAL '1 day' * $2
	`

	var throughput int
	err := r.db.QueryRowContext(ctx, query, model.ItemStatusDone, timeWindowDays).Scan(&throughput)
	if err != nil {
		return 0, fmt.Errorf("failed to query throughput: %w", err)
	}

	return throughput, nil
}

// Additional metrics methods

// GetStatusTransitionTimes calculates the average time spent in each status
func (r *MetricsRepository) GetStatusTransitionTimes(ctx context.Context, timeWindowDays int) (map[model.ItemStatus]float64, error) {
	// This requires event sourcing data to track status changes
	// Here is a simplified version based on events table
	query := `
		SELECT 
			e.payload->>'previousStatus' as status,
			AVG(EXTRACT(EPOCH FROM (e.created_at - prev_e.created_at)) / 86400) as avg_days
		FROM 
			events e
		JOIN 
			events prev_e ON e.item_id = prev_e.item_id AND prev_e.id = (
				SELECT id FROM events 
				WHERE item_id = e.item_id AND created_at < e.created_at
				ORDER BY created_at DESC LIMIT 1
			)
		WHERE 
			e.event_type = 'ITEM_UPDATED' AND
			e.payload->>'previousStatus' IS NOT NULL AND
			e.created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY 
			e.payload->>'previousStatus'
	`

	rows, err := r.db.QueryContext(ctx, query, timeWindowDays)
	if err != nil {
		return nil, fmt.Errorf("failed to query status transition times: %w", err)
	}
	defer rows.Close()

	result := make(map[model.ItemStatus]float64)
	
	// Initialize with zeros for all statuses
	result[model.ItemStatusNew] = 0
	result[model.ItemStatusReady] = 0
	result[model.ItemStatusInProgress] = 0
	result[model.ItemStatusBlocked] = 0
	result[model.ItemStatusDone] = 0

	for rows.Next() {
		var (
			status  model.ItemStatus
			avgDays float64
		)

		err := rows.Scan(&status, &avgDays)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status transition times: %w", err)
		}

		result[status] = avgDays
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetBlockedItemsMetrics retrieves metrics about blocked items
func (r *MetricsRepository) GetBlockedItemsMetrics(ctx context.Context) (int, float64, error) {
	query := `
		SELECT 
			COUNT(*) as blocked_count,
			AVG(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - updated_at)) / 86400) as avg_blocked_days
		FROM backlog_items
		WHERE status = $1
	`

	var (
		blockedCount   int
		avgBlockedDays float64
	)

	err := r.db.QueryRowContext(ctx, query, model.ItemStatusBlocked).Scan(&blockedCount, &avgBlockedDays)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query blocked items metrics: %w", err)
	}

	return blockedCount, avgBlockedDays, nil
}

// GetAgeingItemsCount retrieves the count of items that have been in a non-Done status for too long
func (r *MetricsRepository) GetAgeingItemsCount(ctx context.Context, thresholdDays int) (int, error) {
	query := `
		SELECT COUNT(*) as ageing_count
		FROM backlog_items
		WHERE 
			status != $1 AND
			EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)) / 86400 > $2
	`

	var ageingCount int
	err := r.db.QueryRowContext(ctx, query, model.ItemStatusDone, thresholdDays).Scan(&ageingCount)
	if err != nil {
		return 0, fmt.Errorf("failed to query ageing items count: %w", err)
	}

	return ageingCount, nil
}

// GetStoryPointsProgress retrieves story points completion metrics
func (r *MetricsRepository) GetStoryPointsProgress(ctx context.Context, timeWindowDays int) (int, int, float64, error) {
	// Query for completed story points
	completedQuery := `
		SELECT COALESCE(SUM(story_points), 0) as completed_points
		FROM backlog_items
		WHERE 
			status = $1 AND
			updated_at >= NOW() - INTERVAL '1 day' * $2
	`

	var completedPoints int
	err := r.db.QueryRowContext(ctx, completedQuery, model.ItemStatusDone, timeWindowDays).Scan(&completedPoints)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to query completed story points: %w", err)
	}

	// Query for total story points (both completed and in-progress)
	totalQuery := `
		SELECT COALESCE(SUM(story_points), 0) as total_points
		FROM backlog_items
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	`

	var totalPoints int
	err = r.db.QueryRowContext(ctx, totalQuery, timeWindowDays).Scan(&totalPoints)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to query total story points: %w", err)
	}

	// Calculate completion percentage
	var completionPercentage float64
	if totalPoints > 0 {
		completionPercentage = float64(completedPoints) / float64(totalPoints) * 100
	}

	return completedPoints, totalPoints, completionPercentage, nil
}

// GetItemTypeDistribution calculates the distribution of item types
func (r *MetricsRepository) GetItemTypeDistribution(ctx context.Context) (map[model.ItemType]float64, error) {
	query := `
		WITH item_counts AS (
			SELECT type, COUNT(*) as count
			FROM backlog_items
			GROUP BY type
		),
		total AS (
			SELECT SUM(count) as total_count
			FROM item_counts
		)
		SELECT 
			ic.type, 
			(ic.count::float / t.total_count) * 100 as percentage
		FROM 
			item_counts ic
		CROSS JOIN 
			total t
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query item type distribution: %w", err)
	}
	defer rows.Close()

	result := make(map[model.ItemType]float64)
	
	// Initialize with zeros for all types
	result[model.ItemTypeEpic] = 0
	result[model.ItemTypeFeature] = 0
	result[model.ItemTypeStory] = 0

	for rows.Next() {
		var (
			itemType   model.ItemType
			percentage float64
		)

		err := rows.Scan(&itemType, &percentage)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item type distribution: %w", err)
		}

		result[itemType] = percentage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}
