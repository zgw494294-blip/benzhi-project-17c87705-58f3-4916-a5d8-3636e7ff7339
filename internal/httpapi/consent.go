package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
)

func (s *Server) HandleLockConsent(w http.ResponseWriter, r *http.Request) {
	var cmd application.LockConsentCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.service.LockConsent(r.Context(), r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
