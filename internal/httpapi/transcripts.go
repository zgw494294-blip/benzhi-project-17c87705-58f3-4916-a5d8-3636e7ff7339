package httpapi

import (
	"net/http"
	"oralarchive/internal/application"
	"strconv"
)

func (s *Server) HandleAddSegment(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddSegmentCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	seg, err := s.service.AddSegment(r.Context(), r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, seg)
}

func (s *Server) HandleBatchSegments(w http.ResponseWriter, r *http.Request) {
	var cmd application.ImportTranscriptBatchCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.service.ImportTranscriptBatch(r.Context(), r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) HandleReviseTimecode(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviseTimecodeCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.service.ReviseSegmentTimecode(r.Context(), r.PathValue("caseID"), r.PathValue("segmentID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) HandleReviewSegment(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviewCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, err)
		return
	}
	seg, err := s.service.ReviewSegment(r.Context(), r.PathValue("caseID"), r.PathValue("segmentID"), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, seg)
}
func (s *Server) HandleCompleteReview(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, err)
		return
	}
	c, err := s.service.CompleteReview(r.Context(), r.PathValue("caseID"), body.ExpectedVersion)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func expectedVersion(r *http.Request) (int64, error) {
	v, err := strconv.ParseInt(r.URL.Query().Get("expectedVersion"), 10, 64)
	if err != nil {
		return 0, domainValidation("expectedVersion", "expectedVersion 查询参数无效")
	}
	return v, nil
}
