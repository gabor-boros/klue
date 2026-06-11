package diagnose

import (
	"fmt"
	"sort"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// Engine evaluates a set of rules against a resource graph to produce a
// diagnosis. It is deterministic: the same inputs always yield the same output.
type Engine struct {
	rulesByKind map[resource.Kind][]Rule
	anyRules    []Rule
}

// NewEngine creates an Engine from the given rules.
func NewEngine(rules ...Rule) *Engine {
	engine := &Engine{
		rulesByKind: make(map[resource.Kind][]Rule),
		anyRules:    make([]Rule, 0),
	}

	for i := range rules {
		rule := rules[i]
		applies := rule.AppliesTo()
		appliesToAny := false
		for _, kind := range applies {
			if kind == resource.KindAny {
				appliesToAny = true
				break
			}
		}

		if appliesToAny {
			engine.anyRules = append(engine.anyRules, rule)
			continue
		}

		for _, kind := range applies {
			engine.rulesByKind[kind] = append(engine.rulesByKind[kind], rule)
		}
	}

	return engine
}

// Diagnose locates the target in the graph, traverses its neighborhood in
// breadth-first layers, runs applicable rules, and assembles a diagnosis.
// Traversal stops after the first layer that produces findings or when the
// reachable graph is exhausted.
func (e *Engine) Diagnose(rctx RuleContext, target resource.Reference) Diagnosis {
	diagnosis := Diagnosis{Target: target}

	if rctx.Graph == nil {
		diagnosis.Summary = "no resource graph available"
		return diagnosis
	}

	targetNode, ok := rctx.Graph.FindByRef(target)
	if !ok {
		diagnosis.Summary = fmt.Sprintf("%s not found", target.Display())
		return diagnosis
	}

	visited, findings, exhausted := e.diagnoseUntilFound(rctx, targetNode, rctx.Options.MaxDepth)
	chainNodes := visited

	if len(findings) == 0 && exhausted && rctx.Options.ScanNamespaceRemainder {
		visitedKeys := nodeKeys(visited)
		fallbackFindings, fallbackNodes := e.scanNamespaceRemainder(rctx, targetNode, visitedKeys)
		findings = append(findings, fallbackFindings...)
		chainNodes = append(chainNodes, fallbackNodes...)
	}

	correlated := annotateCorroboration(findings)
	var suppressed int
	findings, suppressed = suppressRedundantBuiltinFindings(findings)
	sortFindings(findings)

	diagnosis.Findings = findings
	diagnosis.Chain = buildChain(chainNodes, findings)
	diagnosis.Suggestions = aggregateSuggestions(findings)
	if rctx.Options.Debug {
		diagnosis.Debug = &DebugInfo{
			EventWindow:        rctx.Options.EventWindow.String(),
			CorrelatedFindings: correlated,
			SuppressedFindings: suppressed,
		}
	}

	switch {
	case len(findings) > 0:
		diagnosis.RootCause = &findings[0]
		diagnosis.Summary = findings[0].Title
	case exhausted:
		diagnosis.Summary = fmt.Sprintf("%s appears healthy", target.Display())
	default:
		diagnosis.Summary = fmt.Sprintf("no issues found within %d graph hop(s) from %s", rctx.Options.MaxDepth, target.Display())
	}

	return diagnosis
}

// evaluateNode runs every rule that applies to the node's kind.
func (e *Engine) evaluateNode(rctx RuleContext, node *graph.Node) []Finding {
	applicable := e.rulesByKind[node.Ref.Kind]
	findings := make([]Finding, 0, len(applicable)+len(e.anyRules))

	for _, rule := range applicable {
		findings = append(findings, rule.Evaluate(rctx, node)...)
	}
	for _, rule := range e.anyRules {
		findings = append(findings, rule.Evaluate(rctx, node)...)
	}

	return findings
}

// diagnoseUntilFound traverses the graph in BFS layers from start, evaluating
// each layer as it is visited. It stops as soon as a layer produces findings.
// A non-positive maxDepth means unlimited traversal.
func (e *Engine) diagnoseUntilFound(rctx RuleContext, start graph.Node, maxDepth int) ([]graph.Node, []Finding, bool) {
	type item struct {
		node  graph.Node
		depth int
	}

	unlimitedDepth := maxDepth <= 0
	visited := map[string]bool{start.Ref.Key(): true}
	order := make([]graph.Node, 0, 1)
	frontier := []item{{node: start, depth: 0}}

	for len(frontier) > 0 {
		layerFindings := make([]Finding, 0, len(frontier))
		for i := range frontier {
			current := frontier[i]
			order = append(order, current.node)
			layerFindings = append(layerFindings, e.evaluateNode(rctx, &current.node)...)
		}
		if len(layerFindings) > 0 {
			return order, layerFindings, false
		}

		next := make([]item, 0)
		truncatedByDepth := false

		for i := range frontier {
			current := frontier[i]
			related := rctx.Graph.GetRelatedNodes(current.node)
			sort.Slice(related, func(i, j int) bool {
				return related[i].Ref.Key() < related[j].Ref.Key()
			})

			if !unlimitedDepth && current.depth >= maxDepth {
				for _, node := range related {
					if !visited[node.Ref.Key()] {
						truncatedByDepth = true
						break
					}
				}
				continue
			}

			for _, node := range related {
				key := node.Ref.Key()
				if visited[key] {
					continue
				}
				visited[key] = true
				next = append(next, item{node: node, depth: current.depth + 1})
			}
		}

		if len(next) == 0 {
			return order, nil, !truncatedByDepth
		}
		frontier = next
	}

	return order, nil, true
}

// scanNamespaceRemainder evaluates nodes that were not reached during
// traversal but are still in the target namespace.
func (e *Engine) scanNamespaceRemainder(rctx RuleContext, targetNode graph.Node, visitedKeys map[string]struct{}) ([]Finding, []graph.Node) {
	allNodes := rctx.Graph.GetNodes()
	findings := make([]Finding, 0)
	nodesWithFindings := make([]graph.Node, 0)
	targetNamespace := targetNode.Ref.Namespace

	for i := range allNodes {
		node := allNodes[i]
		key := node.Ref.Key()
		if _, alreadyVisited := visitedKeys[key]; alreadyVisited {
			continue
		}
		visitedKeys[key] = struct{}{}

		if node.Ref.Namespace != targetNamespace {
			continue
		}
		if node.IsPlaceholder() {
			continue
		}

		nodeFindings := e.evaluateNode(rctx, &node)
		if len(nodeFindings) == 0 {
			continue
		}

		findings = append(findings, nodeFindings...)
		nodesWithFindings = append(nodesWithFindings, node)
	}

	return findings, nodesWithFindings
}

func nodeKeys(nodes []graph.Node) map[string]struct{} {
	keys := make(map[string]struct{}, len(nodes))
	for i := range nodes {
		keys[nodes[i].Ref.Key()] = struct{}{}
	}
	return keys
}

// severityRank orders severities from most to least urgent.
func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 4
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// sortFindings orders findings by severity, then confidence, then ID so the
// most likely root cause is first and the ordering is deterministic.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		ri, rj := severityRank(findings[i].Severity), severityRank(findings[j].Severity)
		if ri != rj {
			return ri > rj
		}
		if findings[i].Corroboration != findings[j].Corroboration {
			return findings[i].Corroboration > findings[j].Corroboration
		}
		if findings[i].Confidence != findings[j].Confidence {
			return findings[i].Confidence > findings[j].Confidence
		}
		return findings[i].ID < findings[j].ID
	})
}

