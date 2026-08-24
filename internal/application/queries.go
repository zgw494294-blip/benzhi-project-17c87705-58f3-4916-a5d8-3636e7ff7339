package application

import (
	"context"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

type CaseDetail struct {
	Case          domain.OralHistoryCase     `json:"case"`
	Consent       *domain.ConsentScope       `json:"consent,omitempty"`
	Segments      []domain.TranscriptSegment `json:"segments"`
	Confirmation  *domain.Confirmation       `json:"confirmation,omitempty"`
	Confirmations []domain.Confirmation      `json:"confirmations"`
	Package       *domain.ReleasePackage     `json:"releasePackage,omitempty"`
	Manifest      *domain.Manifest           `json:"manifest,omitempty"`
	Audit         []domain.AuditEvent        `json:"audit"`
	Gates         []domain.GateIssue         `json:"gates"`
	GateReport    domain.GateReport          `json:"gateReport"`
	Media         []storage.MediaObject      `json:"media"`
	Integrity     storage.IntegrityStatus    `json:"integrity"`
}

func (s *Service) CaseDetail(ctx context.Context, id string) (CaseDetail, error) {
	c, err := s.GetCase(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	consent, _ := s.store.GetConsent(ctx, id)
	segments, err := s.store.ListSegments(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	confirmation, _ := s.store.LatestConfirmation(ctx, id)
	confirmations, err := s.store.ListConfirmations(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	pkg, _ := s.store.GetPackageByCase(ctx, id)
	var manifest *domain.Manifest
	if pkg != nil {
		manifest, _ = s.store.ReadManifest(ctx, *pkg)
	}
	audit, err := s.store.ListAudit(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	media, err := s.store.ListMedia(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	integrity, err := s.store.IntegrityStatus(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	report := domain.BuildGateReport(c, consent, segments, confirmation, s.now())
	return CaseDetail{Case: c, Consent: consent, Segments: segments, Confirmation: confirmation, Confirmations: confirmations, Package: pkg, Manifest: manifest, Audit: audit, Gates: report.Issues, GateReport: report, Media: media, Integrity: integrity}, nil
}
