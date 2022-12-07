package repository

import (
	"fmt"
	"testing"
	"time"

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

	// delete older events
	err := tester.repo.Event.DeleteOlderEvents()

	if err != nil {
		t.Fatalf("Expected no error after deleting older events, got %v", err)
	}

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
	}
}