func annotateCorroboration(findings []Finding) int {
	correlatedFindings := 0
	for i := range findings {
		evidenceTypes := make(map[EvidenceType]struct{}, len(findings[i].Evidence))
		for _, ev := range findings[i].Evidence {
			evidenceTypes[ev.Type] = struct{}{}
		}

		score := len(evidenceTypes)
		_, hasEvent := evidenceTypes[EvidenceEvent]
		_, hasLog := evidenceTypes[EvidenceLog]
		_, hasStatus := evidenceTypes[EvidenceStatus]
		_, hasCondition := evidenceTypes[EvidenceCondition]

		if hasEvent && hasLog {
			score += 2
		}
		if hasEvent && hasStatus {
			score++
		}
		if hasCondition && hasLog {
			score++
		}

		findings[i].Corroboration = score
		if score > 1 {
			correlatedFindings++
		}
	}
	return correlatedFindings
}

func suppressRedundantBuiltinFindings(findings []Finding) ([]Finding, int) {
	if len(findings) == 0 {
		return findings, 0
	}

	typedEventEvidence := make(map[string]map[string]struct{})
	typedLogEvidence := make(map[string]map[string]struct{})
	for _, finding := range findings {
		if finding.ID == "builtin/warning-events" || finding.ID == "builtin/log-signal" {
			continue
		}
		key := finding.Resource.Key()
		for _, ev := range finding.Evidence {
			switch ev.Type {
			case EvidenceEvent:
				if _, ok := typedEventEvidence[key]; !ok {
					typedEventEvidence[key] = make(map[string]struct{})
				}
				typedEventEvidence[key][evidenceSignature(ev)] = struct{}{}
			case EvidenceLog:
				if _, ok := typedLogEvidence[key]; !ok {
					typedLogEvidence[key] = make(map[string]struct{})
				}
				typedLogEvidence[key][evidenceSignature(ev)] = struct{}{}
			}
		}
	}

	out := make([]Finding, 0, len(findings))
	suppressed := 0
	for _, finding := range findings {
		resourceKey := finding.Resource.Key()
		if finding.ID == "builtin/warning-events" {
			if signatures, ok := typedEventEvidence[resourceKey]; ok {
				if hasMatchingEvidenceSignature(finding.Evidence, EvidenceEvent, signatures) {
					suppressed++
					continue
				}
			}
		}
		if finding.ID == "builtin/log-signal" {
			if signatures, ok := typedLogEvidence[resourceKey]; ok {
				if hasMatchingEvidenceSignature(finding.Evidence, EvidenceLog, signatures) {
					suppressed++
					continue
				}
			}
		}
		out = append(out, finding)
	}
	return out, suppressed
}

