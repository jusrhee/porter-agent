package repository

import (
	"fmt"
	"log"
	"sync"

	"github.com/porter-dev/porter-agent/internal/models"
	"github.com/porter-dev/porter-agent/internal/utils"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

// NewEventRepository returns pointer to repo along with the db
func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db}
}

func (r *EventRepository) CreateEvent(event *models.Event) (*models.Event, error) {
	if err := r.db.Create(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func (r *EventRepository) ReadEvent(id uint) (*models.Event, error) {
	event := &models.Event{}

	if err := r.db.Where("id = ?", id).First(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func (r *EventRepository) UpdateEvent(event *models.Event) (*models.Event, error) {
	if err := r.db.Save(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func (r *EventRepository) ListEvents(
	filter *utils.ListEventsFilter,
	opts ...utils.QueryOption,
) ([]*models.Event, *utils.PaginatedResult, error) {
	var events []*models.Event

	db := r.db.Model(&models.Event{})

	if filter.Type != nil {
		db = db.Where("type = ?", *filter.Type)
	}

	if filter.ReleaseName != nil {
		db = db.Where("release_name = ?", *filter.ReleaseName)
	}

	if filter.ReleaseNamespace != nil {
		db = db.Where("release_namespace = ?", *filter.ReleaseNamespace)
	}

	if filter.AdditionalQueryMeta != nil {
		db = db.Where("additional_query_meta = ?", *filter.AdditionalQueryMeta)
	}

	paginatedResult := &utils.PaginatedResult{}

	db = db.Scopes(utils.Paginate(opts, db, paginatedResult))

	if err := db.Find(&events).Error; err != nil {
		return nil, nil, err
	}

	return events, paginatedResult, nil
}

func (r *EventRepository) DeleteEvent(uid string) error {
	incident := &models.Event{}

	if err := r.db.Where("unique_id = ?", uid).First(incident).Error; err != nil {
		return err
	}

	if err := r.db.Delete(incident).Error; err != nil {
		return err
	}

	return nil
}

type ReleaseInfo struct {
	ReleaseName      string
	ReleaseNamespace string
}

func (r *EventRepository) DeleteOlderEvents() error {
	var distinctReleases []ReleaseInfo

	if err := r.db.Model(&models.Event{}).Distinct().Find(&distinctReleases).Error; err != nil {
		return fmt.Errorf("error fetching distinct release name, namespace pairs: %w", err)
	}

	var wg sync.WaitGroup
	maxEvents := 100

	for _, info := range distinctReleases {
		wg.Add(1)

		go func(info ReleaseInfo) {
			defer wg.Done()

			var count int64

			if err := r.db.Model(&models.Event{}).
				Where("release_name = ? AND release_namespace = ?", info.ReleaseName, info.ReleaseNamespace).
				Count(&count).Error; err != nil {
				log.Printf("error counting events for release %s, namespace %s: %v",
					info.ReleaseName, info.ReleaseNamespace, err)
				return
			}

			if count <= int64(maxEvents) {
				return
			}

			log.Printf("deleting older events for release %s, namespace %s", info.ReleaseName, info.ReleaseNamespace)

			for i := 1; i < (int(count)/maxEvents)+1; i++ {
				var events []*models.Event

				if err := r.db.Model(&models.Event{}).
					Where("release_name = ? AND release_namespace = ?", info.ReleaseName, info.ReleaseNamespace).
					Order("timestamp DESC").
					Limit(maxEvents).
					Offset(i * maxEvents).
					Find(&events).
					Error; err != nil {
					log.Printf("error fetching events for release %s, namespace %s: %v",
						info.ReleaseName, info.ReleaseNamespace, err)
					continue
				}

				for _, ev := range events {
					// use GORM's Unscoped().Delete() to permanently delete the row from the DB
					if err := r.db.Unscoped().Delete(ev).Error; err != nil {
						log.Printf("error deleting event %d for release %s, namespace %s: %v",
							ev.ID, info.ReleaseName, info.ReleaseNamespace, err)
					}
				}
			}
		}(info)
	}

	wg.Wait()

	return nil
}
