package key

import "testing"

func TestString(t *testing.T) {
	k := Key{Name: "name"}
	if k.String() != "context value name" {
		t.Errorf("Stringer for context key returns incorrect value")
	}
}
