package log

import (
	"os"
	"path"
	"time"

	"github.com/rs/zerolog"
)

var (
	logger zerolog.Logger
)

const (
	sasLogLevelEnvVar = "SAS_LOG_LEVEL"
	logLevelEnvVar    = "LOG_LEVEL"
)

func init() {
	initLogger()
}

func initLogger() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "timeStamp"

	level := zerolog.InfoLevel
	if envLevel, ok := os.LookupEnv(sasLogLevelEnvVar); ok {
		if parsedLevel, err := zerolog.ParseLevel(envLevel); err == nil {
			level = parsedLevel
		}
	} else if envLevel, ok := os.LookupEnv(logLevelEnvVar); ok {
		if parsedLevel, err := zerolog.ParseLevel(envLevel); err == nil {
			level = parsedLevel
		}
	}

	logger = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr}).Level(level).
		With().
		Timestamp().
		Caller().
		Str("source", getSource()).
		Logger()
}

func GetLogger() *zerolog.Logger {
	return &logger
}

func getSource() string {
	source, _ := os.Executable()
	source = path.Base(source)
	return source
}
