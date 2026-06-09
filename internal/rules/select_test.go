package rules_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/rules"
)

func TestSelectReturnsAllByDefault(t *testing.T) {
	t.Parallel()

	all := rules.All()
	selected, err := rules.Select(all, nil, nil)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if len(selected) != len(all) {
		t.Fatalf("len(selected) = %d, want %d", len(selected), len(all))
	}
}

func TestSelectOnly(t *testing.T) {
	t.Parallel()

	all := rules.All()
	selected, err := rules.Select(all, []string{"pod/crashloop", "lease/stale"}, nil)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].ID() != "pod/crashloop" {
		t.Errorf("selected[0].ID() = %q, want pod/crashloop", selected[0].ID())
	}
	if selected[1].ID() != "lease/stale" {
		t.Errorf("selected[1].ID() = %q, want lease/stale", selected[1].ID())
	}
}

func TestSelectDisable(t *testing.T) {
	t.Parallel()

	all := rules.All()
	selected, err := rules.Select(all, nil, []string{"pod/crashloop"})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if len(selected) != len(all)-1 {
		t.Fatalf("len(selected) = %d, want %d", len(selected), len(all)-1)
	}
	for _, rule := range selected {
		if rule.ID() == "pod/crashloop" {
			t.Fatal("pod/crashloop should have been disabled")
		}
	}
}

func TestSelectUnknownID(t *testing.T) {
	t.Parallel()

	all := rules.All()

	_, err := rules.Select(all, []string{"pod/crashloop", "does/not-exist"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown only rule ID")
	}

	_, err = rules.Select(all, nil, []string{"does/not-exist"})
	if err == nil {
		t.Fatal("expected error for unknown disable rule ID")
	}
}
