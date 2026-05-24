package build

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/radial/uetx/internal/domain"
)

const friendly = `NSLOCTEXT("MaterialGraphNode", "Space", " ")`

// BuildRequest holds the inputs for IR construction.
type BuildRequest struct {
	HLSL         string
	Inputs       []domain.NodeInput
	OutputType   domain.OutputType
	Routing      []string
	MaterialName string
}

// BuildResult holds the constructed graph.
type BuildResult struct {
	Nodes       []*domain.GraphNode
	Edges       []domain.Edge
	HasBreakOut bool
}

// BuildIR constructs the material graph IR from a build request.
func BuildIR(req BuildRequest, guidFn domain.GUIDFunc) (*BuildResult, []domain.Diagnostic) {
	var diags []domain.Diagnostic
	nodeIndex := 0

	effectiveRouting := req.Routing
	if len(effectiveRouting) == 0 {
		effectiveRouting = domain.DefaultRouting(req.OutputType)
	}

	needsBO := NeedsBreakOut(req.OutputType, effectiveRouting)

	// 1. Root node
	root := buildRoot(guidFn)

	// 2. Custom node (placeholder — extraBody filled after params)
	customGraphName := fmt.Sprintf("MaterialGraphNode_%d", nodeIndex)
	nodeIndex++
	customExprGUID := guidFn()
	customExprName := fmt.Sprintf("MaterialExpressionCustom_%s", customExprGUID[:8])

	customPins := make([]*domain.Pin, 0, len(req.Inputs)+1)
	for _, inp := range req.Inputs {
		customPins = append(customPins, &domain.Pin{
			ID:               guidFn(),
			Name:             inp.Name,
			Dir:              domain.PinDirIn,
			Category:         "required",
			IsUObjectWrapper: true,
		})
	}
	customOutPin := &domain.Pin{
		ID:               guidFn(),
		Name:             "Output",
		Dir:              domain.PinDirOut,
		FriendlyName:     friendly,
		IsUObjectWrapper: true,
	}
	customPins = append(customPins, customOutPin)

	customNode := &domain.GraphNode{
		GraphName: customGraphName,
		ExprName:  customExprName,
		ExprClass: "MaterialExpressionCustom",
		X:         -432, Y: 528,
		NodeGUID: guidFn(),
		ExprGUID: customExprGUID,
		Pins:     customPins,
	}

	// 3. Parameter nodes
	paramNodes, inputsStrings, paramDiags := buildParams(req.Inputs, &nodeIndex, guidFn)
	diags = append(diags, paramDiags...)

	// Fill custom node extraBody
	customNode.ExtraBody = buildCustomExtraBody(req.HLSL, req.OutputType, inputsStrings)

	// Collect nodes: Root, Params, Custom (matching TS reference order)
	nodes := make([]*domain.GraphNode, 0, 2+len(paramNodes)+1)
	nodes = append(nodes, root)
	nodes = append(nodes, paramNodes...)
	nodes = append(nodes, customNode)

	// Edges: param output -> custom input
	var edges []domain.Edge
	for i, pn := range paramNodes {
		outPin := pn.Pins[0] // first pin is always the output
		edges = append(edges, domain.Edge{
			From: domain.PinRef{GraphName: pn.GraphName, PinID: outPin.ID},
			To:   domain.PinRef{GraphName: customGraphName, PinID: customPins[i].ID},
		})
	}

	// 4. BreakOut node
	var breakOutNode *domain.GraphNode
	breakOutPinMap := map[string]string{} // channel -> pinID
	if needsBO {
		breakOutNode, breakOutPinMap = buildBreakOut(customNode, &nodeIndex, guidFn)
		nodes = append(nodes, breakOutNode)

		// Edge: custom output -> breakout input
		edges = append(edges, domain.Edge{
			From: domain.PinRef{GraphName: customGraphName, PinID: customOutPin.ID},
			To:   domain.PinRef{GraphName: breakOutNode.GraphName, PinID: breakOutNode.Pins[0].ID},
		})
	}

	// 5. Routing edges
	breakOutChannels := []string{"R", "G", "B", "A"}
	scalarIdx := 0
	for _, slot := range effectiveRouting {
		rootPin := findRootPin(root, slot)
		if rootPin == nil {
			diags = append(diags, domain.Diagnostic{
				Code:    "W009",
				Message: fmt.Sprintf("routing slot %q does not match any root pin; dropped", slot),
			})
			continue
		}

		if needsBO && breakOutNode != nil {
			if _, isScalar := domain.ScalarSlots[slot]; isScalar {
				ch := breakOutChannels[scalarIdx%len(breakOutChannels)]
				scalarIdx++
				edges = append(edges, domain.Edge{
					From: domain.PinRef{GraphName: breakOutNode.GraphName, PinID: breakOutPinMap[ch]},
					To:   domain.PinRef{GraphName: root.GraphName, PinID: rootPin.ID},
				})
				continue
			}
		}
		edges = append(edges, domain.Edge{
			From: domain.PinRef{GraphName: customGraphName, PinID: customOutPin.ID},
			To:   domain.PinRef{GraphName: root.GraphName, PinID: rootPin.ID},
		})
	}

	// 6. Apply edges bidirectionally
	if err := applyEdges(edges, nodes); err != nil {
		diags = append(diags, domain.Diagnostic{
			Code:    "E099",
			Message: err.Error(),
		})
	}

	return &BuildResult{
		Nodes:       nodes,
		Edges:       edges,
		HasBreakOut: needsBO,
	}, diags
}

