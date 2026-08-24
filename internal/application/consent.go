package application

import (
	"context"
	"fmt"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func (s *Service) LockConsent(ctx context.Context, caseID string, cmd LockConsentCommand) (domain.ConsentScope, error) {
	p, err := authorize(ctx, RoleCollector, RoleRepresentative)
	if err != nil {
		return domain.ConsentScope{}, err
	}
	if err = requireKey(cmd.IdempotencyKey); err != nil {
		return domain.ConsentScope{}, err
	}
	var result domain.ConsentScope
	err = s.store.WithTx(ctx, func(tx *storage.Tx) error {
		c, err := tx.GetCase(ctx, caseID)
		if err != nil {
			return err
		}
		if err = requireVersion(c.Version, cmd.ExpectedVersion); err != nil {
			return err
		}
		if c.Status != domain.StatusDraft {
			return domain.NewRuleError(domain.CodeInvalidState, "status", "只有草稿案卷可冻结同意范围")
		}
		consent, err := domain.NewConsent(newID("consent"), caseID, c.Version, cmd.AllowedAudiences, cmd.AllowedPurposes, cmd.EmbargoUntil, cmd.WithdrawalTerms, cmd.ConfirmedBy, s.now())
		if err != nil {
			return err
		}
		expected := c.Version
		if err = c.Transition(domain.StatusConsentLocked, s.now()); err != nil {
			return err
		}
		if err = tx.InsertConsent(ctx, *consent); err != nil {
			return err
		}
		if err = tx.UpdateCase(ctx, c, expected); err != nil {
			return err
		}
		if err = tx.AppendAudit(ctx, domain.AuditEvent{CaseID: caseID, Action: "CONSENT_LOCKED", Actor: p.Name, Detail: fmt.Sprintf("冻结同意版本 %d，摘要 %s", consent.Version, consent.Digest), Version: c.Version, OccurredAt: s.now()}); err != nil {
			return err
		}
		result = *consent
		return nil
	})
	return result, mapStorage(err)
}
func (s *Service) GetConsent(ctx context.Context, caseID string) (*domain.ConsentScope, error) {
	c, err := s.store.GetConsent(ctx, caseID)
	return c, mapStorage(err)
}
