package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
)

func (s *Server) HandlePreview(w http.ResponseWriter, r *http.Request) {
	p, err := s.service.Preview(r.Context(), r.PathValue("caseID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
func (s *Server) HandleConfirmation(w http.ResponseWriter, r *http.Request) {
	var cmd application.ConfirmCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.service.ConfirmPreview(r.Context(), r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s *Server) HandleRelease(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReleaseCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	pkg, err := s.service.Release(r.Context(), r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pkg)
}

func (s *Server) HandleReleaseVerification(w http.ResponseWriter, r *http.Request) {
	report, err := s.service.VerifyReleasePackage(r.Context(), r.PathValue("caseID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}
