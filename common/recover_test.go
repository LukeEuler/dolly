package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverExitCodeConvertsPanicToFailure(t *testing.T) {
	exitCode := runWithRecoveredPanic()
	assert.Equal(t, 1, exitCode)
}

func TestRecoverExitCodeKeepsSuccessfulExitCode(t *testing.T) {
	exitCode := runWithoutPanic()
	assert.Equal(t, 0, exitCode)
}

func runWithRecoveredPanic() (exitCode int) {
	defer RecoverExitCode(&exitCode)
	panic("main failed")
}

func runWithoutPanic() (exitCode int) {
	defer RecoverExitCode(&exitCode)
	return 0
}
