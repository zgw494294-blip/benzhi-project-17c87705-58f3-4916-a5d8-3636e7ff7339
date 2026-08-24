package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"oralarchive/internal/domain"
)

type checkClient struct {
	base   string
	client *http.Client
}

func runSelfcheck(ctx context.Context, base string) error {
	c := &checkClient{base: base, client: &http.Client{}}
	if err := c.webPage(ctx); err != nil {
		return err
	}
	var caseRecord domain.OralHistoryCase
	if err := c.json(ctx, "POST", "/api/cases", "采集员甲", "collector", map[string]any{"title": "城南纺织厂口述史", "intervieweeAlias": "林师傅", "collector": "采集员甲", "idempotencyKey": "selfcheck-case"}, &caseRecord); err != nil {
		return err
	}
	var consent domain.ConsentScope
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/consent", "确认代理人甲", "representative", map[string]any{"expectedVersion": caseRecord.Version, "idempotencyKey": "selfcheck-consent", "allowedAudiences": []string{"公众", "研究人员"}, "allowedPurposes": []string{"数字档案"}, "withdrawalTerms": "发布前可书面撤回", "confirmedBy": "确认代理人甲"}, &consent); err != nil {
		return err
	}
	caseRecord.Version = 2
	var media struct {
		Digest string `json:"digest"`
	}
	if err := c.raw(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/media", "采集员甲", "collector", "audio/wav", []byte("RIFF oral history selfcheck audio"), &media); err != nil {
		return err
	}
	var batch struct {
		Case     domain.OralHistoryCase     `json:"case"`
		Segments []domain.TranscriptSegment `json:"segments"`
	}
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/segments/batch", "采集员甲", "collector", map[string]any{"expectedVersion": int64(2), "idempotencyKey": "selfcheck-segment-batch", "segments": []map[string]any{{"segmentID": "selfcheck-segment", "mediaDigest": media.Digest, "speaker": "林师傅", "startMillis": 0, "endMillis": 4200, "transcript": "我在一九六八年进入城南纺织厂。", "sensitivityTags": []string{"姓名"}}}}, &batch); err != nil {
		return err
	}
	if len(batch.Segments) != 1 {
		return fmt.Errorf("批量导入未返回片段")
	}
	segment := batch.Segments[0]
	if err := c.json(ctx, "PATCH", "/api/cases/"+caseRecord.CaseID+"/segments/"+segment.SegmentID+"/timecode", "采集员甲", "collector", map[string]any{"expectedVersion": int64(3), "idempotencyKey": "selfcheck-timecode", "mediaDigest": media.Digest, "startMillis": int64(100), "endMillis": int64(4300)}, &struct {
		Segment domain.TranscriptSegment `json:"segment"`
	}{}); err != nil {
		return err
	}
	if err := c.json(ctx, "PATCH", "/api/cases/"+caseRecord.CaseID+"/segments/"+segment.SegmentID+"/decision", "审查员甲", "reviewer", map[string]any{"expectedVersion": int64(4), "idempotencyKey": "selfcheck-review", "decisionStatus": "REPLACE", "redactionText": "我在一九六八年进入本地纺织厂。", "reason": "隐去可定位的工作单位", "reviewedBy": "审查员甲"}, &segment); err != nil {
		return err
	}
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/review-complete", "审查员甲", "reviewer", map[string]any{"expectedVersion": int64(5)}, &caseRecord); err != nil {
		return err
	}
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/confirmation", "确认代理人甲", "representative", map[string]any{"expectedVersion": int64(6), "idempotencyKey": "selfcheck-confirm", "confirmed": true, "returnedSegmentIDs": []string{}, "comment": "确认发布文本"}, &caseRecord); err != nil {
		return err
	}
	var pkg domain.ReleasePackage
	release := map[string]any{"expectedVersion": int64(7), "idempotencyKey": "selfcheck-release", "issuedBy": "发布员甲"}
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/release", "发布员甲", "archivist", release, &pkg); err != nil {
		return err
	}
	var repeated domain.ReleasePackage
	if err := c.json(ctx, "POST", "/api/cases/"+caseRecord.CaseID+"/release", "发布员甲", "archivist", release, &repeated); err != nil {
		return fmt.Errorf("幂等发布校验失败: %w", err)
	}
	if pkg.PackageID == "" || pkg.ManifestDigest == "" || pkg.PackageID != repeated.PackageID {
		return fmt.Errorf("发布包摘要或幂等结果无效")
	}
	var verification domain.ReleaseVerification
	if err := c.json(ctx, "GET", "/api/cases/"+caseRecord.CaseID+"/release/verify", "发布员甲", "archivist", nil, &verification); err != nil || !verification.Valid {
		return fmt.Errorf("发布包核验失败: %v", err)
	}
	var detail struct {
		Case           domain.OralHistoryCase `json:"case"`
		ReleasePackage *domain.ReleasePackage `json:"releasePackage"`
		Audit          []domain.AuditEvent    `json:"audit"`
	}
	if err := c.json(ctx, "GET", "/api/cases/"+caseRecord.CaseID, "采集员甲", "collector", nil, &detail); err != nil {
		return err
	}
	if detail.Case.Status != domain.StatusReleased || detail.ReleasePackage == nil || len(detail.Audit) < 7 {
		return fmt.Errorf("发布状态或审计时间线不完整")
	}
	return nil
}

func (c *checkClient) webPage(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/", nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	page := string(body)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, "<html") || !strings.Contains(page, `id="caseView"`) {
		return fmt.Errorf("浏览器工作台未正确交付")
	}
	return nil
}
func (c *checkClient) json(ctx context.Context, method, path, actor, role string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	return c.do(ctx, method, path, actor, role, "application/json", reader, out)
}
func (c *checkClient) raw(ctx context.Context, method, path, actor, role, contentType string, body []byte, out any) error {
	return c.do(ctx, method, path, actor, role, contentType, bytes.NewReader(body), out)
}
func (c *checkClient) do(ctx context.Context, method, path, actor, role, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Actor", actor)
	req.Header.Set("X-Role", role)
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s %s 返回 %d: %s", method, path, res.StatusCode, string(payload))
	}
	if out != nil && len(payload) > 0 {
		if err = json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("解析 %s: %w", path, err)
		}
	}
	return nil
}