func hasMatchingEvidenceSignature(items []Evidence, evidenceType EvidenceType, signatures map[string]struct{}) bool {
	for _, ev := range items {
		if ev.Type != evidenceType {
			continue
		}
		if _, ok := signatures[evidenceSignature(ev)]; ok {
			return true
		}
	}
	return false
}

func evidenceSignature(ev Evidence) string {
	return ev.Raw + "|" + ev.Message
}

// buildChain converts the traversal order into chain steps, attaching the first
// finding explanation recorded for each resource.
func buildChain(nodes []graph.Node, findings []Finding) []ChainStep {
	explanations := make(map[string]string, len(findings))
	for i := range findings {
		key := findings[i].Resource.Key()
		if _, ok := explanations[key]; !ok {
			explanations[key] = findings[i].Explanation
		}
	}

	steps := make([]ChainStep, 0, len(nodes))
	for i := range nodes {
		steps = append(steps, ChainStep{
			Resource:    nodes[i].Ref,
			State:       nodes[i].Status,
			Explanation: explanations[nodes[i].Ref.Key()],
		})
	}

	return steps
}

// aggregateSuggestions collects suggestions from all findings, de-duplicating
// by title and command while preserving order.
func aggregateSuggestions(findings []Finding) []Suggestion {
	seen := make(map[string]bool)
	var suggestions []Suggestion

	for i := range findings {
		for _, suggestion := range findings[i].Suggestions {
			key := suggestion.Title + "|" + suggestion.Command
			if seen[key] {
				continue
			}
			seen[key] = true
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}
