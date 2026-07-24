package security

import "context"

type authInfoKey struct{}

func WithAuthInfo(ctx context.Context, auth AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey{}, auth)
}

func AuthInfoFromContext(ctx context.Context) (AuthInfo, bool) {
	value := ctx.Value(authInfoKey{})
	auth, ok := value.(AuthInfo)
	return auth, ok
}
