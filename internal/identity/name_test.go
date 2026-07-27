package identity_test

import (
	"strings"
	"testing"

	"dudka/internal/identity"
)

func TestResolveDisplayNameFlagWins(t *testing.T) {
	t.Parallel()
	r := identity.NameResolver{
		Prompt:   func() (string, bool) { return "ИзПромпта", true },
		Hostname: func() (string, bool) { return "hostbox", true },
		Generate: func() string { return "Сонный+Барсук" },
	}
	got := r.Resolve("  Вася  ")
	if got != "Вася" {
		t.Fatalf("flag branch: got %q want Вася", got)
	}
}

func TestResolveDisplayNamePromptWhenNoFlag(t *testing.T) {
	t.Parallel()
	r := identity.NameResolver{
		Prompt:   func() (string, bool) { return "ИзПромпта", true },
		Hostname: func() (string, bool) { return "hostbox", true },
		Generate: func() string { return "Сонный+Барсук" },
	}
	got := r.Resolve("")
	if got != "ИзПромпта" {
		t.Fatalf("prompt branch: got %q want ИзПромпта", got)
	}
}

func TestResolveDisplayNameHostnameWhenPromptSkipped(t *testing.T) {
	t.Parallel()
	r := identity.NameResolver{
		Prompt:   func() (string, bool) { return "", false },
		Hostname: func() (string, bool) { return "kitchen-mac", true },
		Generate: func() string { return "Сонный+Барсук" },
	}
	got := r.Resolve("")
	if got != "kitchen-mac" {
		t.Fatalf("hostname branch: got %q want kitchen-mac", got)
	}
}

func TestResolveDisplayNameGeneratedWhenHostnameUnusable(t *testing.T) {
	t.Parallel()
	r := identity.NameResolver{
		Prompt:   func() (string, bool) { return "  ", true },
		Hostname: func() (string, bool) { return "localhost", true },
		Generate: func() string { return "Сонный+Барсук" },
	}
	got := r.Resolve("")
	if got != "Сонный+Барсук" {
		t.Fatalf("generated branch: got %q want Сонный+Барсук", got)
	}
}

func TestGenerateAdjectiveAnimalShape(t *testing.T) {
	t.Parallel()
	name := identity.GenerateAdjectiveAnimal(func(n int) int { return 0 })
	parts := strings.Split(name, "+")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("generated name %q want Прилагательное+Животное", name)
	}
}
