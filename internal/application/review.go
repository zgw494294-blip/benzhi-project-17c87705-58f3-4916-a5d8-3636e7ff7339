package application

import (
	"context"
	"fmt"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) ReviewSegment(ctx context.Context, caseID, segmentID string, cmd ReviewCommand) (domain.TranscriptSegment, error) {
	p, err := authorize(ctx, RoleReviewer)
	if err != nil {
		return domain.TranscriptSegment{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.TranscriptSegment{}, err
	}
	seg, err := s.store.GetSegment(ctx, segmentID)
	if err != nil {
		return seg, mapStorage(err)
	}
	if seg.CaseID != caseID {
		return seg, domain.NewRuleError(domain.CodeNotFound, "segmentID", "案卷中不存在该片段")
	}
	if err = seg.Decide(cmd.DecisionStatus, cmd.RedactionText, cmd.Reason, cmd.ReviewedBy); err != nil {
		return seg, err
	}
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		c, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(c.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		if c.Status != domain.StatusTranscribing {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "只有转录阶段可执行脱敏裁定")
		}
		expected := c.Version
		bump(&c, s.now())
		if err = tx.UpdateSegmentDecision(ctx, seg); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, c, expected); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "REDACTION_DECIDED", Actor: p.Name, Detail: fmt.Sprintf("片段 %s 裁定为 %s：%s", segmentID, cmd.DecisionStatus, cmd.Reason), Version: c.Version, OccurredAt: s.now()})
	})
	if err != nil {
		return seg, mapStorage(err)
	}
	return seg, nil
}
func (s *Service) CompleteReview(ctx context.Context, caseID string, expected int64) (domain.OralHistoryCase, error) {
	p, err := authorize(ctx, RoleReviewer)
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	segments, err := s.store.ListSegments(ctx, caseID)
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	if len(segments) == 0 {
		return domain.OralHistoryCase{}, domain.NewRuleError(domain.CodeGateFailed, "segments", "没有可审查的转录片段")
	}
	for _, seg := range segments {
		if seg.DecisionStatus == domain.DecisionPending {
			return domain.OralHistoryCase{}, domain.NewRuleError(domain.CodeGateFailed, "decisionStatus", "片段 %s 尚未裁定", seg.SegmentID)
		}
	}
	var out domain.OralHistoryCase
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		c, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(c.Version, expected); err != nil {
			return err
		}
		old := c.Version
		if err = c.Transition(domain.StatusAwaitingConfirmation, s.now()); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, c, old); err != nil {
			return err
		}
		if err = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "REVIEW_COMPLETED", Actor: p.Name, Detail: "全部敏感片段已完成裁定", Version: c.Version, OccurredAt: s.now()}); err != nil {
			return err
		}
		out = c
		return nil
	})
	return out, mapStorage(err)
}
