package application

import (
	"context"
	"encoding/json"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
	"strings"
)

func (s *Service) CreateCase(ctx context.Context, cmd CreateCaseCommand) (domain.OralHistoryCase, error) {
	p, err := authorize(ctx, RoleCollector)
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.OralHistoryCase{}, err
	}
	collector := strings.TrimSpace(cmd.Collector)
	if collector == "" {
		collector = p.Name
	}
	var result domain.OralHistoryCase
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		if cached, ok, lookupErr := tx.GetIdempotencyByKey(ctx, cmd.IdempotencyKey, "case"); lookupErr != nil {
			return lookupErr
		} else if ok {
			return json.Unmarshal(cached, &result)
		}
		c, newErr := domain.NewCase(newID("case"), cmd.Title, cmd.IntervieweeAlias, collector, s.now())
		if newErr != nil {
			return newErr
		}
		if err = tx.InsertCase(ctx, *c); err != nil {
			return err
		}
		if err = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: c.CaseID, Action: "CASE_CREATED", Actor: p.Name, Detail: "建立口述史采集案卷", Version: c.Version, OccurredAt: c.CreatedAt}); err != nil {
			return err
		}
		result = *c
		response, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		return tx.PutIdempotency(ctx, cmd.IdempotencyKey, "case", c.CaseID, response)
	})
	if err != nil {
		return domain.OralHistoryCase{}, mapStorage(err)
	}
	return result, nil
}
func (s *Service) GetCase(ctx context.Context, id string) (domain.OralHistoryCase, error) {
	c, err := s.store.GetCase(ctx, id)
	return c, mapStorage(err)
}
func (s *Service) ListCases(ctx context.Context) ([]domain.OralHistoryCase, error) {
	return s.store.ListCases(ctx)
}
