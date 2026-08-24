package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
	"time"
)

type Server struct {
	service *application.Service
	mux     *http.ServeMux
}

func New(service *application.Service) *Server {
	s := &Server{service: service, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return s.withRequestContext(s.mux) }
func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.HandleHealth)
	s.mux.HandleFunc("GET /api/cases", s.HandleListCases)
	s.mux.HandleFunc("POST /api/cases", s.HandleCreateCase)
	s.mux.HandleFunc("GET /api/cases/{caseID}", s.HandleGetCase)
	s.mux.HandleFunc("POST /api/cases/{caseID}/consent", s.HandleLockConsent)
	s.mux.HandleFunc("POST /api/cases/{caseID}/media", s.HandleUploadMedia)
	s.mux.HandleFunc("POST /api/cases/{caseID}/segments", s.HandleAddSegment)
	s.mux.HandleFunc("POST /api/cases/{caseID}/segments/batch", s.HandleBatchSegments)
	s.mux.HandleFunc("PATCH /api/cases/{caseID}/segments/{segmentID}/timecode", s.HandleReviseTimecode)
	s.mux.HandleFunc("PATCH /api/cases/{caseID}/segments/{segmentID}/decision", s.HandleReviewSegment)
	s.mux.HandleFunc("POST /api/cases/{caseID}/review-complete", s.HandleCompleteReview)
	s.mux.HandleFunc("GET /api/cases/{caseID}/preview", s.HandlePreview)
	s.mux.HandleFunc("POST /api/cases/{caseID}/confirmation", s.HandleConfirmation)
	s.mux.HandleFunc("POST /api/cases/{caseID}/release", s.HandleRelease)
	s.mux.HandleFunc("GET /api/cases/{caseID}/release/verify", s.HandleReleaseVerification)
	s.mux.HandleFunc("GET /api/cases/{caseID}/release/verification", s.HandleReleaseVerification)
}
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}
