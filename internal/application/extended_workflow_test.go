package application

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func fixture(t *testing.T) (*Service, context.Context, domain.OralHistoryCase, storage.MediaObject) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := New(store)
	collector := roleContext(RoleCollector, "采集员")
	c, err := svc.CreateCase(collector, CreateCaseCommand{Title: "批量测试", IntervieweeAlias: "受访者", Collector: "采集员", IdempotencyKey: "case"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.LockConsent(roleContext(RoleRepresentative, "代理人"), c.CaseID, LockConsentCommand{ExpectedVersion: 1, IdempotencyKey: "consent", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"档案"}, WithdrawalTerms: "书面撤回", ConfirmedBy: "代理人"}); err != nil {
		t.Fatal(err)
	}
	media, err := svc.UploadMedia(collector, UploadMediaCommand{CaseID: c.CaseID, ContentType: "audio/wav", Reader: bytes.NewBufferString("RIFF fixture")})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.GetCase(ctx, c.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	return svc, collector, c, media
}

func TestBatchImportIsAtomicAndIdempotent(t *testing.T) {
	svc, collector, c, media := fixture(t)
	cmd := ImportTranscriptBatchCommand{ExpectedVersion: c.Version, IdempotencyKey: "batch", Segments: []BatchSegmentCommand{
		{SegmentID: "s1", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 0, EndMillis: 1000, Transcript: "第一段", SensitivityTags: []string{" 地点", "地点"}},
		{SegmentID: "s2", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 1000, EndMillis: 2000, Transcript: "第二段"},
	}}
	result, err := svc.ImportTranscriptBatch(collector, c.CaseID, cmd)
	if err != nil || len(result.Segments) != 2 || result.Case.Version != 3 {
		t.Fatalf("批量导入失败: %v %#v", err, result)
	}
	replayed, err := svc.ImportTranscriptBatch(collector, c.CaseID, cmd)
	if err != nil || replayed.Case.Version != result.Case.Version || replayed.Segments[0].SegmentID != result.Segments[0].SegmentID {
		t.Fatalf("批量幂等回放失败: %v", err)
	}
	bad := cmd
	bad.ExpectedVersion = result.Case.Version
	bad.IdempotencyKey = "bad-batch"
	bad.Segments = []BatchSegmentCommand{{SegmentID: "s3", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 1500, EndMillis: 2500, Transcript: "重叠"}}
	_, err = svc.ImportTranscriptBatch(collector, c.CaseID, bad)
	rule, ok := err.(*domain.RuleError)
	if !ok || !strings.Contains(rule.Field, "segments[0]") {
		t.Fatalf("重叠错误应定位到批次索引: %v", err)
	}
	segments, _ := svc.ListSegments(context.Background(), c.CaseID)
	if len(segments) != 2 {
		t.Fatalf("失败批次不应留下部分片段: %#v", segments)
	}
}

func TestReturnedSegmentsResetAndCanBeReconfirmed(t *testing.T) {
	svc, collector, c, media := fixture(t)
	seg, err := svc.AddSegment(collector, c.CaseID, AddSegmentCommand{ExpectedVersion: c.Version, IdempotencyKey: "seg", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 0, EndMillis: 1000, Transcript: "含地点", SensitivityTags: []string{"地点"}})
	if err != nil {
		t.Fatal(err)
	}
	seg, err = svc.ReviewSegment(roleContext(RoleReviewer, "审查员"), c.CaseID, seg.SegmentID, ReviewCommand{ExpectedVersion: 3, IdempotencyKey: "review", DecisionStatus: domain.DecisionReplace, RedactionText: "含区域", Reason: "隐去地点", ReviewedBy: "审查员"})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.CompleteReview(roleContext(RoleReviewer, "审查员"), c.CaseID, 4)
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.ConfirmPreview(roleContext(RoleRepresentative, "代理人"), c.CaseID, ConfirmCommand{ExpectedVersion: c.Version, IdempotencyKey: "return", ReturnedSegmentIDs: []string{seg.SegmentID, seg.SegmentID}, Comment: "请重新裁定"})
	if err != nil || c.Status != domain.StatusTranscribing {
		t.Fatalf("退回失败: %v", err)
	}
	current, _ := svc.Store().GetSegment(context.Background(), seg.SegmentID)
	if current.DecisionStatus != domain.DecisionPending || current.RedactionText != "" || current.DecisionReason != "" || current.ReviewedBy != "" {
		t.Fatalf("退回未清除旧裁定: %#v", current)
	}
	if _, err = svc.CompleteReview(roleContext(RoleReviewer, "审查员"), c.CaseID, c.Version); err == nil {
		t.Fatal("退回片段未重新裁定不应完成审查")
	}
	_, err = svc.ReviewSegment(roleContext(RoleReviewer, "审查员"), c.CaseID, seg.SegmentID, ReviewCommand{ExpectedVersion: c.Version, IdempotencyKey: "review-again", DecisionStatus: domain.DecisionKeep, Reason: "复核后保留", ReviewedBy: "审查员"})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.CompleteReview(roleContext(RoleReviewer, "审查员"), c.CaseID, c.Version+1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ConfirmPreview(roleContext(RoleRepresentative, "代理人"), c.CaseID, ConfirmCommand{ExpectedVersion: c.Version, IdempotencyKey: "confirm-again", Confirmed: true, Comment: "确认"}); err != nil {
		t.Fatal(err)
	}
	history, err := svc.Store().ListConfirmations(context.Background(), c.CaseID)
	if err != nil || len(history) != 2 || !history[0].Confirmed || history[1].Confirmed {
		t.Fatalf("确认历史不完整: %v %#v", err, history)
	}
}

func TestTimecodeRevisionAndReleaseVerification(t *testing.T) {
	svc, collector, c, media := fixture(t)
	first, err := svc.AddSegment(collector, c.CaseID, AddSegmentCommand{ExpectedVersion: c.Version, IdempotencyKey: "first", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 0, EndMillis: 1000, Transcript: "第一段"})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.GetCase(context.Background(), c.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.AddSegment(collector, c.CaseID, AddSegmentCommand{ExpectedVersion: c.Version, IdempotencyKey: "second", MediaDigest: media.Digest, Speaker: "受访者", StartMillis: 1000, EndMillis: 2000, Transcript: "第二段"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReviseSegmentTimecode(collector, c.CaseID, second.SegmentID, ReviseTimecodeCommand{ExpectedVersion: c.Version + 1, IdempotencyKey: "overlap-time", MediaDigest: media.Digest, StartMillis: 500, EndMillis: 1500})
	if err == nil {
		t.Fatal("重叠时间码应被拒绝")
	}
	result, err := svc.ReviseSegmentTimecode(collector, c.CaseID, second.SegmentID, ReviseTimecodeCommand{ExpectedVersion: c.Version + 1, IdempotencyKey: "fix-time", MediaDigest: media.Digest, StartMillis: 1100, EndMillis: 2100})
	if err != nil || result.Segment.StartMillis != 1100 {
		t.Fatalf("时间码修订失败: %v %#v", err, result)
	}
	_, _ = first, result
	current, _ := svc.GetCase(context.Background(), c.CaseID)
	seg, _ := svc.Store().GetSegment(context.Background(), first.SegmentID)
	_, err = svc.ReviewSegment(roleContext(RoleReviewer, "审查员"), current.CaseID, seg.SegmentID, ReviewCommand{ExpectedVersion: current.Version, IdempotencyKey: "rv-1", DecisionStatus: domain.DecisionKeep, Reason: "无需脱敏", ReviewedBy: "审查员"})
	if err != nil {
		t.Fatal(err)
	}
	current, _ = svc.GetCase(context.Background(), current.CaseID)
	seg, _ = svc.Store().GetSegment(context.Background(), second.SegmentID)
	_, err = svc.ReviewSegment(roleContext(RoleReviewer, "审查员"), current.CaseID, seg.SegmentID, ReviewCommand{ExpectedVersion: current.Version, IdempotencyKey: "rv-2", DecisionStatus: domain.DecisionKeep, Reason: "无需脱敏", ReviewedBy: "审查员"})
	if err != nil {
		t.Fatal(err)
	}
	current, _ = svc.GetCase(context.Background(), current.CaseID)
	current, err = svc.CompleteReview(roleContext(RoleReviewer, "审查员"), current.CaseID, current.Version)
	if err != nil {
		t.Fatal(err)
	}
	current, err = svc.ConfirmPreview(roleContext(RoleRepresentative, "代理人"), current.CaseID, ConfirmCommand{ExpectedVersion: current.Version, IdempotencyKey: "ok-confirm", Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Release(roleContext(RoleArchivist, "发布员"), current.CaseID, ReleaseCommand{ExpectedVersion: current.Version, IdempotencyKey: "pkg", IssuedBy: "发布员"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := svc.VerifyReleasePackage(roleContext(RoleArchivist, "发布员"), current.CaseID)
	if err != nil || !report.Valid || len(report.Failures) != 0 {
		t.Fatalf("发布包核验失败: %v %#v", err, report)
	}
}
