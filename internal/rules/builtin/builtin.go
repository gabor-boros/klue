// Package builtin contains generic diagnostic rules that apply to every
// Kubernetes built-in resource. They reason about patterns common to all
// objects (warning events, status conditions, deletion timestamps, owner
// references) rather than a single resource type, complementing the typed rules
// in the sibling packages.
package builtin

import (
	"time"

	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// defaultTerminatingGracePeriod is the age beyond which an object still carrying
// a deletion timestamp is considered stuck terminating.
const defaultTerminatingGracePeriod = 5 * time.Minute

// describeKind returns a lowercase kubectl-friendly token for a resource kind.
func describeKind(kind resource.Kind) string {
	return ruleutil.KubectlKind(kind)
}
