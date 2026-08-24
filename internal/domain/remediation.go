package domain

import "time"

type GateReport struct {
	Ready       bool                `json:"ready"`
	CheckedAt   time.Time           `json:"checkedAt"`
	IssueCount  int                 `json:"issueCount"`
	Issues      []GateIssue         `json:"issues"`
	Remediation []RemediationAction `json:"remediation"`
}

type RemediationAction struct {
	Code      string `json:"code"`
	Stage     string `json:"stage"`
	TargetID  string `json:"targetID,omitempty"`
	Action    string `json:"action"`
	OwnerRole string `json:"ownerRole"`
}

func BuildGateReport(c OralHistoryCase, consent *ConsentScope, segments []TranscriptSegment, confirmation *Confirmation, now time.Time) GateReport {
	issues := CheckGates(c, consent, segments, confirmation, now)
	report := GateReport{Ready: len(issues) == 0 && c.Status == StatusApproved, CheckedAt: now.UTC(), IssueCount: len(issues), Issues: issues}
	for _, issue := range issues {
		action := RemediationAction{Code: issue.Code, TargetID: issue.SegmentID}
		switch issue.Code {
		case "MISSING_CONSENT":
			action.Stage, action.Action, action.OwnerRole = "CONSENT", "登记并冻结知情同意范围", "representative"
		case "EMBARGO_ACTIVE":
			action.Stage, action.Action, action.OwnerRole = "CONSENT", "等待封存期结束或重新取得授权", "representative"
		case "MISSING_TRANSCRIPT":
			action.Stage, action.Action, action.OwnerRole = "TRANSCRIPT", "上传音频并录入至少一个转录片段", "collector"
		case "PENDING_REDACTION":
			action.Stage, action.Action, action.OwnerRole = "REVIEW", "对指定敏感片段作出保留、替换或删除裁定", "reviewer"
		case "OVERLAPPING_SEGMENT":
			action.Stage, action.Action, action.OwnerRole = "TRANSCRIPT", "调整指定片段时间码以消除重叠", "collector"
		case "MISSING_CONFIRMATION":
			action.Stage, action.Action, action.OwnerRole = "CONFIRMATION", "生成预览并取得受访者确认", "representative"
		default:
			action.Stage, action.Action, action.OwnerRole = "RELEASE", "复核发布门禁", "archivist"
		}
		report.Remediation = append(report.Remediation, action)
	}
	return report
}
