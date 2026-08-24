package application

import (
	"io"
	"oralarchive/internal/domain"
	"time"
)

type CreateCaseCommand struct {
	Title            string `json:"title"`
	IntervieweeAlias string `json:"intervieweeAlias"`
	Collector        string `json:"collector"`
	IdempotencyKey   string `json:"idempotencyKey"`
}
type LockConsentCommand struct {
	ExpectedVersion  int64      `json:"expectedVersion"`
	IdempotencyKey   string     `json:"idempotencyKey"`
	AllowedAudiences []string   `json:"allowedAudiences"`
	AllowedPurposes  []string   `json:"allowedPurposes"`
	EmbargoUntil     *time.Time `json:"embargoUntil"`
	WithdrawalTerms  string     `json:"withdrawalTerms"`
	ConfirmedBy      string     `json:"confirmedBy"`
}
type UploadMediaCommand struct {
	CaseID      string
	ContentType string
	Reader      io.Reader
}
type AddSegmentCommand struct {
	ExpectedVersion int64    `json:"expectedVersion"`
	IdempotencyKey  string   `json:"idempotencyKey"`
	MediaDigest     string   `json:"mediaDigest"`
	Speaker         string   `json:"speaker"`
	StartMillis     int64    `json:"startMillis"`
	EndMillis       int64    `json:"endMillis"`
	Transcript      string   `json:"transcript"`
	SensitivityTags []string `json:"sensitivityTags"`
}
type BatchSegmentCommand struct {
	SegmentID       string   `json:"segmentID,omitempty"`
	MediaDigest     string   `json:"mediaDigest"`
	Speaker         string   `json:"speaker"`
	StartMillis     int64    `json:"startMillis"`
	EndMillis       int64    `json:"endMillis"`
	Transcript      string   `json:"transcript"`
	SensitivityTags []string `json:"sensitivityTags"`
}
type ImportTranscriptBatchCommand struct {
	ExpectedVersion int64                 `json:"expectedVersion"`
	IdempotencyKey  string                `json:"idempotencyKey"`
	Segments        []BatchSegmentCommand `json:"segments"`
}
type BatchTranscriptResult struct {
	Case     domain.OralHistoryCase     `json:"case"`
	Segments []domain.TranscriptSegment `json:"segments"`
}
type ReviseTimecodeCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	MediaDigest     string `json:"mediaDigest"`
	StartMillis     int64  `json:"startMillis"`
	EndMillis       int64  `json:"endMillis"`
}
type TimecodeRevisionResult struct {
	Case       domain.OralHistoryCase   `json:"case"`
	Segment    domain.TranscriptSegment `json:"segment"`
	GateReport domain.GateReport        `json:"gateReport"`
}
type ReviewCommand struct {
	ExpectedVersion int64                 `json:"expectedVersion"`
	IdempotencyKey  string                `json:"idempotencyKey"`
	DecisionStatus  domain.DecisionStatus `json:"decisionStatus"`
	RedactionText   string                `json:"redactionText"`
	Reason          string                `json:"reason"`
	ReviewedBy      string                `json:"reviewedBy"`
}
type ConfirmCommand struct {
	ExpectedVersion    int64    `json:"expectedVersion"`
	IdempotencyKey     string   `json:"idempotencyKey"`
	Confirmed          bool     `json:"confirmed"`
	ReturnedSegmentIDs []string `json:"returnedSegmentIDs"`
	Comment            string   `json:"comment"`
}
type ReleaseCommand struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	IssuedBy        string `json:"issuedBy"`
}
