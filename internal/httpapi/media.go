package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
	"oralarchive/internal/domain"
	"strings"
)

func (s *Server) HandleUploadMedia(w http.ResponseWriter, r *http.Request) {
	contentType := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(contentType, "audio/") {
		writeError(w, domainValidation("contentType", "Content-Type 必须为 audio/*"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, application.MaxMediaBytes+1)
	obj, err := s.service.UploadMedia(r.Context(), application.UploadMediaCommand{CaseID: r.PathValue("caseID"), ContentType: contentType, Reader: r.Body})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}
func domainValidation(field, msg string) error {
	return &domain.RuleError{Code: domain.CodeValidation, Field: field, Message: msg}
}
