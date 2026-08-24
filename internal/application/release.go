package application

import (
	"context"
	"encoding/json"
	"fmt"

	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) Release(ctx context.Context, caseID string, cmd ReleaseCommand) (domain.ReleasePackage, error) {
	p, err := authorize(ctx, RoleArchivist)
	if err != nil {
		return domain.ReleasePackage{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.ReleasePackage{}, err
	}
	if cached, ok, lookupErr := s.store.GetIdempotency(ctx, cmd.IdempotencyKey, "release", caseID); lookupErr != nil {
		return domain.ReleasePackage{}, lookupErr
	} else if ok {
		var prior domain.ReleasePackage
		if err = json.Unmarshal(cached, &prior); err != nil {
			return prior, err
		}
		return prior, nil
	}
	c, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return domain.ReleasePackage{}, mapStorage(err)
	}
	if err = requireVersion(c.Version, cmd.ExpectedVersion); err != nil {
		return domain.ReleasePackage{}, err
	}
	consent, err := s.store.GetConsent(ctx, caseID)
	if err != nil {
		return domain.ReleasePackage{}, mapStorage(err)
	}
	segments, err := s.store.ListSegments(ctx, caseID)
	if err != nil {
		return domain.ReleasePackage{}, err
	}
	confirmation, _ := s.store.LatestConfirmation(ctx, caseID)
	issues := domain.CheckGates(c, consent, segments, confirmation, s.now())
	if c.Status != domain.StatusApproved {
		issues = append(issues, domain.GateIssue{Code: "NOT_APPROVED", Field: "status", Message: "案卷尚未通过受访者确认"})
	}
	if len(issues) > 0 {
		return domain.ReleasePackage{}, &GateError{Issues: issues}
	}
	issuedBy := cmd.IssuedBy
	if issuedBy == "" {
		issuedBy = p.Name
	}
	issuedAt, packageID := s.now().UTC(), newID("pkg")
	manifest := domain.Manifest{Schema: "oralarchive-manifest/v1", PackageID: packageID, CaseID: caseID, Title: c.Title, IntervieweeAlias: c.IntervieweeAlias, ConsentDigest: consent.Digest, IssuedAt: issuedAt, IssuedBy: issuedBy}
	digests := []string{}
	for _, seg := range segments {
		if seg.DecisionStatus == domain.DecisionRemove {
			continue
		}
		d := seg.Digest()
		digests = append(digests, d)
		manifest.Segments = append(manifest.Segments, domain.ManifestSegment{SegmentID: seg.SegmentID, Speaker: seg.Speaker, StartMillis: seg.StartMillis, EndMillis: seg.EndMillis, Text: seg.PublicText(), Digest: d})
	}
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	pkg := domain.ReleasePackage{PackageID: packageID, CaseID: caseID, ManifestDigest: domain.BytesDigest(manifestBytes), ConsentDigest: consent.Digest, SegmentDigests: digests, IssuedAt: issuedAt, IssuedBy: issuedBy}
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		current, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(current.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		old := current.Version
		if err = current.Transition(domain.StatusReleased, s.now()); err != nil {
			return err
		}
		if err = tx.InsertPackage(ctx, pkg, manifestBytes); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, current, old); err != nil {
			return err
		}
		if err = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "PACKAGE_RELEASED", Actor: p.Name, Detail: fmt.Sprintf("签发发布包 %s，清单摘要 %s", pkg.PackageID, pkg.ManifestDigest), Version: current.Version, OccurredAt: s.now()}); err != nil {
			return err
		}
		response, _ := json.Marshal(pkg)
		return tx.PutIdempotency(ctx, cmd.IdempotencyKey, "release", caseID, response)
	})
	return pkg, mapStorage(err)
}

type GateError struct {
	Issues []domain.GateIssue `json:"issues"`
}

func (e *GateError) Error() string { return "发布门禁未通过" }
