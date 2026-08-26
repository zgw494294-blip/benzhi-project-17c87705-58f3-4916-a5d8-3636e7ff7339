package review_version_conflict_pollution_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func principal(role application.Role, name string) context.Context {
	return application.WithPrincipal(context.Background(), application.Principal{Role: role, Name: name})
}

func TestStaleReviewDoesNotPersistSegmentDecision(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	svc := application.New(store)
	collector := principal(application.RoleCollector, "采集员")
	representative := principal(application.RoleRepresentative, "代理人")
	reviewer := principal(application.RoleReviewer, "审查员")

	c, err := svc.CreateCase(collector, application.CreateCaseCommand{
		Title: "并发裁定测试", IntervieweeAlias: "受访者", Collector: "采集员", IdempotencyKey: "create-review-conflict",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.LockConsent(representative, c.CaseID, application.LockConsentCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "lock-review-conflict",
		AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"数字档案"},
		WithdrawalTerms: "发布前书面撤回", ConfirmedBy: "代理人",
	})
	if err != nil {
		t.Fatal(err)
	}
	media, err := svc.UploadMedia(collector, application.UploadMediaCommand{
		CaseID: c.CaseID, ContentType: "audio/wav", Reader: bytes.NewBufferString("RIFF review conflict"),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.AddSegment(collector, c.CaseID, application.AddSegmentCommand{
		ExpectedVersion: 2, IdempotencyKey: "segment-first", MediaDigest: media.Digest,
		Speaker: "受访者", StartMillis: 0, EndMillis: 1000, Transcript: "第一处敏感地点", SensitivityTags: []string{"地点"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.AddSegment(collector, c.CaseID, application.AddSegmentCommand{
		ExpectedVersion: 3, IdempotencyKey: "segment-second", MediaDigest: media.Digest,
		Speaker: "受访者", StartMillis: 1000, EndMillis: 2000, Transcript: "第二处敏感地点", SensitivityTags: []string{"地点"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReviewSegment(reviewer, c.CaseID, first.SegmentID, application.ReviewCommand{
		ExpectedVersion: 4, IdempotencyKey: "review-first", DecisionStatus: domain.DecisionReplace,
		RedactionText: "第一处区域", Reason: "隐去地点", ReviewedBy: "审查员甲",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ReviewSegment(reviewer, c.CaseID, second.SegmentID, application.ReviewCommand{
		ExpectedVersion: 4, IdempotencyKey: "review-stale", DecisionStatus: domain.DecisionRemove,
		Reason: "陈旧版本裁定", ReviewedBy: "审查员乙",
	})
	var conflict *domain.RuleError
	if !errors.As(err, &conflict) || conflict.Code != domain.CodeConflict {
		t.Fatalf("陈旧裁定应返回版本冲突，得到 %v", err)
	}

	persisted, err := store.GetSegment(ctx, second.SegmentID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.DecisionStatus != domain.DecisionPending || persisted.DecisionReason != "" || persisted.ReviewedBy != "" {
		t.Fatalf("版本冲突的失败请求污染了片段裁定: %#v", persisted)
	}
}
