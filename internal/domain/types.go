package domain

// ParamType identifies the kind of material parameter node.
type ParamType string

const (
	ParamScalar        ParamType = "scalar"
	ParamVector        ParamType = "vector"
	ParamTime          ParamType = "time"
	ParamUV            ParamType = "uv"
	ParamWorldPosition ParamType = "worldposition"
)

// OutputType corresponds to UE's CMOT enum for Custom node output width.
type OutputType string

const (
	CMOTFloat1 OutputType = "CMOT_Float1"
	CMOTFloat2 OutputType = "CMOT_Float2"
	CMOTFloat3 OutputType = "CMOT_Float3"
	CMOTFloat4 OutputType = "CMOT_Float4"
)

// MaterialOutputSlot names a material output pin on the Root node.
type MaterialOutputSlot = string

// PinDir is the direction of a pin.
type PinDir string

const (
	PinDirIn  PinDir = "In"
	PinDirOut PinDir = "Out"
)

// PinRef identifies a pin on a specific graph node (for LinkedTo).
type PinRef struct {
	GraphName string
	PinID     string
}

// Pin represents a single pin on a graph node.
type Pin struct {
	ID               string
	Name             string
	Dir              PinDir
	Category         string // "materialinput", "required", "optional", "mask", ""
	SubCategory      string
	FriendlyName     string // e.g. NSLOCTEXT(...)
	IsUObjectWrapper bool
	LinkedTo         []PinRef
}

// GraphNode represents a node in the material graph.
type GraphNode struct {
	GraphName string
	ExprName  string
	ExprClass string
	IsRoot    bool
	X, Y      int
	NodeGUID  string
	ExprGUID  string
	ExtraBody string // pre-formatted lines with 6-space indent and CRLF
	Pins      []*Pin
	CanRename bool
}

// Edge is a directed connection between two pins.
type Edge struct {
	From PinRef
	To   PinRef
}

// NodeInput describes a single input parameter parsed from the HLSL template.
type NodeInput struct {
	Name         string    `json:"name"`
	Type         ParamType `json:"type"`
	DefaultValue string    `json:"defaultValue,omitempty"`
	UseRGBMask   bool      `json:"useRGBMask,omitempty"`
}
