package application

import (
	"bytes"
	"context"
	"testing"

	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func roleContext(role Role, name string) context.Context {
	return WithPrincipal(context.Background(), Principal{Name: name, Role: role})
}

func TestControlledReleaseWorkflow(t *testing.T) {
	store, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store)
	c, err := svc.CreateCase(roleContext(RoleCollector, "采集员"), CreateCaseCommand{Title: "厂史访谈", IntervieweeAlias: "受访者甲", Collector: "采集员", IdempotencyKey: "case-key"})
	if err != nil {
		t.Fatal(err)
	}
	consent, err := svc.LockConsent(roleContext(RoleRepresentative, "代理人"), c.CaseID, LockConsentCommand{ExpectedVersion: 1, IdempotencyKey: "consent-key", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"数字档案"}, WithdrawalTerms: "发布前书面撤回", ConfirmedBy: "代理人"})
	if err != nil || consent.Digest == "" {
		t.Fatalf("冻结同意失败: %v", err)
	}
	media, err := svc.UploadMedia(roleContext(RoleCollector, "采集员"), UploadMediaCommand{CaseID: c.CaseID, ContentType: "audio/wav", Reader: bytes.NewBufferString("RIFF test audio")})
	if err != nil {
		t.Fatal(err)
	}
	seg, err := svc.AddSegment(roleContext(RoleCollector, "采集员"), c.CaseID, AddSegmentCommand{ExpectedVersion: 2, IdempotencyKey: "segment-key", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 0, EndMillis: 3000, Transcript: "我在城南工厂工作。", SensitivityTags: []string{"地点"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReviewSegment(roleContext(RoleReviewer, "审查员"), c.CaseID, seg.SegmentID, ReviewCommand{ExpectedVersion: 3, IdempotencyKey: "review-key", DecisionStatus: domain.DecisionReplace, RedactionText: "我在本地工厂工作。", Reason: "隐去地点", ReviewedBy: "审查员"})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.CompleteReview(roleContext(RoleReviewer, "审查员"), c.CaseID, 4)
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.ConfirmPreview(roleContext(RoleRepresentative, "代理人"), c.CaseID, ConfirmCommand{ExpectedVersion: c.Version, IdempotencyKey: "confirm-key", Confirmed: true, Comment: "确认"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := ReleaseCommand{ExpectedVersion: c.Version, IdempotencyKey: "release-key", IssuedBy: "发布员"}
	pkg, err := svc.Release(roleContext(RoleArchivist, "发布员"), c.CaseID, cmd)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := svc.Release(roleContext(RoleArchivist, "发布员"), c.CaseID, cmd)
	if err != nil {
		t.Fatalf("幂等重试失败: %v", err)
	}
	if pkg.PackageID != repeated.PackageID {
		t.Fatal("相同幂等键必须返回同一发布包")
	}
	detail, err := svc.CaseDetail(context.Background(), c.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Case.Status != domain.StatusReleased || detail.Manifest == nil || detail.Integrity.MissingReferences != 0 {
		t.Fatal("发布详情或内容引用不完整")
	}
}

func TestVersionConflictIsReported(t *testing.T) {
	store, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store)
	c, err := svc.CreateCase(roleContext(RoleCollector, "采集员"), CreateCaseCommand{Title: "测试", IntervieweeAlias: "甲", Collector: "采集员", IdempotencyKey: "new"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.LockConsent(roleContext(RoleRepresentative, "代理人"), c.CaseID, LockConsentCommand{ExpectedVersion: 99, IdempotencyKey: "bad-version", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"档案"}, WithdrawalTerms: "书面撤回", ConfirmedBy: "代理人"})
	rule, ok := err.(*domain.RuleError)
	if !ok || rule.Code != domain.CodeConflict {
		t.Fatalf("应返回版本冲突，得到 %v", err)
	}
}
