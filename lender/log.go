package lender

import (
	"github.com/rs/zerolog"
	"github.com/sassoftware/gopher-hole/internal/log"
)

var (
	logger *zerolog.Logger
)

func init() {
	logger = log.GetLogger()
}
