package createcaseidempotency

import (
	"context"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/storage"
)

func TestCreateCaseReplayDoesNotCreateAnotherCase(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := application.New(store)
	collector := application.WithPrincipal(ctx, application.Principal{Name: "采集员", Role: application.RoleCollector})
	cmd := application.CreateCaseCommand{Title: "幂等建档", IntervieweeAlias: "甲", Collector: "采集员", IdempotencyKey: "case-once"}
	want, err := svc.CreateCase(collector, cmd)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.CreateCase(collector, cmd)
	if err != nil || got.CaseID != want.CaseID {
		t.Fatalf("相同 idempotencyKey 重试应返回首次案卷: %v %#v", err, got)
	}
	cases, err := svc.ListCases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 {
		t.Fatalf("幂等重试不应新增案卷，实际 %d 个", len(cases))
	}
}
