package domain

import (
	"sort"
)

// NormalizeSegments returns a stable media/time order for persistence and previews.
func NormalizeSegments(segments []TranscriptSegment) []TranscriptSegment {
	out := append([]TranscriptSegment(nil), segments...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MediaDigest != out[j].MediaDigest {
			return out[i].MediaDigest < out[j].MediaDigest
		}
		if out[i].StartMillis != out[j].StartMillis {
			return out[i].StartMillis < out[j].StartMillis
		}
		if out[i].EndMillis != out[j].EndMillis {
			return out[i].EndMillis < out[j].EndMillis
		}
		return out[i].SegmentID < out[j].SegmentID
	})
	return out
}

// ValidateNoOverlaps checks each audio digest independently. NewSegment validates
// non-negative, strictly increasing bounds for each individual segment.
func ValidateNoOverlaps(segments []TranscriptSegment) error {
	sorted := NormalizeSegments(segments)
	for i := 1; i < len(sorted); i++ {
		previous, current := sorted[i-1], sorted[i]
		if previous.MediaDigest == current.MediaDigest && previous.EndMillis > current.StartMillis {
			return NewRuleError(CodeValidation, "startMillis", "片段 %s 与 %s 在同一音频上时间段重叠", current.SegmentID, previous.SegmentID)
		}
	}
	return nil
}
