package application

import (
	"context"
	"oralarchive/internal/domain"
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
	c, err := domain.NewCase(newID("case"), cmd.Title, cmd.IntervieweeAlias, collector, s.now())
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	if err = s.store.CreateCase(ctx, *c, p.Name); err != nil {
		return domain.OralHistoryCase{}, mapStorage(err)
	}
	return *c, nil
}
func (s *Service) GetCase(ctx context.Context, id string) (domain.OralHistoryCase, error) {
	c, err := s.store.GetCase(ctx, id)
	return c, mapStorage(err)
}
func (s *Service) ListCases(ctx context.Context) ([]domain.OralHistoryCase, error) {
	return s.store.ListCases(ctx)
}
