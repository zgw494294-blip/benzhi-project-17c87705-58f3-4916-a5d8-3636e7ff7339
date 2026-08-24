package application

import (
	"context"
	"fmt"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
	"strings"
)

const MaxMediaBytes int64 = 32 << 20

func (s *Service) UploadMedia(ctx context.Context, cmd UploadMediaCommand) (storage.MediaObject, error) {
	p, err := authorize(ctx, RoleCollector)
	if err != nil {
		return storage.MediaObject{}, err
	}
	if !strings.HasPrefix(cmd.ContentType, "audio/") {
		return storage.MediaObject{}, domain.NewRuleError(domain.CodeValidation, "contentType", "仅允许 audio/* 内容类型")
	}
	obj, err := s.store.PutMedia(ctx, cmd.CaseID, cmd.ContentType, cmd.Reader, MaxMediaBytes)
	if err != nil {
		return obj, err
	}
	c, err := s.store.GetCase(ctx, cmd.CaseID)
	if err != nil {
		return storage.MediaObject{}, mapStorage(err)
	}
	if c.Status != domain.StatusConsentLocked && c.Status != domain.StatusTranscribing {
		return storage.MediaObject{}, domain.NewRuleError(domain.CodeInvalidState, "status", "冻结同意后才能上传音频")
	}
	_ = s.store.AppendAudit(ctx, domain.AuditEvent{CaseID: cmd.CaseID, Action: "MEDIA_UPLOADED", Actor: p.Name, Detail: fmt.Sprintf("保存音频 %s（%d 字节）", obj.Digest, obj.Size), Version: c.Version, OccurredAt: s.now()})
	return obj, nil
}
