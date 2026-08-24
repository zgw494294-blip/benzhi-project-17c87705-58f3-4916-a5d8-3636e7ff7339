package domain

type Preview struct {
	CaseID           string           `json:"caseID"`
	Title            string           `json:"title"`
	IntervieweeAlias string           `json:"intervieweeAlias"`
	ConsentDigest    string           `json:"consentDigest"`
	Segments         []PreviewSegment `json:"segments"`
	Issues           []GateIssue      `json:"issues"`
}
type PreviewSegment struct {
	SegmentID      string         `json:"segmentID"`
	Speaker        string         `json:"speaker"`
	StartMillis    int64          `json:"startMillis"`
	EndMillis      int64          `json:"endMillis"`
	Text           string         `json:"text"`
	DecisionStatus DecisionStatus `json:"decisionStatus"`
}

func BuildPreview(c OralHistoryCase, consent *ConsentScope, segments []TranscriptSegment) Preview {
	p := Preview{CaseID: c.CaseID, Title: c.Title, IntervieweeAlias: c.IntervieweeAlias}
	if consent != nil {
		p.ConsentDigest = consent.Digest
	}
	for _, s := range segments {
		if s.DecisionStatus != DecisionRemove {
			p.Segments = append(p.Segments, PreviewSegment{s.SegmentID, s.Speaker, s.StartMillis, s.EndMillis, s.PublicText(), s.DecisionStatus})
		}
	}
	return p
}
