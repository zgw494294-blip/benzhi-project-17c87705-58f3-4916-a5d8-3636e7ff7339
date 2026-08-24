package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
)

func (s *Server) HandleListCases(w http.ResponseWriter, r *http.Request) {
	items, err := s.service.ListCases(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cases": items})
}
func (s *Server) HandleCreateCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateCaseCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.service.CreateCase(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
func (s *Server) HandleGetCase(w http.ResponseWriter, r *http.Request) {
	detail, err := s.service.CaseDetail(r.Context(), r.PathValue("caseID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
