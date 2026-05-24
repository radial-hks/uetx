package material

import (
	"fmt"
	"regexp"

	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/material/build"
	"github.com/radial/uetx/internal/material/parser"
	"github.com/radial/uetx/internal/material/serializer"
)

var materialNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// returnKeywordRe matches the HLSL `return` keyword as a standalone token.
var returnKeywordRe = regexp.MustCompile(`\breturn\b`)

// lineCommentRe strips // line comments; blockCommentRe strips /* ... */ blocks.
var lineCommentRe = regexp.MustCompile(`//[^\n]*`)
var blockCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)

// stripCommentsForScan removes comments so identifier/keyword scans don't trip
// on template metadata blocks.
func stripCommentsForScan(s string) string {
	s = blockCommentRe.ReplaceAllString(s, "")
	s = lineCommentRe.ReplaceAllString(s, "")
	return s
}

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
			if len(d.Code) > 0 && d.Code[0] == 'E' {
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

	// W001: HLSL missing `return` statement
	codeBody := stripCommentsForScan(req.HLSL)
	if !returnKeywordRe.MatchString(codeBody) {
		warnings = append(warnings, domain.Diagnostic{
			Code:    "W001",
			Message: "HLSL template has no `return` statement",
			Hint:    "Custom node expects an HLSL function body that returns a value",
		})
	}

	// W003: pin name not referenced as a variable in the HLSL code body
	for _, inp := range inputs {
		varRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(inp.Name) + `\b`)
		if !varRe.MatchString(codeBody) {
			warnings = append(warnings, domain.Diagnostic{
				Code:    "W003",
				Message: fmt.Sprintf("pin %q is not referenced in the HLSL code body", inp.Name),
			})
		}
	}

	// W004: useRGBMask=true on non-vector param type (silently dropped by build)
	for _, inp := range inputs {
		if inp.UseRGBMask && inp.Type != domain.ParamVector {
			warnings = append(warnings, domain.Diagnostic{
				Code:    "W004",
				Message: fmt.Sprintf("useRGBMask=true on non-vector input %q (type=%s); mask will be ignored", inp.Name, inp.Type),
			})
		}
	}

	// Routing
	routing := req.Routing
	if len(routing) == 0 {
		routing = domain.DefaultRouting(outputType)
		warnings = append(warnings, domain.Diagnostic{
			Code:    "W008",
			Message: "routing empty, using default routing",
		})
	}

	// W005: routing contains scalar slot but outputType is not Float4
	if outputType != domain.CMOTFloat4 {
		for _, slot := range routing {
			if _, ok := domain.ScalarSlots[slot]; ok {
				warnings = append(warnings, domain.Diagnostic{
					Code:    "W005",
					Message: fmt.Sprintf("routing contains scalar slot %q but outputType is %s (expected CMOT_Float4)", slot, outputType),
				})
				break
			}
		}
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
		if len(d.Code) > 0 && d.Code[0] == 'E' {
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
// CLI overrides in req (MaterialName, OutputType, Routing, Inputs) are applied on top
// of the parsed result so that callers can preview the effective configuration.
func Inspect(req domain.GenerateRequest) domain.GenerateResponse {
	parsed, diags := parser.ParseTemplate(req.HLSL)
	var warnings, errors []domain.Diagnostic
	for _, d := range diags {
		if len(d.Code) > 0 && d.Code[0] == 'E' {
			errors = append(errors, d)
		} else {
			warnings = append(warnings, d)
		}
	}

	// Effective output type: explicit override > parsed > default
	ot := domain.CMOTFloat3
	if req.OutputType != "" {
		ot = req.OutputType
	} else if parsed.HasOutputType {
		ot = parsed.OutputType
	}

	// Effective inputs: explicit override > parsed
	inputs := parsed.Inputs
	if len(req.Inputs) > 0 {
		inputs = req.Inputs
	}

	// Effective routing: explicit override > default for output type
	routing := domain.DefaultRouting(ot)
	if len(req.Routing) > 0 {
		routing = req.Routing
	}

	return domain.GenerateResponse{
		OK:                  len(errors) == 0,
		InferredInputs:      inputs,
		EffectiveOutputType: ot,
		EffectiveRouting:    routing,
		MaterialName:        req.MaterialName,
		Warnings:            warnings,
		Errors:              errors,
	}
}

// validOutputTypes is the set of recognised CMOT values.
var validOutputTypes = map[domain.OutputType]struct{}{
	domain.CMOTFloat1: {},
	domain.CMOTFloat2: {},
	domain.CMOTFloat3: {},
	domain.CMOTFloat4: {},
}

// validParamTypes is the set of recognised input parameter types.
var validParamTypes = map[domain.ParamType]struct{}{
	domain.ParamScalar:        {},
	domain.ParamVector:        {},
	domain.ParamTime:          {},
	domain.ParamUV:            {},
	domain.ParamWorldPosition: {},
}

// Validate checks the HLSL template and the fully-merged request, returning diagnostics.
// Config JSON merging should be handled at the cmd layer before calling this function.
func Validate(req domain.GenerateRequest) domain.GenerateResponse {
	resp := Inspect(req)

	// Material name validation
	if req.MaterialName != "" && !materialNameRe.MatchString(req.MaterialName) {
		resp.Errors = append(resp.Errors, domain.Diagnostic{
			Code:    "E110",
			Message: fmt.Sprintf("materialName %q contains illegal characters", req.MaterialName),
			Hint:    "use only [A-Za-z0-9_]",
		})
	}

	// Output type validation
	if req.OutputType != "" {
		if _, ok := validOutputTypes[req.OutputType]; !ok {
			resp.Errors = append(resp.Errors, domain.Diagnostic{
				Code:    "E101",
				Message: fmt.Sprintf("unknown outputType %q", req.OutputType),
				Hint:    "use CMOT_Float1, CMOT_Float2, CMOT_Float3, or CMOT_Float4",
			})
		}
	}

	// Routing slot validation
	if len(req.Routing) > 0 {
		valid := make(map[string]struct{}, len(domain.RootPinTable))
		for _, entry := range domain.RootPinTable {
			valid[entry.Name] = struct{}{}
		}
		for _, slot := range req.Routing {
			if _, ok := valid[slot]; !ok {
				resp.Errors = append(resp.Errors, domain.Diagnostic{
					Code:    "E102",
					Message: fmt.Sprintf("unknown routing slot %q", slot),
				})
			}
		}
	}

	// Input type validation
	for _, inp := range req.Inputs {
		if _, ok := validParamTypes[inp.Type]; !ok {
			resp.Errors = append(resp.Errors, domain.Diagnostic{
				Code:    "E103",
				Message: fmt.Sprintf("unknown input type %q for %q", inp.Type, inp.Name),
			})
		}
	}

	if len(resp.Errors) > 0 {
		resp.OK = false
	}

	return resp
}
