package releaseverificationcachestale_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"oralarchive/internal/httpapi"
	"oralarchive/internal/storage"
)

func principal(role application.Role, name string) context.Context {
	return application.WithPrincipal(context.Background(), application.Principal{Name: name, Role: role})
}

func releasedCase(t *testing.T, root string) (*application.Service, domain.OralHistoryCase, domain.ReleasePackage) {
	t.Helper()
	store, err := storage.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := application.New(store)

	c, err := svc.CreateCase(principal(application.RoleCollector, "采集员"), application.CreateCaseCommand{
		Title: "缓存失效核验", IntervieweeAlias: "受访者甲", Collector: "采集员", IdempotencyKey: "cache-case",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.LockConsent(principal(application.RoleRepresentative, "代理人"), c.CaseID, application.LockConsentCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "cache-consent", AllowedAudiences: []string{"公众"},
		AllowedPurposes: []string{"数字档案"}, WithdrawalTerms: "发布前书面撤回", ConfirmedBy: "代理人",
	})
	if err != nil {
		t.Fatal(err)
	}
	media, err := svc.UploadMedia(principal(application.RoleCollector, "采集员"), application.UploadMediaCommand{
		CaseID: c.CaseID, ContentType: "audio/wav", Reader: bytes.NewBufferString("RIFF cache verification fixture"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddSegment(principal(application.RoleCollector, "采集员"), c.CaseID, application.AddSegmentCommand{
		ExpectedVersion: 2, IdempotencyKey: "cache-segment", MediaDigest: media.Digest,
		Speaker: "受访者甲", StartMillis: 0, EndMillis: 1000, Transcript: "公开访谈内容",
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.CompleteReview(principal(application.RoleReviewer, "审查员"), c.CaseID, 3)
	if err != nil {
		t.Fatal(err)
	}
	c, err = svc.ConfirmPreview(principal(application.RoleRepresentative, "代理人"), c.CaseID, application.ConfirmCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "cache-confirm", Confirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := svc.Release(principal(application.RoleArchivist, "发布员"), c.CaseID, application.ReleaseCommand{
		ExpectedVersion: c.Version, IdempotencyKey: "cache-release", IssuedBy: "发布员",
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, c, pkg
}

func fetchVerification(t *testing.T, baseURL, caseID string) domain.ReleaseVerification {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/cases/"+caseID+"/release/verify", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Actor", "发布员")
	req.Header.Set("X-Role", string(application.RoleArchivist))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("核验请求状态异常：%d", resp.StatusCode)
	}
	var report domain.ReleaseVerification
	if err = json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	return report
}

func TestVerificationRechecksManifestAfterCachedSuccess(t *testing.T) {
	root := t.TempDir()
	svc, c, pkg := releasedCase(t, root)
	server := httptest.NewServer(httpapi.New(svc).Handler())
	defer server.Close()

	first := fetchVerification(t, server.URL, c.CaseID)
	if !first.Valid {
		t.Fatalf("初次核验应通过：%#v", first.Failures)
	}

	manifestName := strings.TrimPrefix(pkg.ManifestDigest, "sha256:") + ".json"
	manifestPath := filepath.Join(root, "manifests", manifestName)
	if err := os.WriteFile(manifestPath, []byte(`{"tampered":true}`), 0600); err != nil {
		t.Fatal(err)
	}

	second := fetchVerification(t, server.URL, c.CaseID)
	if second.Valid {
		t.Fatal("清单文件失效后再次核验仍返回 valid=true")
	}
	found := false
	for _, failure := range second.Failures {
		if failure.Code == "MANIFEST_READ_FAILED" || failure.Code == "MANIFEST_DIGEST_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("再次核验未报告清单完整性故障：%#v", second.Failures)
	}
}
