package repository

import (
	"time"

	"github.com/porter-dev/porter-agent/internal/logger"
	"github.com/porter-dev/porter-agent/internal/models"
	"gorm.io/gorm"
)

type EventCacheRepository struct {
	db *gorm.DB
}

// NewEventCacheRepository returns pointer to repo along with the db
func NewEventCacheRepository(db *gorm.DB) *EventCacheRepository {
	return &EventCacheRepository{db}
}

func (r *EventCacheRepository) CreateEventCache(cache *models.EventCache) (*models.EventCache, error) {
	if err := r.db.Create(cache).Error; err != nil {
		return nil, err
	}

	return cache, nil
}

func (r *EventCacheRepository) ListEventCachesForEvent(uid string) ([]*models.EventCache, error) {
	var caches []*models.EventCache

	if err := r.db.Where("event_uid = ? AND timestamp >= ?", uid, time.Now().Add(-time.Hour)).
		Order("timestamp desc").
		Find(&caches).Error; err != nil {
		return nil, err
	}

	return caches, nil
}

func (r *EventCacheRepository) ListEventCachesForPod(name, namespace string) ([]*models.EventCache, error) {
	var caches []*models.EventCache

	if err := r.db.Where("pod_name = ? AND pod_namespace = ? AND timestamp >= ?", name, namespace, time.Now().Add(-time.Hour)).
		Order("timestamp desc").
		Find(&caches).Error; err != nil {
		return nil, err
	}

	return caches, nil
}

func (r *EventCacheRepository) DeleteOlderEventCaches(l *logger.Logger) {
	l.Info().Caller().Msgf("cleaning up old event caches")

	var olderCache []*models.EventCache

	if err := r.db.Model(&models.EventCache{}).Where("timestamp <= ?", time.Now().Add(-time.Hour)).Find(&olderCache).Error; err == nil {
		numDeleted := 0

		for _, cache := range olderCache {
			// use GORM's Unscoped().Delete() to permanently delete the row from the DB
			if err := r.db.Unscoped().Delete(cache).Error; err != nil {
				l.Error().Caller().Msgf("error deleting old event cache with ID: %d. Error: %v", cache.ID, err)
				numDeleted++
			}
		}

		l.Info().Caller().Msgf("deleted %d event cache objects from database", numDeleted)
	} else {
		l.Error().Caller().Msgf("error querying for older event cache DB entries: %v", err)
	}
}
