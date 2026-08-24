package application

import (
	"context"
	"oralarchive/internal/domain"
)

type Role string

const (
	RoleCollector      Role = "collector"
	RoleReviewer       Role = "reviewer"
	RoleRepresentative Role = "representative"
	RoleArchivist      Role = "archivist"
)

type Principal struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}
type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
func principal(ctx context.Context) (Principal, error) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	if !ok || p.Name == "" {
		return p, domain.NewRuleError(domain.CodeForbidden, "actor", "请求缺少操作人")
	}
	return p, nil
}
func authorize(ctx context.Context, roles ...Role) (Principal, error) {
	p, err := principal(ctx)
	if err != nil {
		return p, err
	}
	for _, r := range roles {
		if p.Role == r {
			return p, nil
		}
	}
	return p, domain.NewRuleError(domain.CodeForbidden, "role", "角色 %s 无权执行此操作", p.Role)
}
