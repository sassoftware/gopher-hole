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
	initLogger()

	assert.Equal(t, zerolog.InfoLevel, GetLogger().GetLevel())
}

func TestInit_SasLogLevelEnvVar(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "debug")
	defer func() { os.Unsetenv(sasLogLevelEnvVar) }()

	initLogger()

	assert.Equal(t, zerolog.DebugLevel, GetLogger().GetLevel())
}

func TestInit_LogLevelEnvVar(t *testing.T) {
	os.Unsetenv(sasLogLevelEnvVar)
	t.Setenv(logLevelEnvVar, "warn")
	defer func() { os.Unsetenv(logLevelEnvVar) }()

	initLogger()

	assert.Equal(t, zerolog.WarnLevel, GetLogger().GetLevel())
}

func TestInit_SasLogLevelTakesPrecedenceOverLogLevel(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "error")
	t.Setenv(logLevelEnvVar, "debug")

	initLogger()

	assert.Equal(t, zerolog.ErrorLevel, GetLogger().GetLevel())
}

func TestInit_InvalidLevelFallsBackToInfo(t *testing.T) {
	t.Setenv(sasLogLevelEnvVar, "notvalid")
	os.Unsetenv(logLevelEnvVar)

	initLogger()

	assert.Equal(t, zerolog.InfoLevel, GetLogger().GetLevel())
}

func TestGetSource_ReturnsNonEmptyString(t *testing.T) {
	src := getSource()
	assert.NotEmpty(t, src)
}
