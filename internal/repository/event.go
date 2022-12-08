package repository

import (
	"sync"

	"github.com/porter-dev/porter-agent/internal/logger"
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

func (r *EventRepository) DeleteOlderEvents(l *logger.Logger) {
	l.Info().Caller().Msgf("cleaning up old events")

	var distinctReleases []ReleaseInfo

	if err := r.db.Model(&models.Event{}).Distinct().Find(&distinctReleases).Error; err != nil {
		l.Error().Caller().Msgf("error fetching distinct release name, namespace pairs: %w", err)
		return
	}

	var wg sync.WaitGroup

	for _, info := range distinctReleases {
		wg.Add(1)

		go func(info ReleaseInfo) {
			defer wg.Done()

			if err := r.db.Exec(`DELETE FROM events WHERE (release_name = ? AND release_namespace = ?) AND id NOT IN (SELECT id FROM events e2 WHERE (e2.release_name = ? AND e2.release_namespace = ?) ORDER BY e2.timestamp desc LIMIT 100)`, info.ReleaseName, info.ReleaseNamespace, info.ReleaseName, info.ReleaseNamespace).Error; err != nil {
				l.Error().Caller().Msgf("error deleting older events for release %s, namespace %s: %v",
					info.ReleaseName, info.ReleaseNamespace, err)
			}
		}(info)
	}

	wg.Wait()

	l.Info().Caller().Msgf("finished cleaning up old events")
}
