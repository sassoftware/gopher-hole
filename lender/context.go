package lender

import (
	"context"
	"errors"

	"github.com/sassoftware/gopher-hole/internal/key"
)

var lenderkey = key.Key{Name: "lender"}

// NewLenderContext adds the lender.Lender to the existing context
func NewLenderContext(ctx context.Context, lender *Lender) context.Context {
	return context.WithValue(ctx, lenderkey, lender)
}

// GetLenderFromContext retrieves the lender.Lender from the context
func GetLenderFromContext(ctx context.Context) (*Lender, error) {
	lender, ok := ctx.Value(lenderkey).(*Lender)
	if !ok {
		return nil, errors.New("lender not found in context")
	}
	return lender, nil
}