// buildRoot creates the Root material graph node with 30 input pins.
func buildRoot(guidFn domain.GUIDFunc) *domain.GraphNode {
	pins := make([]*domain.Pin, len(domain.RootPinTable))
	for i, entry := range domain.RootPinTable {
		pins[i] = &domain.Pin{
			ID:          guidFn(),
			Name:        entry.Name,
			Dir:         domain.PinDirIn,
			Category:    "materialinput",
			SubCategory: entry.Sub,
		}
	}
	return &domain.GraphNode{
		GraphName: "MaterialGraphNode_Root_0",
		IsRoot:    true,
		X:         352, Y: 528,
		NodeGUID: guidFn(),
		ExprGUID: guidFn(),
		Pins:     pins,
	}
}

// buildParams creates parameter nodes for each input.
func buildParams(inputs []domain.NodeInput, nodeIndex *int, guidFn domain.GUIDFunc) ([]*domain.GraphNode, []string, []domain.Diagnostic) {
	nodes := make([]*domain.GraphNode, 0, len(inputs))
	inputsStrings := make([]string, 0, len(inputs))
	var diags []domain.Diagnostic

	for i, inp := range inputs {
		graphName := fmt.Sprintf("MaterialGraphNode_%d", *nodeIndex)
		*nodeIndex++
		exprGUID := guidFn()
		x := -800
		y := i*150 - (len(inputs)*75) + 528

		var exprClass, extraBody string
		var pins []*domain.Pin
		outPinID := guidFn()

		switch inp.Type {
		case domain.ParamScalar:
			exprClass = "MaterialExpressionScalarParameter"
			val, err := parseScalarDefault(inp.DefaultValue)
			if err != nil {
				diags = append(diags, domain.Diagnostic{
					Code:    "W010",
					Message: fmt.Sprintf("input %q: %v; falling back to 0", inp.Name, err),
				})
			}
			extraBody = fmt.Sprintf("      DefaultValue=%s\r\n      ParameterName=\"%s\"", val, inp.Name)
			pins = []*domain.Pin{
				{ID: outPinID, Name: "Output", Dir: domain.PinDirOut, FriendlyName: friendly, IsUObjectWrapper: true},
			}

		case domain.ParamVector:
			exprClass = "MaterialExpressionVectorParameter"
			r, g, b, a, err := parseVectorDefault(inp.DefaultValue)
			if err != nil {
				diags = append(diags, domain.Diagnostic{
					Code:    "W010",
					Message: fmt.Sprintf("input %q: %v; falling back to defaults", inp.Name, err),
				})
			}
			extraBody = fmt.Sprintf("      DefaultValue=(R=%s,G=%s,B=%s,A=%s)\r\n      ParameterName=\"%s\"",
				r, g, b, a, inp.Name)
			pins = []*domain.Pin{
				{ID: outPinID, Name: "Output", Dir: domain.PinDirOut, Category: "mask", FriendlyName: friendly, IsUObjectWrapper: true},
				{ID: guidFn(), Name: "Output2", Dir: domain.PinDirOut, Category: "mask", SubCategory: "red", FriendlyName: friendly, IsUObjectWrapper: true},
				{ID: guidFn(), Name: "Output3", Dir: domain.PinDirOut, Category: "mask", SubCategory: "green", FriendlyName: friendly},
				{ID: guidFn(), Name: "Output4", Dir: domain.PinDirOut, Category: "mask", SubCategory: "blue", FriendlyName: friendly},
				{ID: guidFn(), Name: "Output5", Dir: domain.PinDirOut, Category: "mask", SubCategory: "alpha", FriendlyName: friendly},
			}

		case domain.ParamWorldPosition:
			exprClass = "MaterialExpressionWorldPosition"
			pins = []*domain.Pin{
				{ID: outPinID, Name: "Output", Dir: domain.PinDirOut, FriendlyName: friendly, IsUObjectWrapper: true},
			}

		case domain.ParamTime:
			exprClass = "MaterialExpressionTime"
			pins = []*domain.Pin{
				{ID: outPinID, Name: "Output", Dir: domain.PinDirOut, FriendlyName: friendly, IsUObjectWrapper: true},
			}

		case domain.ParamUV:
			exprClass = "MaterialExpressionTextureCoordinate"
			pins = []*domain.Pin{
				{ID: outPinID, Name: "Output", Dir: domain.PinDirOut, FriendlyName: friendly, IsUObjectWrapper: true},
			}
		}

		exprName := fmt.Sprintf("%s_%s", exprClass, exprGUID[:8])
		canRename := inp.Type == domain.ParamScalar || inp.Type == domain.ParamVector

		nodes = append(nodes, &domain.GraphNode{
			GraphName: graphName,
			ExprName:  exprName,
			ExprClass: exprClass,
			X:         x, Y: y,
			NodeGUID:  guidFn(),
			ExprGUID:  exprGUID,
			ExtraBody: extraBody,
			Pins:      pins,
			CanRename: canRename,
		})

		// Build Inputs(i) string for custom node
		maskStr := ""
		if inp.Type == domain.ParamVector && inp.UseRGBMask {
			maskStr = ",Mask=1,MaskR=1,MaskG=1,MaskB=1"
		}
		inputsStrings = append(inputsStrings, fmt.Sprintf(
			"      Inputs(%d)=(InputName=\"%s\",Input=(Expression=%s'\"%s.%s\"'%s))",
			i, inp.Name, exprClass, graphName, exprName, maskStr,
		))
	}

	return nodes, inputsStrings, diags
}

