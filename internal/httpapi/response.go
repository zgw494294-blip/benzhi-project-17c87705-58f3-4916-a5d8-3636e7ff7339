package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"oralarchive/internal/application"
	"oralarchive/internal/domain"
)

type errorBody struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Field   string             `json:"field,omitempty"`
	Issues  []domain.GateIssue `json:"issues,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	body := apiError{Code: "INTERNAL_ERROR", Message: "服务处理请求失败"}
	var rule *domain.RuleError
	if errors.As(err, &rule) {
		body.Code = string(rule.Code)
		body.Message = rule.Message
		body.Field = rule.Field
		switch rule.Code {
		case domain.CodeValidation:
			status = http.StatusUnprocessableEntity
		case domain.CodeConflict:
			status = http.StatusConflict
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeForbidden:
			status = http.StatusForbidden
		case domain.CodeInvalidState, domain.CodeGateFailed, domain.CodeIntegrityFailed:
			status = http.StatusConflict
		}
	}
	var gates *application.GateError
	if errors.As(err, &gates) {
		status = http.StatusConflict
		body.Code = "GATE_FAILED"
		body.Message = gates.Error()
		body.Issues = gates.Issues
	}
	if errors.Is(err, contextDeadline()) {
		status = http.StatusGatewayTimeout
		body.Code = "REQUEST_TIMEOUT"
		body.Message = "请求处理超时"
	}
	writeJSON(w, status, errorBody{Error: body})
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.NewRuleError(domain.CodeValidation, "body", "JSON 请求无效：%v", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.NewRuleError(domain.CodeValidation, "body", "JSON 请求只能包含一个对象")
	}
	return nil
}
func contextDeadline() error { return context.DeadlineExceeded }
