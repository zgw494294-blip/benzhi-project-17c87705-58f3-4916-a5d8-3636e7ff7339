package preview_storage_error_propagation

import (
	"context"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/storage"
)

func TestPreviewPropagatesConsentStorageFailure(t *testing.T) {
	ctx := application.WithPrincipal(context.Background(), application.Principal{Name: "采集员", Role: application.RoleCollector})
	root := t.TempDir()
	store, err := storage.Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.New(store)
	caseRecord, err := service.CreateCase(ctx, application.CreateCaseCommand{
		Title:            "存储错误传播测试",
		IntervieweeAlias: "受访者甲",
		Collector:        "采集员",
		IdempotencyKey:   "preview-storage-error-case",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB().ExecContext(ctx, "DROP TABLE consents"); err != nil {
		t.Fatal(err)
	}

	_, err = service.Preview(ctx, caseRecord.CaseID)
	if err == nil {
		t.Fatal("Preview returned a normal preview after the consent store became unavailable")
	}
}
