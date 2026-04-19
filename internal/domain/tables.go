package domain

// RootPinEntry defines one input slot on the material Root node.
type RootPinEntry struct {
	Name string
	Sub  string // PinSubCategory — a numeric string, not an int
}

// RootPinTable is the fixed list of 30 material input slots on the Root node.
// Order matters — it must match UE's expected pin ordering.
var RootPinTable = []RootPinEntry{
	{"Base Color", "5"},
	{"Metallic", "6"},
	{"Specular", "7"},
	{"Roughness", "8"},
	{"Anisotropy", "9"},
	{"Emissive Color", "0"},
	{"Opacity", "1"},
	{"Opacity Mask", "2"},
	{"Normal", "10"},
	{"Tangent", "11"},
	{"World Position Offset", "12"},
	{"World Displacement", "13"},
	{"Tessellation Multiplier", "14"},
	{"Subsurface Color", "15"},
	{"Custom Data 0", "16"},
	{"Custom Data 1", "17"},
	{"Tree Light Info", "30"},
	{"Ambient Occlusion", "18"},
	{"Refraction", "19"},
	{"Customized UV0", "20"},
	{"Customized UV1", "21"},
	{"Customized UV2", "22"},
	{"Customized UV3", "23"},
	{"Customized UV4", "24"},
	{"Customized UV5", "25"},
	{"Customized UV6", "26"},
	{"Customized UV7", "27"},
	{"Pixel Depth Offset", "28"},
	{"Shading Model", "29"},
	{"Material Attributes", "31"},
}

// ScalarSlots lists Root pin names that only accept Float1.
// When routing a Float4 output to these, a BreakOut node is required.
var ScalarSlots = map[string]struct{}{
	"Opacity":                 {},
	"Opacity Mask":            {},
	"Metallic":                {},
	"Specular":                {},
	"Roughness":               {},
	"Anisotropy":              {},
	"Ambient Occlusion":       {},
	"Refraction":              {},
	"Tessellation Multiplier": {},
	"Pixel Depth Offset":      {},
}

// DefaultRouting returns the default Root slot targets for a given OutputType.
func DefaultRouting(ot OutputType) []MaterialOutputSlot {
	switch ot {
	case CMOTFloat1:
		return []string{"Emissive Color"}
	case CMOTFloat2:
		return []string{"Emissive Color"}
	case CMOTFloat3:
		return []string{"Base Color"}
	case CMOTFloat4:
		return []string{"Base Color", "Opacity"}
	default:
		return []string{"Base Color"}
	}
}
