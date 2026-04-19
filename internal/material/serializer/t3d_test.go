package serializer

import (
	"os"
	"strings"
	"testing"

	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/material/build"
	"github.com/radial/uetx/internal/material/parser"
)

func TestSerializeGraph_WaterLevel_CRLF(t *testing.T) {
	result := buildWaterLevel(t)
	output := SerializeGraph(result.Nodes, "M_WaterLevel")

	// Every \n must be preceded by \r
	for i, c := range output {
		if c == '\n' && (i == 0 || output[i-1] != '\r') {
			t.Fatalf("found bare LF at offset %d (not preceded by \\r)", i)
		}
	}
}

func TestSerializeGraph_WaterLevel_Idempotent(t *testing.T) {
	r1 := buildWaterLevelWithSeed(t, 42)
	r2 := buildWaterLevelWithSeed(t, 42)
	o1 := SerializeGraph(r1.Nodes, "M_WaterLevel")
	o2 := SerializeGraph(r2.Nodes, "M_WaterLevel")
	if o1 != o2 {
		t.Fatal("same seed did not produce identical output")
	}
}

func TestSerializeGraph_WaterLevel_StructuralCheck(t *testing.T) {
	result := buildWaterLevel(t)
	output := SerializeGraph(result.Nodes, "M_WaterLevel")

	// Read golden file
	golden, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.t3d")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	// Structural comparison: count nodes by type
	genNodes := countNodeTypes(output)
	goldenNodes := countNodeTypes(string(golden))

	// Both should have Root
	if genNodes["MaterialGraphNode_Root"] != 1 {
		t.Errorf("generated has %d Root nodes, want 1", genNodes["MaterialGraphNode_Root"])
	}
	if goldenNodes["MaterialGraphNode_Root"] != 1 {
		t.Errorf("golden has %d Root nodes, want 1", goldenNodes["MaterialGraphNode_Root"])
	}

	// Both should have 1 Custom node
	if genNodes["MaterialExpressionCustom"] != 1 {
		t.Errorf("generated has %d Custom nodes, want 1", genNodes["MaterialExpressionCustom"])
	}

	// Both should have BreakOut
	if genNodes["MaterialExpressionMaterialFunctionCall"] != 1 {
		t.Errorf("generated has %d BreakOut nodes, want 1", genNodes["MaterialExpressionMaterialFunctionCall"])
	}

	// Both have 7 parameter nodes (3 vector + 4 scalar)
	genParamCount := genNodes["MaterialExpressionVectorParameter"] + genNodes["MaterialExpressionScalarParameter"]
	if genParamCount != 7 {
		t.Errorf("generated has %d param nodes, want 7", genParamCount)
	}

	// Verify root pin names match
	genRootPins := extractRootPinNames(output)
	goldenRootPins := extractRootPinNames(string(golden))
	if len(genRootPins) != len(goldenRootPins) {
		t.Errorf("generated root pins = %d, golden root pins = %d", len(genRootPins), len(goldenRootPins))
	}
	for i := range genRootPins {
		if i < len(goldenRootPins) && genRootPins[i] != goldenRootPins[i] {
			t.Errorf("root pin[%d] = %q, golden = %q", i, genRootPins[i], goldenRootPins[i])
		}
	}
}

func TestSerializeGraph_NoBreakOut(t *testing.T) {
	guidFn := domain.NewSeededGUIDFunc(99)
	result, _ := build.BuildIR(build.BuildRequest{
		HLSL:       "/* Pin 0 Name: [X] | Type suggestion: Scalar */\nreturn X;",
		Inputs:     []domain.NodeInput{{Name: "X", Type: domain.ParamScalar}},
		OutputType: domain.CMOTFloat3,
	}, guidFn)

	output := SerializeGraph(result.Nodes, "M_Test")
	if strings.Contains(output, "BreakOutFloat4Components") {
		t.Error("output should not contain BreakOut for Float3")
	}
	if !strings.Contains(output, "MaterialExpressionCustom") {
		t.Error("output should contain Custom node")
	}
}

func buildWaterLevel(t *testing.T) *build.BuildResult {
	t.Helper()
	return buildWaterLevelWithSeed(t, 42)
}

func buildWaterLevelWithSeed(t *testing.T, seed int64) *build.BuildResult {
	t.Helper()
	hlsl, err := os.ReadFile("../../../testdata/material/golden/M_WaterLevel.hlsl")
	if err != nil {
		t.Fatalf("read HLSL: %v", err)
	}
	parsed, diags := parser.ParseTemplate(string(hlsl))
	for _, d := range diags {
		if d.Code[0] == 'E' {
			t.Fatalf("parse error: %s", d.Message)
		}
	}
	guidFn := domain.NewSeededGUIDFunc(seed)
	result, _ := build.BuildIR(build.BuildRequest{
		HLSL:         string(hlsl),
		Inputs:       parsed.Inputs,
		OutputType:   parsed.OutputType,
		MaterialName: "M_WaterLevel",
	}, guidFn)
	return result
}

// countNodeTypes counts occurrences of expression classes in T3D output.
func countNodeTypes(t3d string) map[string]int {
	counts := map[string]int{}
	for _, line := range strings.Split(t3d, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Begin Object Class=/Script/UnrealEd.MaterialGraphNode_Root") {
			counts["MaterialGraphNode_Root"]++
		} else if strings.HasPrefix(line, "Begin Object Class=/Script/Engine.") {
			// Extract expression class
			after := strings.TrimPrefix(line, "Begin Object Class=/Script/Engine.")
			parts := strings.Fields(after)
			if len(parts) > 0 {
				counts[parts[0]]++
			}
		}
	}
	return counts
}

// extractRootPinNames extracts PinName values from the root node's pins.
func extractRootPinNames(t3d string) []string {
	var names []string
	inRoot := false
	for _, line := range strings.Split(t3d, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "MaterialGraphNode_Root") && strings.HasPrefix(trimmed, "Begin Object") {
			inRoot = true
			continue
		}
		if inRoot && trimmed == "End Object" {
			break
		}
		if inRoot && strings.Contains(trimmed, "CustomProperties Pin") {
			// Extract PinName
			idx := strings.Index(trimmed, "PinName=\"")
			if idx >= 0 {
				rest := trimmed[idx+9:]
				end := strings.Index(rest, "\"")
				if end >= 0 {
					names = append(names, rest[:end])
				}
			}
		}
	}
	return names
}
