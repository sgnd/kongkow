package cli

import (
	"os"
	"testing"
)

// Tests keep their data in a fresh KONGKOW_HOME each, which the agent
// guard stops when the tests themselves run under an AI agent. Agent
// detection is off here; guard_home_test.go turns it on where it tests it.
func TestMain(m *testing.M) {
	agentVars = nil
	os.Exit(m.Run())
}
