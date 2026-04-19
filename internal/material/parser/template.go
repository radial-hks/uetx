package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/radial/uetx/internal/domain"
)

// ParseResult holds the parsed template data.
type ParseResult struct {
	Inputs        []domain.NodeInput
	OutputType    domain.OutputType
	HasOutputType bool
}

var (
	commentBlockRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	pinRe          = regexp.MustCompile(`(?i)Pin\s+\d+\s+Name:\s*\[([^\]]+)\]\s*\|\s*Type suggestion:\s*([^\n|(]+)(?:\s*\(Default:\s*([^)]+)\))?`)
	outputTypeRe   = regexp.MustCompile(`(?i)Output Type\s*(?:\(输出类型\))?:\s*\[(.*?)\]`)
)

// ParseTemplate extracts NodeInputs and OutputType from the first comment block.
func ParseTemplate(hlsl string) (ParseResult, []domain.Diagnostic) {
	var result ParseResult
	var diags []domain.Diagnostic

	if strings.TrimSpace(hlsl) == "" {
		diags = append(diags, domain.Diagnostic{
			Code:    "E001",
			Message: "HLSL input is empty",
		})
		return result, diags
	}

	block := commentBlockRe.FindString(hlsl)
	if block == "" {
		diags = append(diags, domain.Diagnostic{
			Code:    "E002",
			Message: "no template comment block found (/* ... */)",
			Hint:    "wrap template metadata in a /* ... */ block at the top of your HLSL",
		})
		return result, diags
	}

	// Parse pins
	matches := pinRe.FindAllStringSubmatch(block, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		typeSuggestion := strings.TrimSpace(m[2])
		defaultVal := ""
		if len(m) > 3 {
			defaultVal = strings.TrimSpace(m[3])
		}

		if seen[name] {
			diags = append(diags, domain.Diagnostic{
				Code:    "E011",
				Message: "duplicate pin name: " + name,
			})
			continue
		}
		seen[name] = true

		result.Inputs = append(result.Inputs, domain.NodeInput{
			Name:         name,
			Type:         InferParamType(typeSuggestion),
			DefaultValue: defaultVal,
		})
	}

	if len(result.Inputs) == 0 {
		diags = append(diags, domain.Diagnostic{
			Code:    "W002",
			Message: "template parsed 0 input pins",
		})
	}

	// Parse output type
	otMatch := outputTypeRe.FindStringSubmatch(block)
	if otMatch != nil {
		ot, err := normalizeOutputType(otMatch[1])
		if err != nil {
			diags = append(diags, domain.Diagnostic{
				Code:    "E020",
				Message: "unrecognized output type: " + otMatch[1],
				Hint:    "use CMOT_Float1, CMOT_Float2, CMOT_Float3, or CMOT_Float4",
			})
		} else {
			result.OutputType = ot
			result.HasOutputType = true
		}
	}

	return result, diags
}

// normalizeOutputType applies the normalization rules from CORE_KNOWLEDGE §11.
func normalizeOutputType(raw string) (domain.OutputType, error) {
	s := strings.TrimSpace(raw)
	// "Float 4" → "Float4"
	s = regexp.MustCompile(`(?i)Float\s+(\d)`).ReplaceAllString(s, "Float$1")
	// All whitespace → underscore
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "_")

	switch s {
	case "CMOT_Float1":
		return domain.CMOTFloat1, nil
	case "CMOT_Float2":
		return domain.CMOTFloat2, nil
	case "CMOT_Float3":
		return domain.CMOTFloat3, nil
	case "CMOT_Float4":
		return domain.CMOTFloat4, nil
	default:
		return "", fmt.Errorf("invalid output type: %s", s)
	}
}
