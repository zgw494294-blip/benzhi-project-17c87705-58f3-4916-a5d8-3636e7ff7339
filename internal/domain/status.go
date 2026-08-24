package domain

type CaseStatus string

const (
	StatusDraft                CaseStatus = "DRAFT"
	StatusConsentLocked        CaseStatus = "CONSENT_LOCKED"
	StatusTranscribing         CaseStatus = "TRANSCRIBING"
	StatusAwaitingConfirmation CaseStatus = "AWAITING_CONFIRMATION"
	StatusApproved             CaseStatus = "APPROVED"
	StatusReleased             CaseStatus = "RELEASED"
)

var transitions = map[CaseStatus]map[CaseStatus]bool{
	StatusDraft:                {StatusConsentLocked: true},
	StatusConsentLocked:        {StatusTranscribing: true},
	StatusTranscribing:         {StatusAwaitingConfirmation: true},
	StatusAwaitingConfirmation: {StatusTranscribing: true, StatusApproved: true},
	StatusApproved:             {StatusReleased: true},
}

func CanTransition(from, to CaseStatus) bool { return transitions[from][to] }
