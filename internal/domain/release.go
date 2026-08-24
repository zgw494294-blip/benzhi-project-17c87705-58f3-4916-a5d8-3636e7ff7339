package domain

import "time"

type ReleasePackage struct {
	PackageID      string    `json:"packageID"`
	CaseID         string    `json:"caseID"`
	ManifestDigest string    `json:"manifestDigest"`
	ConsentDigest  string    `json:"consentDigest"`
	SegmentDigests []string  `json:"segmentDigests"`
	IssuedAt       time.Time `json:"issuedAt"`
	IssuedBy       string    `json:"issuedBy"`
}

type Manifest struct {
	Schema           string            `json:"schema"`
	PackageID        string            `json:"packageID"`
	CaseID           string            `json:"caseID"`
	Title            string            `json:"title"`
	IntervieweeAlias string            `json:"intervieweeAlias"`
	ConsentDigest    string            `json:"consentDigest"`
	Segments         []ManifestSegment `json:"segments"`
	IssuedAt         time.Time         `json:"issuedAt"`
	IssuedBy         string            `json:"issuedBy"`
}
type ManifestSegment struct {
	SegmentID   string `json:"segmentID"`
	Speaker     string `json:"speaker"`
	StartMillis int64  `json:"startMillis"`
	EndMillis   int64  `json:"endMillis"`
	Text        string `json:"text"`
	Digest      string `json:"digest"`
}

type VerificationFailure struct {
	Code      string `json:"code"`
	Field     string `json:"field"`
	SegmentID string `json:"segmentID,omitempty"`
	Message   string `json:"message"`
}

type ReleaseVerification struct {
	Valid          bool                  `json:"valid"`
	CheckedAt      time.Time             `json:"checkedAt"`
	Package        *ReleasePackage       `json:"package,omitempty"`
	Manifest       *Manifest             `json:"manifest,omitempty"`
	Failures       []VerificationFailure `json:"failures"`
	ManifestDigest string                `json:"manifestDigest,omitempty"`
}
