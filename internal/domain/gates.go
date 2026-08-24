package domain

import (
	"sort"
	"time"
)

type GateIssue struct {
	Code      string `json:"code"`
	Field     string `json:"field"`
	SegmentID string `json:"segmentID,omitempty"`
	Message   string `json:"message"`
}

func CheckGates(c OralHistoryCase, consent *ConsentScope, segments []TranscriptSegment, confirmation *Confirmation, now time.Time) []GateIssue {
	issues := []GateIssue{}
	if consent == nil {
		issues = append(issues, GateIssue{Code: "MISSING_CONSENT", Field: "consent", Message: "尚未冻结知情同意范围"})
	} else if consent.EmbargoUntil != nil && now.Before(*consent.EmbargoUntil) {
		issues = append(issues, GateIssue{Code: "EMBARGO_ACTIVE", Field: "embargoUntil", Message: "资料仍处于封存期"})
	}
	if len(segments) == 0 {
		issues = append(issues, GateIssue{Code: "MISSING_TRANSCRIPT", Field: "segments", Message: "至少需要一个转录片段"})
	}
	sorted := append([]TranscriptSegment(nil), segments...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartMillis < sorted[j].StartMillis })
	for i, s := range sorted {
		if s.DecisionStatus == DecisionPending {
			issues = append(issues, GateIssue{Code: "PENDING_REDACTION", Field: "decisionStatus", SegmentID: s.SegmentID, Message: "敏感片段尚未裁定"})
		}
		if i > 0 && sorted[i-1].MediaDigest == s.MediaDigest && sorted[i-1].EndMillis > s.StartMillis {
			issues = append(issues, GateIssue{Code: "OVERLAPPING_SEGMENT", Field: "startMillis", SegmentID: s.SegmentID, Message: "同一音频的转录时间段重叠"})
		}
	}
	if confirmation == nil || !confirmation.Confirmed {
		issues = append(issues, GateIssue{Code: "MISSING_CONFIRMATION", Field: "confirmation", Message: "受访者尚未确认当前预览"})
	}
	return issues
}
