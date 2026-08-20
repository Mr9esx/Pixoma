package condition

import "context"

type ctxKey int

const (
	ctxUserID ctxKey = iota + 1
	ctxCaseID
)

// WithUserID attaches the current user id to the evaluation context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

// UserIDFrom reads the user id attached by WithUserID.
func UserIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok && v != ""
}

// WithCaseID attaches the current case id to the evaluation context.
func WithCaseID(ctx context.Context, caseID string) context.Context {
	return context.WithValue(ctx, ctxCaseID, caseID)
}

// CaseIDFrom reads the case id attached by WithCaseID.
func CaseIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxCaseID).(string)
	return v, ok && v != ""
}
