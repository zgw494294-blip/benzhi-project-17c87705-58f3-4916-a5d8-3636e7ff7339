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
	// 校验案卷处于允许上传音频的状态后再持久化内容，避免被拒绝的上传留下
	// 媒体引用或上传审计（同意冻结前上传应在状态检查处被拒绝）。
	c, err := s.store.GetCase(ctx, cmd.CaseID)
	if err != nil {
		return storage.MediaObject{}, mapStorage(err)
	}
	if c.Status != domain.StatusConsentLocked && c.Status != domain.StatusTranscribing {
		return storage.MediaObject{}, domain.NewRuleError(domain.CodeInvalidState, "status", "冻结同意后才能上传音频")
	}
	obj, err := s.store.PutMedia(ctx, cmd.CaseID, cmd.ContentType, cmd.Reader, MaxMediaBytes)
	if err != nil {
		return obj, err
	}
	_ = s.store.AppendAudit(ctx, domain.AuditEvent{CaseID: cmd.CaseID, Action: "MEDIA_UPLOADED", Actor: p.Name, Detail: fmt.Sprintf("保存音频 %s（%d 字节）", obj.Digest, obj.Size), Version: c.Version, OccurredAt: s.now()})
	return obj, nil
}
