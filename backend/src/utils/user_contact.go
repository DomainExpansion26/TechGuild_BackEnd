// src/utils/user_contact.go
package utils

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

var ErrUserContactMissing = errors.New("user contact missing from request context")

type userContactCtxKey struct{}

type userContact struct {
	Email string
	Phone string
}

// WithUserContactHuma is called by the auth middleware, right after token
// validation, with values taken from JWT claims.
func WithUserContactHuma(ctx huma.Context, email, phone string) huma.Context {
	return huma.WithValue(ctx, userContactCtxKey{}, userContact{Email: email, Phone: phone})
}

func GetUserContactFromHumaContext(ctx context.Context) (email, phone string, err error) {
	c, ok := ctx.Value(userContactCtxKey{}).(userContact)
	if !ok || c.Email == "" {
		return "", "", ErrUserContactMissing
	}
	return c.Email, c.Phone, nil
}
