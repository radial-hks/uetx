package domain

// GenerateRequest is the input contract for material generation.
type GenerateRequest struct {
	HLSL         string              `json:"hlsl"`
	MaterialName string              `json:"materialName,omitempty"`
	OutputType   OutputType          `json:"outputType,omitempty"`
	Inputs       []NodeInput         `json:"inputs,omitempty"`
	Routing      []MaterialOutputSlot `json:"routing,omitempty"`
	Seed         int64               `json:"seed,omitempty"`
}

// GenerateResponse is the output contract for material generation.
type GenerateResponse struct {
	OK                  bool         `json:"ok"`
	T3D                 string       `json:"t3d,omitempty"`
	InferredInputs      []NodeInput  `json:"inferredInputs,omitempty"`
	EffectiveOutputType OutputType   `json:"effectiveOutputType,omitempty"`
	EffectiveRouting    []string     `json:"effectiveRouting,omitempty"`
	MaterialName        string       `json:"materialName,omitempty"`
	Stats               *Stats       `json:"stats,omitempty"`
	Warnings            []Diagnostic `json:"warnings,omitempty"`
	Errors              []Diagnostic `json:"errors,omitempty"`
}

// Stats summarises the generated graph.
type Stats struct {
	NodeCount  int  `json:"nodeCount"`
	EdgeCount  int  `json:"edgeCount"`
	HasBreakOut bool `json:"hasBreakOut"`
}

// Diagnostic is a structured warning or error.
type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}
