package plugin

import "testing"

// setJinVersionForTest overrides the jin version used by compat checks
// for the duration of the test and returns a restore function callers
// defer. Needed by tests that exercise the jin compat rejection path —
// the default "dev" is treated as "satisfies everything" so a real
// version has to be pinned to trip the check.
//
// The override goes through SetJinVersion (an atomic pointer swap), so
// concurrent reads from a still-running publish goroutine — even one
// belonging to a previous test — are race-free.
func setJinVersionForTest(t *testing.T, v string) func() {
	t.Helper()
	prev := SetJinVersion(v)
	return func() { SetJinVersion(prev) }
}
