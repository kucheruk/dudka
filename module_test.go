package dudka_test

import (
	"testing"

	"dudka"
)

func TestModulePath(t *testing.T) {
	t.Parallel()
	if dudka.ModulePath != "dudka" {
		t.Fatalf("ModulePath = %q, want %q", dudka.ModulePath, "dudka")
	}
}
