package consentlockcancel_test

import (
	"context"
	"errors"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
)

func TestCancelledConsentLockDoesNotCommitState(t *testing.T) {
	store, err := storage.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := application.New(store)

	collector := application.WithPrincipal(context.Background(), application.Principal{Name: "采集员", Role: application.RoleCollector})
	c, err := svc.CreateCase(collector, application.CreateCaseCommand{
		Title: "取消冻结测试", IntervieweeAlias: "受访者", Collector: "采集员", IdempotencyKey: "cancel-case",
	})
	if err != nil {
		t.Fatal(err)
	}

	representative := application.WithPrincipal(context.Background(), application.Principal{Name: "代理人", Role: application.RoleRepresentative})
	cancelled, cancel := context.WithCancel(representative)
	cancel()
	_, err = svc.LockConsent(cancelled, c.CaseID, application.LockConsentCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "cancel-consent", AllowedAudiences: []string{"公众"},
		AllowedPurposes: []string{"数字档案"}, WithdrawalTerms: "发布前书面撤回", ConfirmedBy: "代理人",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消请求应返回 context.Canceled，得到 %v", err)
	}

	current, err := svc.GetCase(context.Background(), c.CaseID)
	if err != nil {
		t.Fatal(err)
	}
	consent, consentErr := store.GetConsent(context.Background(), c.CaseID)
	if current.Status != domain.StatusDraft || current.Version != c.Version || !errors.Is(consentErr, storage.ErrNotFound) {
		t.Fatalf("已取消的同意冻结仍提交状态：status=%s version=%d consent=%#v err=%v", current.Status, current.Version, consent, consentErr)
	}
}
