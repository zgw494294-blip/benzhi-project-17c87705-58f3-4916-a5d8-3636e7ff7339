package domain

import (
	"sort"
	"strings"
	"time"
)

type ConsentScope struct {
	ConsentID        string     `json:"consentID"`
	CaseID           string     `json:"caseID"`
	Version          int64      `json:"version"`
	AllowedAudiences []string   `json:"allowedAudiences"`
	AllowedPurposes  []string   `json:"allowedPurposes"`
	EmbargoUntil     *time.Time `json:"embargoUntil,omitempty"`
	WithdrawalTerms  string     `json:"withdrawalTerms"`
	ConfirmedBy      string     `json:"confirmedBy"`
	ConfirmedAt      time.Time  `json:"confirmedAt"`
	Digest           string     `json:"digest"`
}

func NewConsent(id, caseID string, version int64, audiences, purposes []string, embargo *time.Time, terms, confirmer string, now time.Time) (*ConsentScope, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(caseID) == "" {
		return nil, NewRuleError(CodeValidation, "consentID", "同意记录和案卷编号不能为空")
	}
	if version < 1 {
		return nil, NewRuleError(CodeValidation, "version", "同意版本必须大于零")
	}
	audiences = normalizedSet(audiences)
	purposes = normalizedSet(purposes)
	if len(audiences) == 0 {
		return nil, NewRuleError(CodeValidation, "allowedAudiences", "至少选择一个允许受众")
	}
	if len(purposes) == 0 {
		return nil, NewRuleError(CodeValidation, "allowedPurposes", "至少选择一个允许用途")
	}
	if strings.TrimSpace(terms) == "" {
		return nil, NewRuleError(CodeValidation, "withdrawalTerms", "必须记录撤回条件")
	}
	if strings.TrimSpace(confirmer) == "" {
		return nil, NewRuleError(CodeValidation, "confirmedBy", "确认人不能为空")
	}
	c := &ConsentScope{ConsentID: id, CaseID: caseID, Version: version, AllowedAudiences: audiences, AllowedPurposes: purposes, EmbargoUntil: embargo, WithdrawalTerms: strings.TrimSpace(terms), ConfirmedBy: strings.TrimSpace(confirmer), ConfirmedAt: now.UTC()}
	c.Digest = StableDigest(c.digestValue())
	return c, nil
}

func (c ConsentScope) digestValue() any {
	return struct {
		CaseID              string
		Version             int64
		Audiences, Purposes []string
		Embargo             string
		Terms, By           string
	}{c.CaseID, c.Version, c.AllowedAudiences, c.AllowedPurposes, timeString(c.EmbargoUntil), c.WithdrawalTerms, c.ConfirmedBy}
}
func normalizedSet(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
func timeString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
