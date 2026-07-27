package version_test

import (
	"testing"

	"dudka/internal/version"
)

func TestVersionNonEmpty(t *testing.T) {
	t.Parallel()
	if version.Version == "" {
		t.Fatal("Version must be non-empty")
	}
}
