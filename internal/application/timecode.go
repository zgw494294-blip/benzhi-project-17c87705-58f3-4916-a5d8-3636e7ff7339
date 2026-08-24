package application

import (
	"context"
	"encoding/json"
	"fmt"

	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) buildGateReport(ctx context.Context, caseID string) (domain.GateReport, error) {
	c, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return domain.GateReport{}, mapStorage(err)
	}
	consent, _ := s.store.GetConsent(ctx, caseID)
	segments, err := s.store.ListSegments(ctx, caseID)
	if err != nil {
		return domain.GateReport{}, err
	}
	confirmation, _ := s.store.LatestConfirmation(ctx, caseID)
	return domain.BuildGateReport(c, consent, segments, confirmation, s.now()), nil
}

// ReviseSegmentTimecode changes only a collector-owned segment's timing and
// rechecks all intervals on the same audio before committing the version bump.
func (s *Service) ReviseSegmentTimecode(ctx context.Context, caseID, segmentID string, cmd ReviseTimecodeCommand) (TimecodeRevisionResult, error) {
	p, err := authorize(ctx, RoleCollector)
	if err != nil {
		return TimecodeRevisionResult{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return TimecodeRevisionResult{}, err
	}
	if cmd.MediaDigest == "" {
		return TimecodeRevisionResult{}, domain.NewRuleError(domain.CodeValidation, "mediaDigest", "mediaDigest 不能为空")
	}
	var revised domain.TranscriptSegment
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		if cached, ok, lookupErr := tx.GetIdempotency(ctx, cmd.IdempotencyKey, "timecode-revision", caseID); lookupErr != nil {
			return lookupErr
		} else if ok {
			if unmarshalErr := json.Unmarshal(cached, &revised); unmarshalErr != nil {
				return domain.NewRuleError(domain.CodeIntegrityFailed, "idempotencyKey", "时间码修订幂等结果无法读取")
			}
			return nil
		}
		c, txErr := tx.GetCase(ctx, caseID)
		if txErr != nil {
			return txErr
		}
		if txErr = requireVersion(c.Version, cmd.ExpectedVersion); txErr != nil {
			return txErr
		}
		if c.Status != domain.StatusConsentLocked && c.Status != domain.StatusTranscribing {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "当前阶段不能修订时间码")
		}
		seg, txErr := tx.GetSegment(ctx, segmentID)
		if txErr != nil {
			return txErr
		}
		if seg.CaseID != caseID {
			return domain.NewRuleError(domain.CodeNotFound, "segmentID", "案卷中不存在该片段")
		}
		if seg.MediaDigest != cmd.MediaDigest {
			return domain.NewRuleError(domain.CodeNotFound, "mediaDigest", "修订目标不属于指定音频摘要")
		}
		if cmd.StartMillis < 0 || cmd.EndMillis <= cmd.StartMillis {
			return domain.NewRuleError(domain.CodeValidation, "endMillis", "时间码必须为非负且结束时间严格晚于开始时间")
		}
		oldStart, oldEnd := seg.StartMillis, seg.EndMillis
		seg.StartMillis, seg.EndMillis = cmd.StartMillis, cmd.EndMillis
		segments, listErr := tx.ListSegments(ctx, caseID)
		if listErr != nil {
			return listErr
		}
		for i := range segments {
			if segments[i].SegmentID == segmentID {
				segments[i] = seg
			}
		}
		if listErr = domain.ValidateNoOverlaps(segments); listErr != nil {
			return listErr
		}
		if txErr = tx.UpdateSegmentTiming(ctx, seg); txErr != nil {
			return txErr
		}
		expected := c.Version
		bump(&c, s.now())
		if txErr = tx.UpdateCase(ctx, c, expected); txErr != nil {
			return txErr
		}
		if txErr = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "TIMECODE_REVISED", Actor: p.Name, Detail: fmt.Sprintf("片段 %s 时间码 %d-%d ms 修订为 %d-%d ms", segmentID, oldStart, oldEnd, seg.StartMillis, seg.EndMillis), Version: c.Version, OccurredAt: s.now()}); txErr != nil {
			return txErr
		}
		revised = seg
		response, marshalErr := json.Marshal(revised)
		if marshalErr != nil {
			return marshalErr
		}
		return tx.PutIdempotency(ctx, cmd.IdempotencyKey, "timecode-revision", caseID, response)
	})
	if err != nil {
		return TimecodeRevisionResult{}, mapStorage(err)
	}
	report, err := s.buildGateReport(ctx, caseID)
	if err != nil {
		return TimecodeRevisionResult{}, err
	}
	c, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return TimecodeRevisionResult{}, mapStorage(err)
	}
	return TimecodeRevisionResult{Case: c, Segment: revised, GateReport: report}, nil
}

func (s *Service) ReviseTimecode(ctx context.Context, caseID, segmentID string, cmd ReviseTimecodeCommand) (TimecodeRevisionResult, error) {
	return s.ReviseSegmentTimecode(ctx, caseID, segmentID, cmd)
}
