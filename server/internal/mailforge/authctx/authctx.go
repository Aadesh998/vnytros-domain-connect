package authctx

import "context"

type ctxKey struct{}

type Principal struct {
	UserID   uint
	Email    string
	UserType string
}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok && p != nil
}

func UserID(ctx context.Context) uint {
	if p, ok := FromContext(ctx); ok {
		return p.UserID
	}
	return 0
}