// buildBreakOut creates the BreakOutFloat4Components function call node.
func buildBreakOut(customNode *domain.GraphNode, nodeIndex *int, guidFn domain.GUIDFunc) (*domain.GraphNode, map[string]string) {
	graphName := fmt.Sprintf("MaterialGraphNode_%d", *nodeIndex)
	*nodeIndex++
	exprGUID := guidFn()
	exprName := fmt.Sprintf("MaterialExpressionMaterialFunctionCall_%s", exprGUID[:8])

	inputPin := &domain.Pin{
		ID:               guidFn(),
		Name:             "Float4 (V4)",
		Dir:              domain.PinDirIn,
		Category:         "optional",
		IsUObjectWrapper: true,
	}
	pinR := &domain.Pin{ID: guidFn(), Name: "R", Dir: domain.PinDirOut, IsUObjectWrapper: true}
	pinG := &domain.Pin{ID: guidFn(), Name: "G", Dir: domain.PinDirOut, IsUObjectWrapper: true}
	pinB := &domain.Pin{ID: guidFn(), Name: "B", Dir: domain.PinDirOut, IsUObjectWrapper: true}
	pinA := &domain.Pin{ID: guidFn(), Name: "A", Dir: domain.PinDirOut, IsUObjectWrapper: true}

	pinMap := map[string]string{"R": pinR.ID, "G": pinG.ID, "B": pinB.ID, "A": pinA.ID}

	const matFunc = `/Engine/Functions/Engine_MaterialFunctions02/Utility/BreakOutFloat4Components.BreakOutFloat4Components`
	lines := []string{
		fmt.Sprintf("      MaterialFunction=MaterialFunction'\"%s\"'", matFunc),
		fmt.Sprintf("      FunctionInputs(0)=(ExpressionInputId=%s,Input=(Expression=MaterialExpressionCustom'\"%s.%s\"',InputName=\"Float4\"))",
			guidFn(), customNode.GraphName, customNode.ExprName),
		fmt.Sprintf("      FunctionOutputs(0)=(ExpressionOutputId=%s,Output=(OutputName=\"R\"))", guidFn()),
		fmt.Sprintf("      FunctionOutputs(1)=(ExpressionOutputId=%s,Output=(OutputName=\"G\"))", guidFn()),
		fmt.Sprintf("      FunctionOutputs(2)=(ExpressionOutputId=%s,Output=(OutputName=\"B\"))", guidFn()),
		fmt.Sprintf("      FunctionOutputs(3)=(ExpressionOutputId=%s,Output=(OutputName=\"A\"))", guidFn()),
		`      Outputs(0)=(OutputName="R")`,
		`      Outputs(1)=(OutputName="G")`,
		`      Outputs(2)=(OutputName="B")`,
		`      Outputs(3)=(OutputName="A")`,
	}

	return &domain.GraphNode{
		GraphName: graphName,
		ExprName:  exprName,
		ExprClass: "MaterialExpressionMaterialFunctionCall",
		X:         -96, Y: 608,
		NodeGUID:  guidFn(),
		ExprGUID:  exprGUID,
		ExtraBody: strings.Join(lines, "\r\n"),
		Pins:      []*domain.Pin{inputPin, pinR, pinG, pinB, pinA},
	}, pinMap
}

