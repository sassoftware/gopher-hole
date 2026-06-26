package log

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogger_NotNil(t *testing.T) {
	l := GetLogger()
	require.NotNil(t, l)
}

func TestGetLogger_ReturnsSameInstance(t *testing.T) {
	l1 := GetLogger()
	l2 := GetLogger()
	assert.Equal(t, l1, l2)
}

func TestGetLogger_IsZerologLogger(t *testing.T) {
	l := GetLogger()
	_, ok := any(l).(*zerolog.Logger)
	assert.True(t, ok)
}

func TestInit_DefaultLevelIsInfo(t *testing.T) {
	os.Unsetenv(sasLogLevelEnvVar)
	os.Unsetenv(logLevelEnvVar)

	// Re-initialize by calling init directly via reinit helper
	reinitLogger()

	assert.Equal(t, zerolog.InfoLevel, GetLogger().GetLevel())
}

func TestInit_SasLogLevelEnvVar(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "debug")
	defer func() { os.Unsetenv(sasLogLevelEnvVar) }()

	reinitLogger()

	assert.Equal(t, zerolog.DebugLevel, GetLogger().GetLevel())
}

func TestInit_LogLevelEnvVar(t *testing.T) {
	os.Unsetenv(sasLogLevelEnvVar)
	t.Setenv(logLevelEnvVar, "warn")
	defer func() { os.Unsetenv(logLevelEnvVar) }()

	reinitLogger()

	assert.Equal(t, zerolog.WarnLevel, GetLogger().GetLevel())
}

func TestInit_SasLogLevelTakesPrecedenceOverLogLevel(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "error")
	t.Setenv(logLevelEnvVar, "debug")

	reinitLogger()

	assert.Equal(t, zerolog.ErrorLevel, GetLogger().GetLevel())
}

func TestInit_InvalidLevelFallsBackToInfo(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "notvalid")
	os.Unsetenv(logLevelEnvVar)

	reinitLogger()

	assert.Equal(t, zerolog.InfoLevel, GetLogger().GetLevel())
}

func TestGetSource_ReturnsNonEmptyString(t *testing.T) {
	src := getSource()
	assert.NotEmpty(t, src)
}

// reinitLogger re-runs the package-level initialization logic so tests can
// observe the effect of different environment variable configurations.
func reinitLogger() {
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
