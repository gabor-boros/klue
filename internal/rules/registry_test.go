package rules_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/rules"
)

func TestAllRulesHaveUniqueIDs(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for _, rule := range rules.All() {
		id := rule.ID()
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate rule ID %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestAllRulesHaveNonEmptyMetadata(t *testing.T) {
	t.Parallel()

	for _, rule := range rules.All() {
		if rule.ID() == "" {
			t.Errorf("rule %T has an empty ID", rule)
		}
		if rule.Description() == "" {
			t.Errorf("rule %q has an empty description", rule.ID())
		}
	}
}

func TestAllRulesDeclareValidKinds(t *testing.T) {
	t.Parallel()

	for _, rule := range rules.All() {
		kinds := rule.AppliesTo()
		if len(kinds) == 0 {
			t.Errorf("rule %q applies to no kinds", rule.ID())
		}
		for _, kind := range kinds {
			if kind == "" {
				t.Errorf("rule %q declares an empty kind", rule.ID())
			}
		}
	}
}
