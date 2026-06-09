package diagnose_test

import (
	"fmt"
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

func BenchmarkEngineDiagnose(b *testing.B) {
	target := podRef("bench-root")

	builder := graph.NewBuilder()
	rootNode := graph.Node{Ref: target, Status: resource.StatusReady}
	builder.AddNode(rootNode)

	current := rootNode
	for i := 0; i < 2500; i++ {
		ref := resource.NewReference(
			resource.ReferenceKindConfigMap,
			"v1",
			"default",
			fmt.Sprintf("cfg-%04d", i),
			fmt.Sprintf("uid-cfg-%04d", i),
		)
		next := graph.Node{Ref: ref, Status: resource.StatusHealthy}
		builder.AddNode(next)
		builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: current, To: next})
		current = next
	}

	engine := diagnose.NewEngine(
		stubRule{id: "noop", kinds: []resource.Kind{resource.KindAny}},
	)
	rctx := diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diagnosis := engine.Diagnose(rctx, target)
		if diagnosis.RootCause != nil {
			b.Fatalf("RootCause = %+v, want nil", diagnosis.RootCause)
		}
	}
}
