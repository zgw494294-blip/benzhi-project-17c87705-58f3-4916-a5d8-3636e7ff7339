package resource_save_state_pollution_test

import (
	"bytes"
	"context"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func TestRejectedDraftUploadDoesNotPersistMediaResource(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store)
	collector := application.WithPrincipal(ctx, application.Principal{Name: "采集员", Role: application.RoleCollector})
	caseRecord, err := service.CreateCase(collector, application.CreateCaseCommand{
		Title:            "草稿阶段资源隔离测试",
		IntervieweeAlias: "受访者甲",
		Collector:        "采集员",
		IdempotencyKey:   "draft-upload-case",
	})
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("RIFF rejected draft upload")
	_, err = service.UploadMedia(collector, application.UploadMediaCommand{
		CaseID:      caseRecord.CaseID,
		ContentType: "audio/wav",
		Reader:      bytes.NewReader(payload),
	})
	rule, ok := err.(*domain.RuleError)
	if !ok || rule.Code != domain.CodeInvalidState {
		t.Fatalf("草稿阶段上传应被拒绝，得到 %v", err)
	}

	media, err := store.ListMedia(ctx, caseRecord.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	if len(media) != 0 {
		t.Fatalf("TestRejectedDraftUploadDoesNotPersistMediaResource: 被拒绝的上传仍持久化了 %d 条媒体引用", len(media))
	}
	exists, err := store.MediaFileExists(ctx, domain.BytesDigest(payload))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("TestRejectedDraftUploadDoesNotPersistMediaResource: 被拒绝的上传仍留下内容寻址文件")
	}
}