// buildCustomExtraBody constructs the extraBody for the Custom expression node.
func buildCustomExtraBody(hlsl string, ot domain.OutputType, inputsStrings []string) string {
	escaped := escapeHLSL(hlsl)
	parts := []string{
		fmt.Sprintf("      Code=\"%s\"", escaped),
		fmt.Sprintf("      OutputType=%s", ot),
	}
	parts = append(parts, inputsStrings...)
	parts = append(parts, `      Desc="Generated by BuilderToolKit"`)
	return strings.Join(parts, "\r\n")
}

// escapeHLSL escapes HLSL code for embedding in the T3D Code field.
// Order matters: backslash first, then CRLF, then bare LF, then quotes.
func escapeHLSL(code string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"\r\n", `\r\n`,
		"\n", `\r\n`,
		`"`, `\"`,
		"\r", `\r\n`,
	)
	return r.Replace(code)
}

func findRootPin(root *domain.GraphNode, slotName string) *domain.Pin {
	for _, p := range root.Pins {
		if p.Name == slotName {
			return p
		}
	}
	return nil
}

func applyEdges(edges []domain.Edge, nodes []*domain.GraphNode) error {
	pinIndex := map[string]*domain.Pin{} // "graphName:pinID" -> *Pin
	for _, n := range nodes {
		for _, p := range n.Pins {
			pinIndex[n.GraphName+":"+p.ID] = p
		}
	}
	for _, e := range edges {
		fromPin := pinIndex[e.From.GraphName+":"+e.From.PinID]
		toPin := pinIndex[e.To.GraphName+":"+e.To.PinID]
		// Invariant: edges are built from pin IDs that exist in the node list.
		// A nil pin here means the edge list and node list disagree — a bug.
		if fromPin == nil || toPin == nil {
			return fmt.Errorf("applyEdges: edge references unknown pin (from=%s:%s to=%s:%s)",
				e.From.GraphName, e.From.PinID, e.To.GraphName, e.To.PinID)
		}
		fromPin.LinkedTo = append(fromPin.LinkedTo, domain.PinRef{GraphName: e.To.GraphName, PinID: e.To.PinID})
		toPin.LinkedTo = append(toPin.LinkedTo, domain.PinRef{GraphName: e.From.GraphName, PinID: e.From.PinID})
	}
	return nil
}

func parseScalarDefault(val string) (string, error) {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return strconv.FormatFloat(0, 'f', 6, 64), nil
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return strconv.FormatFloat(0, 'f', 6, 64), fmt.Errorf("invalid scalar default %q: %w", val, err)
	}
	return strconv.FormatFloat(f, 'f', 6, 64), nil
}

func parseVectorDefault(val string) (r, g, b, a string, err error) {
	rr, gg, bb, aa := 0.0, 0.0, 0.0, 1.0
	if val == "" {
		return fmtF(rr), fmtF(gg), fmtF(bb), fmtF(aa), nil
	}
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	var bad []string
	if len(parts) == 1 {
		v, perr := strconv.ParseFloat(parts[0], 64)
		if perr != nil {
			bad = append(bad, parts[0])
		} else {
			rr, gg, bb = v, v, v
		}
	} else {
		targets := []*float64{&rr, &gg, &bb, &aa}
		for i := 0; i < len(parts) && i < 4; i++ {
			v, perr := strconv.ParseFloat(parts[i], 64)
			if perr != nil {
				bad = append(bad, parts[i])
				continue
			}
			*targets[i] = v
		}
	}
	if len(bad) > 0 {
		err = fmt.Errorf("invalid vector default %q: unparseable component(s) %v", val, bad)
	}
	return fmtF(rr), fmtF(gg), fmtF(bb), fmtF(aa), err
}

func fmtF(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		f = 0
	}
	return strconv.FormatFloat(f, 'f', 6, 64)
}
