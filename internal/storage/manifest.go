package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"oralarchive/internal/domain"
)

func (s *Store) ReadManifest(ctx context.Context, pkg domain.ReleasePackage) (*domain.Manifest, error) {
	_, manifest, err := s.ReadManifestData(ctx, pkg)
	return manifest, err
}

func (s *Store) ReadManifestData(ctx context.Context, pkg domain.ReleasePackage) ([]byte, *domain.Manifest, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	path := filepath.Join(s.manifests, digestFilename(pkg.ManifestDigest)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	if actual := domain.BytesDigest(data); actual != pkg.ManifestDigest {
		return data, nil, fmt.Errorf("发布清单摘要不一致: 期望 %s，实际 %s", pkg.ManifestDigest, actual)
	}
	var manifest domain.Manifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return data, nil, err
	}
	if manifest.PackageID != pkg.PackageID || manifest.CaseID != pkg.CaseID {
		return data, &manifest, fmt.Errorf("发布清单身份与发布包不一致")
	}
	return data, &manifest, nil
}
