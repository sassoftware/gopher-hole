package metrics

import "testing"

func TestString(t *testing.T) {
	k := key{Name: "name"}
	if k.String() != "context value name" {
		t.Errorf("Stringer for context key returns incorrect value")
	}
}
