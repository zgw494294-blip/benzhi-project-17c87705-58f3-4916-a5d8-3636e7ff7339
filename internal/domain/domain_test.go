package domain

import (
	"testing"
	"time"
)

func TestConsentDigestIsStable(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	a, err := NewConsent("c1", "case1", 1, []string{"公众", "研究人员", "公众"}, []string{"研究", "档案"}, nil, "书面撤回", "代理人", now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewConsent("c2", "case1", 1, []string{"研究人员", "公众"}, []string{"档案", "研究"}, nil, "书面撤回", "代理人", now)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest {
		t.Fatalf("规范化集合应产生稳定摘要: %s != %s", a.Digest, b.Digest)
	}
}

func TestCaseRejectsSkippedTransition(t *testing.T) {
	c, err := NewCase("case1", "主题", "代称", "采集员", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Transition(StatusApproved, time.Now()); err == nil {
		t.Fatal("草稿不应跳过流程直接批准")
	}
	if c.Status != StatusDraft || c.Version != 1 {
		t.Fatal("失败的迁移不应修改聚合")
	}
}

func TestGateReportLocatesSegments(t *testing.T) {
	now := time.Now().UTC()
	c, _ := NewCase("case1", "主题", "代称", "采集员", now)
	s1, _ := NewSegment("s1", "case1", "sha256:a", "受访者", 0, 2000, "第一段", []string{"姓名"})
	s2, _ := NewSegment("s2", "case1", "sha256:a", "受访者", 1500, 3000, "第二段", nil)
	report := BuildGateReport(*c, nil, []TranscriptSegment{*s1, *s2}, nil, now)
	if report.Ready {
		t.Fatal("缺失同意、裁定和确认时不应通过门禁")
	}
	foundTarget := false
	for _, action := range report.Remediation {
		if action.TargetID == "s1" && action.OwnerRole == "reviewer" {
			foundTarget = true
		}
	}
	if !foundTarget {
		t.Fatal("门禁报告应把未裁定片段定向给审查员")
	}
}
