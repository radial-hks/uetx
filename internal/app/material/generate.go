package material

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/material/build"
	"github.com/radial/uetx/internal/material/parser"
	"github.com/radial/uetx/internal/material/serializer"
)

var materialNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// Generate runs the full pipeline: parse → build → serialize.
func Generate(req domain.GenerateRequest) domain.GenerateResponse {
	var warnings []domain.Diagnostic
	var errors []domain.Diagnostic

	// Validate material name
	matName := req.MaterialName
	if matName == "" {
		matName = "M_CustomNode"
	}
	if !materialNameRe.MatchString(matName) {
		errors = append(errors, domain.Diagnostic{
			Code:    "E110",
			Message: fmt.Sprintf("materialName %q contains illegal characters", matName),
			Hint:    "use only [A-Za-z0-9_]",
		})
		return domain.GenerateResponse{OK: false, Errors: errors}
	}

	// Parse template (if inputs not provided)
	inputs := req.Inputs
	outputType := req.OutputType
	if len(inputs) == 0 || outputType == "" {
		parsed, diags := parser.ParseTemplate(req.HLSL)
		for _, d := range diags {
			if d.Code[0] == 'E' {
				errors = append(errors, d)
			} else {
				warnings = append(warnings, d)
			}
		}
		if len(errors) > 0 && len(inputs) == 0 {
			return domain.GenerateResponse{OK: false, Errors: errors, Warnings: warnings}
		}
		if len(inputs) == 0 {
			inputs = parsed.Inputs
		}
		if outputType == "" {
			if parsed.HasOutputType {
				outputType = parsed.OutputType
			} else {
				outputType = domain.CMOTFloat3
			}
		}
	}

	// Routing
	routing := req.Routing
	if len(routing) == 0 {
		routing = domain.DefaultRouting(outputType)
		warnings = append(warnings, domain.Diagnostic{
			Code:    "W005",
			Message: "routing empty, using default routing",
		})
	}

	// GUID function
	var guidFn domain.GUIDFunc
	if req.Seed != 0 {
		guidFn = domain.NewSeededGUIDFunc(req.Seed)
	} else {
		guidFn = domain.NewGUID
	}

	// Build IR
	result, bdiags := build.BuildIR(build.BuildRequest{
		HLSL:         req.HLSL,
		Inputs:       inputs,
		OutputType:   outputType,
		Routing:      routing,
		MaterialName: matName,
	}, guidFn)
	for _, d := range bdiags {
		if d.Code[0] == 'E' {
			errors = append(errors, d)
		} else {
			warnings = append(warnings, d)
		}
	}
	if len(errors) > 0 {
		return domain.GenerateResponse{OK: false, Errors: errors, Warnings: warnings}
	}

	// Serialize
	t3d := serializer.SerializeGraph(result.Nodes, matName)

	return domain.GenerateResponse{
		OK:                  true,
		T3D:                 t3d,
		InferredInputs:      inputs,
		EffectiveOutputType: outputType,
		EffectiveRouting:    routing,
		MaterialName:        matName,
		Stats: &domain.Stats{
			NodeCount:   len(result.Nodes),
			EdgeCount:   len(result.Edges),
			HasBreakOut: result.HasBreakOut,
		},
		Warnings: warnings,
	}
}

// Inspect parses the HLSL template and returns inferred metadata (no T3D generation).
func Inspect(hlsl string) domain.GenerateResponse {
	parsed, diags := parser.ParseTemplate(hlsl)
	var warnings, errors []domain.Diagnostic
	for _, d := range diags {
		if d.Code[0] == 'E' {
			errors = append(errors, d)
		} else {
			warnings = append(warnings, d)
		}
	}

	ot := domain.CMOTFloat3
	if parsed.HasOutputType {
		ot = parsed.OutputType
	}

	return domain.GenerateResponse{
		OK:                  len(errors) == 0,
		InferredInputs:      parsed.Inputs,
		EffectiveOutputType: ot,
		EffectiveRouting:    domain.DefaultRouting(ot),
		Warnings:            warnings,
		Errors:              errors,
	}
}

// Validate checks the HLSL template and optional JSON config, returning only diagnostics.
func Validate(hlsl string, configJSON []byte) domain.GenerateResponse {
	resp := Inspect(hlsl)

	if len(configJSON) > 0 {
		var cfg domain.GenerateRequest
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			resp.Errors = append(resp.Errors, domain.Diagnostic{
				Code:    "E100",
				Message: "JSON config parse failed: " + err.Error(),
			})
			resp.OK = false
		}
	}

	return resp
}
