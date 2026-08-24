package domain

import (
	"strings"
	"time"
)

type OralHistoryCase struct {
	CaseID           string     `json:"caseID"`
	Title            string     `json:"title"`
	IntervieweeAlias string     `json:"intervieweeAlias"`
	Collector        string     `json:"collector"`
	Status           CaseStatus `json:"status"`
	Version          int64      `json:"expectedVersion"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func NewCase(id, title, alias, collector string, now time.Time) (*OralHistoryCase, error) {
	if strings.TrimSpace(id) == "" {
		return nil, NewRuleError(CodeValidation, "caseID", "案卷编号不能为空")
	}
	if strings.TrimSpace(title) == "" {
		return nil, NewRuleError(CodeValidation, "title", "访谈主题不能为空")
	}
	if strings.TrimSpace(alias) == "" {
		return nil, NewRuleError(CodeValidation, "intervieweeAlias", "受访者代称不能为空")
	}
	if strings.TrimSpace(collector) == "" {
		return nil, NewRuleError(CodeValidation, "collector", "采集员不能为空")
	}
	now = now.UTC()
	return &OralHistoryCase{CaseID: id, Title: strings.TrimSpace(title), IntervieweeAlias: strings.TrimSpace(alias), Collector: strings.TrimSpace(collector), Status: StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (c *OralHistoryCase) EnsureMutable() error {
	if c.Status == StatusReleased {
		return NewRuleError(CodeInvalidState, "status", "已发布案卷不可修改")
	}
	return nil
}

func (c *OralHistoryCase) Transition(to CaseStatus, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if !CanTransition(c.Status, to) {
		return NewRuleError(CodeInvalidState, "status", "案卷不能从 %s 进入 %s", c.Status, to)
	}
	c.Status, c.Version, c.UpdatedAt = to, c.Version+1, now.UTC()
	return nil
}
