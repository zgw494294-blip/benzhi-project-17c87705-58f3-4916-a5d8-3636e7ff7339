package media_cross_case_state_pollution

import (
	"bytes"
	"context"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/storage"
)

func collectorContext(name string) context.Context {
	return application.WithPrincipal(context.Background(), application.Principal{Name: name, Role: application.RoleCollector})
}

func createLockedCase(t *testing.T, svc *application.Service, suffix string) string {
	t.Helper()
	ctx := collectorContext("采集员-" + suffix)
	c, err := svc.CreateCase(ctx, application.CreateCaseCommand{
		Title: "跨案卷音频测试-" + suffix, IntervieweeAlias: "受访者-" + suffix,
		Collector: "采集员-" + suffix, IdempotencyKey: "case-" + suffix,
	})
	if err != nil {
		t.Fatalf("创建案卷失败: %v", err)
	}
	_, err = svc.LockConsent(application.WithPrincipal(context.Background(), application.Principal{
		Name: "代理人-" + suffix, Role: application.RoleRepresentative,
	}), c.CaseID, application.LockConsentCommand{
		ExpectedVersion: 1, IdempotencyKey: "consent-" + suffix,
		AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"数字档案"},
		WithdrawalTerms: "发布前书面撤回", ConfirmedBy: "代理人-" + suffix,
	})
	if err != nil {
		t.Fatalf("冻结同意失败: %v", err)
	}
	return c.CaseID
}

func TestUploadingSameContentToAnotherCaseDoesNotRemoveFirstMediaReference(t *testing.T) {
	store, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := application.New(store)
	firstCase := createLockedCase(t, svc, "一")
	secondCase := createLockedCase(t, svc, "二")
	content := []byte("RIFF shared oral-history audio")

	first, err := svc.UploadMedia(collectorContext("采集员-一"), application.UploadMediaCommand{
		CaseID: firstCase, ContentType: "audio/wav", Reader: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("首次上传失败: %v", err)
	}
	second, err := svc.UploadMedia(collectorContext("采集员-二"), application.UploadMediaCommand{
		CaseID: secondCase, ContentType: "audio/wav", Reader: bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("第二案卷上传相同内容失败: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("相同内容应保持同一摘要: %q != %q", first.Digest, second.Digest)
	}

	firstMedia, err := store.ListMedia(context.Background(), firstCase)
	if err != nil {
		t.Fatal(err)
	}
	secondMedia, err := store.ListMedia(context.Background(), secondCase)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstMedia) != 1 || len(secondMedia) != 1 {
		t.Fatalf("跨案卷复用音频不应污染引用: 首案卷=%d，次案卷=%d", len(firstMedia), len(secondMedia))
	}
}
