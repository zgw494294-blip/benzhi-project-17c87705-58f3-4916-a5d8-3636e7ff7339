package application

import (
	"context"
	"fmt"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) AddSegment(ctx context.Context, caseID string, cmd AddSegmentCommand) (domain.TranscriptSegment, error) {
	p, err := authorize(ctx, RoleCollector)
	if err != nil {
		return domain.TranscriptSegment{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.TranscriptSegment{}, err
	}
	exists, err := s.store.HasMedia(ctx, cmd.MediaDigest, caseID)
	if err != nil {
		return domain.TranscriptSegment{}, err
	}
	if !exists {
		return domain.TranscriptSegment{}, domain.NewRuleError(domain.CodeValidation, "mediaDigest", "案卷未引用该音频摘要")
	}
	seg, err := domain.NewSegment(newID("seg"), caseID, cmd.MediaDigest, cmd.Speaker, cmd.StartMillis, cmd.EndMillis, cmd.Transcript, cmd.SensitivityTags)
	if err != nil {
		return domain.TranscriptSegment{}, err
	}
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		c, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(c.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		if c.Status != domain.StatusConsentLocked && c.Status != domain.StatusTranscribing {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "当前阶段不能录入转录")
		}
		expected := c.Version
		if c.Status == domain.StatusConsentLocked {
			err = c.Transition(domain.StatusTranscribing, s.now())
		} else {
			bump(&c, s.now())
		}
		if err != nil {
			return err
		}
		if err = tx.InsertSegment(ctx, *seg); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, c, expected); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "TRANSCRIPT_ADDED", Actor: p.Name, Detail: fmt.Sprintf("录入片段 %s，%d-%d ms", seg.SegmentID, seg.StartMillis, seg.EndMillis), Version: c.Version, OccurredAt: s.now()})
	})
	return *seg, mapStorage(err)
}
func (s *Service) ListSegments(ctx context.Context, caseID string) ([]domain.TranscriptSegment, error) {
	return s.store.ListSegments(ctx, caseID)
}
