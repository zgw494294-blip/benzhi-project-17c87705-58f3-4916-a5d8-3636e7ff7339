package application

import (
	"context"
	"encoding/json"
	"fmt"

	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func indexedSegmentError(index int, err error) error {
	if rule, ok := err.(*domain.RuleError); ok {
		field := rule.Field
		if field == "" {
			field = "segment"
		}
		return domain.NewRuleError(rule.Code, fmt.Sprintf("segments[%d].%s", index, field), "%s", rule.Message)
	}
	return err
}

func batchOverlapError(existing, incoming []domain.TranscriptSegment) error {
	all := append(append([]domain.TranscriptSegment{}, existing...), incoming...)
	sorted := domain.NormalizeSegments(all)
	for i := 1; i < len(sorted); i++ {
		previous, current := sorted[i-1], sorted[i]
		if previous.MediaDigest != current.MediaDigest || previous.EndMillis <= current.StartMillis {
			continue
		}
		for index, candidate := range incoming {
			if candidate.SegmentID == current.SegmentID || candidate.SegmentID == previous.SegmentID {
				return domain.NewRuleError(domain.CodeValidation, fmt.Sprintf("segments[%d].startMillis", index), "片段 %s 与 %s 在同一音频上时间段重叠", current.SegmentID, previous.SegmentID)
			}
		}
	}
	return domain.NewRuleError(domain.CodeValidation, "segments", "批次时间码存在重叠")
}

// ImportTranscriptBatch atomically imports a normalized transcript batch.
func (s *Service) ImportTranscriptBatch(ctx context.Context, caseID string, cmd ImportTranscriptBatchCommand) (BatchTranscriptResult, error) {
	p, err := authorize(ctx, RoleCollector)
	if err != nil {
		return BatchTranscriptResult{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return BatchTranscriptResult{}, err
	}
	if len(cmd.Segments) == 0 {
		return BatchTranscriptResult{}, domain.NewRuleError(domain.CodeValidation, "segments", "segments 至少包含一个片段")
	}
	var result BatchTranscriptResult
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		if cached, ok, lookupErr := tx.GetIdempotency(ctx, cmd.IdempotencyKey, "transcript-batch", caseID); lookupErr != nil {
			return lookupErr
		} else if ok {
			return json.Unmarshal(cached, &result)
		}
		c, txErr := tx.GetCase(ctx, caseID)
		if txErr != nil {
			return txErr
		}
		if txErr = requireVersion(c.Version, cmd.ExpectedVersion); txErr != nil {
			return txErr
		}
		if c.Status != domain.StatusConsentLocked && c.Status != domain.StatusTranscribing {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "当前阶段不能批量导入转录")
		}
		existing, txErr := tx.ListSegments(ctx, caseID)
		if txErr != nil {
			return txErr
		}
		seenIDs := make(map[string]bool, len(existing)+len(cmd.Segments))
		for _, seg := range existing {
			seenIDs[seg.SegmentID] = true
		}
		incoming := make([]domain.TranscriptSegment, 0, len(cmd.Segments))
		for i, input := range cmd.Segments {
			id := input.SegmentID
			if id == "" {
				id = newID("seg")
			}
			if seenIDs[id] {
				return domain.NewRuleError(domain.CodeValidation, fmt.Sprintf("segments[%d].segmentID", i), "片段编号重复：%s", id)
			}
			seenIDs[id] = true
			ok, hasErr := tx.HasMedia(ctx, input.MediaDigest, caseID)
			if hasErr != nil {
				return hasErr
			}
			if !ok {
				return domain.NewRuleError(domain.CodeValidation, fmt.Sprintf("segments[%d].mediaDigest", i), "案卷未引用该音频摘要")
			}
			seg, newErr := domain.NewSegment(id, caseID, input.MediaDigest, input.Speaker, input.StartMillis, input.EndMillis, input.Transcript, input.SensitivityTags)
			if newErr != nil {
				return indexedSegmentError(i, newErr)
			}
			incoming = append(incoming, *seg)
		}
		all := append(append([]domain.TranscriptSegment{}, existing...), incoming...)
		if txErr = domain.ValidateNoOverlaps(all); txErr != nil {
			return batchOverlapError(existing, incoming)
		}
		ordered := domain.NormalizeSegments(incoming)
		expected := c.Version
		if c.Status == domain.StatusConsentLocked {
			txErr = c.Transition(domain.StatusTranscribing, s.now())
		} else {
			bump(&c, s.now())
		}
		if txErr != nil {
			return txErr
		}
		for _, seg := range ordered {
			if txErr = tx.InsertSegment(ctx, seg); txErr != nil {
				return txErr
			}
		}
		digest := domain.StableDigest(ordered)
		if txErr = tx.UpdateCase(ctx, c, expected); txErr != nil {
			return txErr
		}
		if txErr = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "TRANSCRIPT_BATCH_IMPORTED", Actor: p.Name, Detail: fmt.Sprintf("批量导入 %d 个片段，批次摘要 %s", len(ordered), digest), Version: c.Version, OccurredAt: s.now()}); txErr != nil {
			return txErr
		}
		result = BatchTranscriptResult{Case: c, Segments: ordered}
		response, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		return tx.PutIdempotency(ctx, cmd.IdempotencyKey, "transcript-batch", caseID, response)
	})
	return result, mapStorage(err)
}

// BatchImportSegments is kept as a descriptive alias for callers outside HTTP.
func (s *Service) BatchImportSegments(ctx context.Context, caseID string, cmd ImportTranscriptBatchCommand) (BatchTranscriptResult, error) {
	return s.ImportTranscriptBatch(ctx, caseID, cmd)
}
