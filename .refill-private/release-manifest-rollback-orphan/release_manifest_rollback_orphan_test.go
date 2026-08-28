package release_manifest_rollback_orphan

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func actor(role application.Role, name string) context.Context {
	return application.WithPrincipal(context.Background(), application.Principal{Name: name, Role: role})
}

func TestFailedReleaseDoesNotLeaveManifestFileAfterTransactionRollback(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := storage.Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := application.New(store)

	c, err := svc.CreateCase(actor(application.RoleCollector, "采集员"), application.CreateCaseCommand{
		Title: "发布回滚测试", IntervieweeAlias: "受访者", Collector: "采集员", IdempotencyKey: "case",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.LockConsent(actor(application.RoleRepresentative, "代理人"), c.CaseID, application.LockConsentCommand{
		ExpectedVersion: 1, IdempotencyKey: "consent", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"档案"}, WithdrawalTerms: "书面撤回", ConfirmedBy: "代理人",
	}); err != nil {
		t.Fatal(err)
	}
	media, err := svc.UploadMedia(actor(application.RoleCollector, "采集员"), application.UploadMediaCommand{
		CaseID: c.CaseID, ContentType: "audio/wav", Reader: bytes.NewBufferString("RIFF rollback manifest"),
	})
	if err != nil {
		t.Fatal(err)
	}
	seg, err := svc.AddSegment(actor(application.RoleCollector, "采集员"), c.CaseID, application.AddSegmentCommand{
		ExpectedVersion: 2, IdempotencyKey: "segment", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 0, EndMillis: 1000, Transcript: "可发布内容",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ReviewSegment(actor(application.RoleReviewer, "审查员"), c.CaseID, seg.SegmentID, application.ReviewCommand{
		ExpectedVersion: 3, IdempotencyKey: "review", DecisionStatus: domain.DecisionKeep, Reason: "无需脱敏", ReviewedBy: "审查员",
	}); err != nil {
		t.Fatal(err)
	}
	c, err = svc.GetCase(ctx, c.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.CompleteReview(actor(application.RoleReviewer, "审查员"), c.CaseID, c.Version)
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.ConfirmPreview(actor(application.RoleRepresentative, "代理人"), c.CaseID, application.ConfirmCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "confirm", Confirmed: true, Comment: "确认",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err = store.DB().ExecContext(ctx, "DROP TABLE audit_events"); err != nil {
		t.Fatal(err)
	}
	pkg, err := svc.Release(actor(application.RoleArchivist, "发布员"), c.CaseID, application.ReleaseCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "release", IssuedBy: "发布员",
	})
	if err == nil {
		t.Fatal("审计存储失效时发布必须失败")
	}
	if _, lookupErr := store.GetPackageByCase(ctx, c.CaseID); lookupErr != storage.ErrNotFound {
		t.Fatalf("失败事务不应提交发布包: %v", lookupErr)
	}
	manifestPath := filepath.Join(root, "manifests", strings.TrimPrefix(pkg.ManifestDigest, "sha256:")+".json")
	if _, statErr := os.Stat(manifestPath); statErr == nil {
		t.Fatalf("事务回滚后不应残留清单文件: %s", manifestPath)
	} else if !os.IsNotExist(statErr) {
		t.Fatal(statErr)
	}
}
