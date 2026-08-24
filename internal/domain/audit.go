package domain

import "time"

type AuditEvent struct {
	EventID    int64     `json:"eventID"`
	CaseID     string    `json:"caseID"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Detail     string    `json:"detail"`
	Version    int64     `json:"version"`
	OccurredAt time.Time `json:"occurredAt"`
}

type Confirmation struct {
	CaseID             string    `json:"caseID"`
	Confirmed          bool      `json:"confirmed"`
	ReturnedSegmentIDs []string  `json:"returnedSegmentIDs"`
	Comment            string    `json:"comment"`
	Actor              string    `json:"actor"`
	DecidedAt          time.Time `json:"decidedAt"`
}
