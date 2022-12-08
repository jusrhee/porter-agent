package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/porter-dev/porter-agent/internal/logger"
	"github.com/porter-dev/porter-agent/internal/models"
	"github.com/porter-dev/porter-agent/internal/utils"
)

func TestDeleteOlderEvents(t *testing.T) {
	tester := &tester{
		dbFileName: "./event_test.db",
	}

	setupTestEnv(tester, t)
	defer cleanup(tester, t)

	// create events for 10 distinct release name, namespace pairs
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("release-%d", i+1)
		namespace := fmt.Sprintf("namespace-%d", i+1)

		for j := 0; j < 200; j++ {
			timeNow := time.Now()

			_, err := tester.repo.Event.CreateEvent(&models.Event{
				ReleaseName:      name,
				ReleaseNamespace: namespace,
				UniqueID:         fmt.Sprintf("unique-id-%d-%d", i+1, j+1),
				Timestamp:        &timeNow,
			})

			if err != nil {
				t.Fatalf("Expected no error after creating event, got %v", err)
			}
		}
	}

	l := logger.NewConsole(false)

	// delete older events
	tester.repo.Event.DeleteOlderEvents(l)

	// check that there are 100 events for each release name, namespace pair
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("release-%d", i+1)
		namespace := fmt.Sprintf("namespace-%d", i+1)

		events, _, err := tester.repo.Event.ListEvents(&utils.ListEventsFilter{
			ReleaseName:      &name,
			ReleaseNamespace: &namespace,
		}, utils.WithLimit(200)) // set a limit of 200 to check that there are no more than 100 events

		if err != nil {
			t.Fatalf("Expected no error after listing events for release %s, namespace %s, got %v", name, namespace, err)
		}

		if len(events) != 100 {
			t.Fatalf("Expected 100 events for release %s, namespace %s, got %d", name, namespace, len(events))
		}

		// check that the most recent 100 events are the ones that still exist
		for j := 0; j < 100; j++ {
			if events[j].UniqueID != fmt.Sprintf("unique-id-%d-%d", i+1, j+101) {
				t.Fatalf("Expected event with unique id unique-id-%d-%d to be the %dth event for release %s, namespace %s, got %s",
					i+1, j+101, j+1, name, namespace, events[j].UniqueID)
			}
		}
	}
}
