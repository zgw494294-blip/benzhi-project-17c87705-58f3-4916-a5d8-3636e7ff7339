package application

import (
	"context"
	"fmt"
	"sort"

	"oralarchive/internal/domain"
)

func verificationFailure(code, field, segmentID, message string) domain.VerificationFailure {
	return domain.VerificationFailure{Code: code, Field: field, SegmentID: segmentID, Message: message}
}

// VerifyReleasePackage checks the immutable manifest and all content references
// without changing the package or any case data.
func (s *Service) VerifyReleasePackage(ctx context.Context, caseID string) (domain.ReleaseVerification, error) {
	if _, err := authorize(ctx, RoleArchivist); err != nil {
		return domain.ReleaseVerification{}, err
	}
	c, err := s.store.GetCase(ctx, caseID)
	if err != nil {
		return domain.ReleaseVerification{}, mapStorage(err)
	}
	if c.Status != domain.StatusReleased {
		return domain.ReleaseVerification{}, domain.NewRuleError(domain.CodeInvalidState, "status", "只有已签发案卷可核验发布包")
	}
	pkg, err := s.store.GetPackageByCase(ctx, caseID)
	if err != nil {
		return domain.ReleaseVerification{}, mapStorage(err)
	}
	report := domain.ReleaseVerification{CheckedAt: s.now().UTC(), Package: pkg, ManifestDigest: pkg.ManifestDigest}
	data, manifest, readErr := s.store.ReadManifestData(ctx, *pkg)
	if readErr != nil {
		report.Failures = append(report.Failures, verificationFailure("MANIFEST_READ_FAILED", "manifest", "", fmt.Sprintf("发布清单读取或解码失败：%v", readErr)))
		if manifest != nil {
			report.Manifest = manifest
		}
		return report, nil
	}
	report.Manifest = manifest
	if domain.BytesDigest(data) != pkg.ManifestDigest {
		report.Failures = append(report.Failures, verificationFailure("MANIFEST_DIGEST_MISMATCH", "manifestDigest", "", "发布清单摘要与发布包不一致"))
	}
	consent, consentErr := s.store.GetConsent(ctx, caseID)
	if consentErr != nil {
		report.Failures = append(report.Failures, verificationFailure("CONSENT_MISSING", "consentDigest", "", "案卷缺少知情同意记录"))
	} else if consent.Digest != pkg.ConsentDigest || manifest.ConsentDigest != consent.Digest {
		report.Failures = append(report.Failures, verificationFailure("CONSENT_DIGEST_MISMATCH", "consentDigest", "", "发布包、清单与当前同意摘要不一致"))
	}
	if manifest.CaseID != caseID || manifest.PackageID != pkg.PackageID {
		report.Failures = append(report.Failures, verificationFailure("PACKAGE_IDENTITY_MISMATCH", "packageID", "", "清单身份与访问案卷或发布包不一致"))
	}
	if !manifest.IssuedAt.Equal(pkg.IssuedAt) || manifest.IssuedBy != pkg.IssuedBy {
		report.Failures = append(report.Failures, verificationFailure("ISSUED_INFO_MISMATCH", "issuedAt", "", "清单签发信息与发布包不一致"))
	}
	segments, segErr := s.store.ListSegments(ctx, caseID)
	if segErr != nil {
		return domain.ReleaseVerification{}, segErr
	}
	current := make(map[string]domain.TranscriptSegment, len(segments))
	for _, seg := range segments {
		if seg.DecisionStatus != domain.DecisionRemove {
			current[seg.SegmentID] = seg
		}
	}
	manifestIDs := make(map[string]bool, len(manifest.Segments))
	for _, item := range manifest.Segments {
		if manifestIDs[item.SegmentID] {
			report.Failures = append(report.Failures, verificationFailure("DUPLICATE_MANIFEST_SEGMENT", "segments", item.SegmentID, "清单重复引用片段"))
			continue
		}
		manifestIDs[item.SegmentID] = true
		seg, ok := current[item.SegmentID]
		if !ok {
			report.Failures = append(report.Failures, verificationFailure("SEGMENT_REFERENCE_MISMATCH", "segments", item.SegmentID, "清单片段在当前持久化案卷中不存在"))
			continue
		}
		if seg.Speaker != item.Speaker || seg.StartMillis != item.StartMillis || seg.EndMillis != item.EndMillis || seg.PublicText() != item.Text || seg.Digest() != item.Digest {
			report.Failures = append(report.Failures, verificationFailure("SEGMENT_CONTENT_MISMATCH", "segments", item.SegmentID, "清单中的文本、时间码或摘要与当前片段不一致"))
		}
	}
	currentIDs := make([]string, 0, len(current))
	for id := range current {
		currentIDs = append(currentIDs, id)
	}
	sort.Strings(currentIDs)
	for _, id := range currentIDs {
		if !manifestIDs[id] {
			report.Failures = append(report.Failures, verificationFailure("SEGMENT_MISSING_FROM_MANIFEST", "segments", id, "当前公开片段未出现在清单中"))
		}
	}
	expectedDigests := make([]string, 0, len(current))
	for _, seg := range current {
		expectedDigests = append(expectedDigests, seg.Digest())
	}
	actualDigests := append([]string(nil), pkg.SegmentDigests...)
	sort.Strings(expectedDigests)
	sort.Strings(actualDigests)
	if len(expectedDigests) != len(actualDigests) {
		report.Failures = append(report.Failures, verificationFailure("SEGMENT_DIGEST_SET_MISMATCH", "segmentDigests", "", "发布包片段摘要集合数量不一致"))
	} else {
		for i := range expectedDigests {
			if expectedDigests[i] != actualDigests[i] {
				report.Failures = append(report.Failures, verificationFailure("SEGMENT_DIGEST_SET_MISMATCH", "segmentDigests", "", "发布包片段摘要集合不一致"))
				break
			}
		}
	}
	media, mediaErr := s.store.ListMedia(ctx, caseID)
	if mediaErr != nil {
		return domain.ReleaseVerification{}, mediaErr
	}
	for _, item := range media {
		ok, fileErr := s.store.MediaFileExists(ctx, item.Digest)
		if fileErr != nil || !ok {
			report.Failures = append(report.Failures, verificationFailure("MEDIA_OBJECT_MISSING", "mediaDigest", "", fmt.Sprintf("音频对象 %s 不存在或不是常规文件", item.Digest)))
		}
	}
	report.Valid = len(report.Failures) == 0
	return report, nil
}

func (s *Service) VerifyRelease(ctx context.Context, caseID string) (domain.ReleaseVerification, error) {
	return s.VerifyReleasePackage(ctx, caseID)
}
