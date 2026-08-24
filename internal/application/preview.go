package application

import (
	"context"
	"encoding/json"
	"fmt"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) Preview(ctx context.Context, caseID string) (domain.Preview, error) {
	c, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return domain.Preview{}, mapStorage(err)
	}
	consent, _ := s.store.GetConsent(ctx, caseID)
	segments, err := s.store.ListSegments(ctx, caseID)
	if err != nil {
		return domain.Preview{}, err
	}
	p := domain.BuildPreview(c, consent, segments)
	confirmation, _ := s.store.LatestConfirmation(ctx, caseID)
	p.Issues = domain.CheckGates(c, consent, segments, confirmation, s.now())
	return p, nil
}
func (s *Service) ConfirmPreview(ctx context.Context, caseID string, cmd ConfirmCommand) (domain.OralHistoryCase, error) {
	p, err := authorize(ctx, RoleRepresentative)
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.OralHistoryCase{}, err
	}
	var out domain.OralHistoryCase
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		if cached, ok, lookupErr := tx.GetIdempotency(ctx, cmd.IdempotencyKey, "confirmation", caseID); lookupErr != nil {
			return lookupErr
		} else if ok {
			return json.Unmarshal(cached, &out)
		}
		c, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(c.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		if c.Status != domain.StatusAwaitingConfirmation {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "案卷尚未进入受访者确认阶段")
		}
		if !cmd.Confirmed && len(cmd.ReturnedSegmentIDs) == 0 {
			return domain.NewRuleError(domain.CodeValidation, "returnedSegmentIDs", "退回时必须指定至少一个片段")
		}
		if cmd.Confirmed && len(cmd.ReturnedSegmentIDs) > 0 {
			return domain.NewRuleError(domain.CodeValidation, "returnedSegmentIDs", "确认分支不能包含退回片段")
		}
		segments, err := tx.ListSegments(ctx, caseID)
		if err != nil {
			return err
		}
		visible := make(map[string]domain.TranscriptSegment, len(segments))
		for _, seg := range segments {
			if seg.DecisionStatus != domain.DecisionRemove {
				visible[seg.SegmentID] = seg
			}
		}
		returned := make([]string, 0, len(cmd.ReturnedSegmentIDs))
		seen := make(map[string]bool, len(cmd.ReturnedSegmentIDs))
		for i, id := range cmd.ReturnedSegmentIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if _, ok := visible[id]; !ok {
				return domain.NewRuleError(domain.CodeValidation, fmt.Sprintf("returnedSegmentIDs[%d]", i), "片段不属于当前可见预览：%s", id)
			}
			returned = append(returned, id)
		}
		confirmation := domain.Confirmation{CaseID: caseID, Confirmed: cmd.Confirmed, ReturnedSegmentIDs: returned, Comment: cmd.Comment, Actor: p.Name, DecidedAt: s.now()}
		old := c.Version
		to := domain.StatusTranscribing
		action := "PREVIEW_RETURNED"
		detail := fmt.Sprintf("退回 %d 个片段：%v；评论：%s", len(returned), returned, cmd.Comment)
		if cmd.Confirmed {
			to = domain.StatusApproved
			action = "PREVIEW_CONFIRMED"
			detail = "受访者确认当前发布预览"
		}
		if err = c.Transition(to, s.now()); err != nil {
			return err
		}
		if !cmd.Confirmed {
			for _, id := range returned {
				seg := visible[id]
				seg.DecisionStatus = domain.DecisionPending
				seg.RedactionText = ""
				seg.DecisionReason = ""
				seg.ReviewedBy = ""
				if err = tx.UpdateSegmentRemediation(ctx, seg); err != nil {
					return err
				}
			}
		}
		if err = tx.InsertConfirmation(ctx, confirmation); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, c, old); err != nil {
			return err
		}
		if err = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: action, Actor: p.Name, Detail: detail, Version: c.Version, OccurredAt: s.now()}); err != nil {
			return err
		}
		out = c
		response, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return marshalErr
		}
		return tx.PutIdempotency(ctx, cmd.IdempotencyKey, "confirmation", caseID, response)
	})
	return out, mapStorage(err)
}
