package graph

// EdgeKind is the kind of the edge.
type EdgeKind string

const (
	EdgeOwns             EdgeKind = "owns"
	EdgeSelectedBy       EdgeKind = "selected_by"
	EdgeReferences       EdgeKind = "references"
	EdgeProduces         EdgeKind = "produces"
	EdgeMounts           EdgeKind = "mounts"
	EdgeUsesSecret       EdgeKind = "uses_secret"
	EdgeUsesConfigMap    EdgeKind = "uses_configmap"
	EdgeBacksService     EdgeKind = "backs_service"
	EdgeExposesPort      EdgeKind = "exposes_port"
	EdgeHasEvent         EdgeKind = "has_event"
	EdgeScheduledOn      EdgeKind = "scheduled_on"
	EdgeUsesStorageClass EdgeKind = "uses_storageclass"
	EdgeScaleTarget      EdgeKind = "scale_target"
	EdgeProtects         EdgeKind = "protects"
	EdgeRoleRef          EdgeKind = "role_ref"
)

// EdgeData carries optional structured metadata describing how an edge was
// derived from the source object.
type EdgeData struct {
	// Path is the JSON field path on the source object that produced the edge.
	Path string
	// Role is the semantic role of the reference (for example "uses" or
	// "produces"). It is empty when the role is not meaningful.
	Role string
}

// Edge is an edge in the graph.
type Edge struct {
	Kind EdgeKind
	From Node
	To   Node

	Reason string
	Data   EdgeData
}
