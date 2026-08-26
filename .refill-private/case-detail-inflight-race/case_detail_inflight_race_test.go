package case_detail_inflight_race_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/storage"
)

func TestConcurrentCaseDetailSynchronizesInflightState(t *testing.T) {
	store, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	svc := application.New(store)
	collector := application.WithPrincipal(context.Background(), application.Principal{
		Name: "并发采集员",
		Role: application.RoleCollector,
	})
	c, err := svc.CreateCase(collector, application.CreateCaseCommand{
		Title:            "并发详情核验",
		IntervieweeAlias: "受访者甲",
		Collector:        "并发采集员",
		IdempotencyKey:   "case-detail-race-fixture",
	})
	if err != nil {
		t.Fatal(err)
	}

	const workers = 12
	start := make(chan struct{})
	results := make(chan error, workers)
	var ready sync.WaitGroup
	var calls sync.WaitGroup
	ready.Add(workers)
	calls.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer calls.Done()
			ready.Done()
			<-start
			detail, detailErr := svc.CaseDetail(context.Background(), c.CaseID)
			if detailErr != nil {
				results <- detailErr
				return
			}
			if detail.Case.CaseID != c.CaseID {
				results <- fmt.Errorf("详情串案: 期望 %s，得到 %s", c.CaseID, detail.Case.CaseID)
				return
			}
			results <- nil
		}()
	}
	ready.Wait()
	close(start)
	calls.Wait()
	close(results)
	for callErr := range results {
		if callErr != nil {
			t.Fatal(callErr)
		}
	}
}
