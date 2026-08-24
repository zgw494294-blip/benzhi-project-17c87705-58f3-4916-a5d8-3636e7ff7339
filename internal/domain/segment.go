package domain

import (
	"strings"
)

type DecisionStatus string

const (
	DecisionPending DecisionStatus = "PENDING"
	DecisionKeep    DecisionStatus = "KEEP"
	DecisionReplace DecisionStatus = "REPLACE"
	DecisionRemove  DecisionStatus = "REMOVE"
)

type TranscriptSegment struct {
	SegmentID       string         `json:"segmentID"`
	CaseID          string         `json:"caseID"`
	MediaDigest     string         `json:"mediaDigest"`
	Speaker         string         `json:"speaker"`
	StartMillis     int64          `json:"startMillis"`
	EndMillis       int64          `json:"endMillis"`
	Transcript      string         `json:"transcript"`
	SensitivityTags []string       `json:"sensitivityTags"`
	RedactionText   string         `json:"redactionText"`
	DecisionStatus  DecisionStatus `json:"decisionStatus"`
	DecisionReason  string         `json:"decisionReason"`
	ReviewedBy      string         `json:"reviewedBy"`
}

func NewSegment(id, caseID, mediaDigest, speaker string, start, end int64, transcript string, tags []string) (*TranscriptSegment, error) {
	if strings.TrimSpace(id) == "" {
		return nil, NewRuleError(CodeValidation, "segmentID", "片段编号不能为空")
	}
	if strings.TrimSpace(mediaDigest) == "" {
		return nil, NewRuleError(CodeValidation, "mediaDigest", "音频摘要不能为空")
	}
	if strings.TrimSpace(speaker) == "" {
		return nil, NewRuleError(CodeValidation, "speaker", "说话人不能为空")
	}
	if start < 0 || end <= start {
		return nil, NewRuleError(CodeValidation, "endMillis", "片段结束时间必须晚于开始时间")
	}
	if strings.TrimSpace(transcript) == "" {
		return nil, NewRuleError(CodeValidation, "transcript", "转录文本不能为空")
	}
	tags = normalizedSet(tags)
	decision := DecisionKeep
	if len(tags) > 0 {
		decision = DecisionPending
	}
	return &TranscriptSegment{SegmentID: id, CaseID: caseID, MediaDigest: mediaDigest, Speaker: strings.TrimSpace(speaker), StartMillis: start, EndMillis: end, Transcript: strings.TrimSpace(transcript), SensitivityTags: tags, DecisionStatus: decision}, nil
}

func (s *TranscriptSegment) Decide(status DecisionStatus, replacement, reason, reviewer string) error {
	if len(s.SensitivityTags) == 0 && status != DecisionKeep {
		return NewRuleError(CodeValidation, "decisionStatus", "无敏感标记片段只能保留")
	}
	if status != DecisionKeep && status != DecisionReplace && status != DecisionRemove {
		return NewRuleError(CodeValidation, "decisionStatus", "脱敏裁定无效")
	}
	if status == DecisionReplace && strings.TrimSpace(replacement) == "" {
		return NewRuleError(CodeValidation, "redactionText", "替换裁定必须提供替换文本")
	}
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(reviewer) == "" {
		return NewRuleError(CodeValidation, "decisionReason", "必须记录裁定理由和审查人")
	}
	s.DecisionStatus = status
	s.RedactionText = strings.TrimSpace(replacement)
	s.DecisionReason = strings.TrimSpace(reason)
	s.ReviewedBy = strings.TrimSpace(reviewer)
	return nil
}

func (s TranscriptSegment) PublicText() string {
	switch s.DecisionStatus {
	case DecisionRemove:
		return ""
	case DecisionReplace:
		return s.RedactionText
	default:
		return s.Transcript
	}
}
func (s TranscriptSegment) Digest() string {
	return StableDigest(struct {
		ID, Media, Speaker string
		Start, End         int64
		Text               string
		Tags               []string
		Decision           DecisionStatus
	}{s.SegmentID, s.MediaDigest, s.Speaker, s.StartMillis, s.EndMillis, s.PublicText(), s.SensitivityTags, s.DecisionStatus})
}
