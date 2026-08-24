package application

import (
	"errors"
	"oralarchive/internal/domain"
	"oralarchive/internal/storage"
	"strings"
	"time"
)

func requireVersion(actual, expected int64) error {
	if expected < 1 {
		return domain.NewRuleError(domain.CodeValidation, "expectedVersion", "expectedVersion 必须大于零")
	}
	if actual != expected {
		return domain.NewRuleError(domain.CodeConflict, "expectedVersion", "版本冲突：当前版本为 %d", actual)
	}
	return nil
}
func requireKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return domain.NewRuleError(domain.CodeValidation, "idempotencyKey", "idempotencyKey 不能为空")
	}
	return nil
}
func mapStorage(err error) error {
	if errors.Is(err, storage.ErrConflict) {
		return domain.NewRuleError(domain.CodeConflict, "expectedVersion", "案卷已被其他操作更新")
	}
	if errors.Is(err, storage.ErrNotFound) {
		return domain.NewRuleError(domain.CodeNotFound, "id", "记录不存在")
	}
	return err
}
func bump(c *domain.OralHistoryCase, now time.Time) { c.Version++; c.UpdatedAt = now.UTC() }
