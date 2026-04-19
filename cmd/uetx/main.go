package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	material "github.com/radial/uetx/internal/app/material"
	"github.com/radial/uetx/internal/domain"
	"github.com/radial/uetx/internal/version"
)

const (
	exitOK       = 0
	exitBusiness = 1
	exitWarn     = 2
	exitUsage    = 64
	exitInternal = 70
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitUsage)
	}

	cmd := os.Args[1]

	// Handle "material generate" as "material" + "generate" or just "generate" shorthand
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println(version.Info())
		return
	case "material":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: uetx material <generate|inspect|validate> [flags]")
			os.Exit(exitUsage)
		}
		runMaterialCommand(os.Args[2], os.Args[3:])
	case "generate", "inspect", "validate":
		// Shorthand: "uetx generate" == "uetx material generate"
		runMaterialCommand(cmd, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(exitUsage)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: uetx <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  material generate   Convert HLSL template to T3D material graph")
	fmt.Fprintln(os.Stderr, "  material inspect    Parse HLSL template, show inferred metadata")
	fmt.Fprintln(os.Stderr, "  material validate   Check HLSL template and config for errors")
	fmt.Fprintln(os.Stderr, "  version             Print version info")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Shorthand: 'uetx generate' is equivalent to 'uetx material generate'")
}

func runMaterialCommand(action string, args []string) {
	fs := flag.NewFlagSet("uetx material "+action, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	inFile := fs.String("i", "-", "Input file (- for stdin)")
	fs.StringVar(inFile, "in", "-", "Input file (- for stdin)")

	outFile := fs.String("o", "-", "Output file (- for stdout)")
	fs.StringVar(outFile, "out", "-", "Output file (- for stdout)")

	configFile := fs.String("c", "", "JSON config file")
	fs.StringVar(configFile, "config", "", "JSON config file")

	matName := fs.String("m", "", "Material name")
	fs.StringVar(matName, "material", "", "Material name")

	outputType := fs.String("t", "", "Output type (CMOT_Float1..4)")
	fs.StringVar(outputType, "output-type", "", "Output type (CMOT_Float1..4)")

	jsonOutput := fs.Bool("json", false, "Output JSON response")
	stdinJSON := fs.Bool("stdin-json", false, "Read JSON request from stdin")
	seed := fs.Int64("seed", 0, "Fixed GUID seed for reproducibility")
	clipboard := fs.Bool("clipboard", false, "Copy T3D to clipboard")
	noCRLF := fs.Bool("no-crlf", false, "Output LF instead of CRLF (debug)")

	var routes multiFlag
	fs.Var(&routes, "r", "Routing slot (repeatable)")
	fs.Var(&routes, "route", "Routing slot (repeatable)")

	var inputs multiFlag
	fs.Var(&inputs, "input", "Input spec name:type[:default[:rgb]] (repeatable)")

	if err := fs.Parse(args); err != nil {
		os.Exit(exitUsage)
	}

	switch action {
	case "generate":
		runGenerate(*inFile, *outFile, *configFile, *matName, *outputType,
			routes, inputs, *jsonOutput, *stdinJSON, *seed, *clipboard, *noCRLF)
	case "inspect":
		runInspect(*inFile, *jsonOutput)
	case "validate":
		runValidate(*inFile, *configFile, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "unknown material action: %s\n", action)
		os.Exit(exitUsage)
	}
}

func runGenerate(inFile, outFile, configFile, matName, outputType string,
	routes, inputSpecs []string, jsonOutput, stdinJSON bool, seed int64, clipboard, noCRLF bool) {

	var req domain.GenerateRequest

	if stdinJSON {
		// Read JSON request from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		if err := json.Unmarshal(data, &req); err != nil {
			log.Printf("E100: JSON parse failed: %v", err)
			os.Exit(exitBusiness)
		}
	} else {
		// Read HLSL from file or stdin
		hlsl := readInput(inFile)
		req.HLSL = hlsl
	}

	// Apply CLI overrides
	if matName != "" {
		req.MaterialName = matName
	}
	if outputType != "" {
		req.OutputType = domain.OutputType(outputType)
	}
	if len(routes) > 0 {
		req.Routing = routes
	}
	if seed != 0 {
		req.Seed = seed
	}
	if len(inputSpecs) > 0 {
		for _, spec := range inputSpecs {
			inp, err := parseInputSpec(spec)
			if err != nil {
				log.Printf("E103: %v", err)
				os.Exit(exitBusiness)
			}
			req.Inputs = append(req.Inputs, inp)
		}
	}

	// Apply JSON config file
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("E100: read config: %v", err)
			os.Exit(exitBusiness)
		}
		var cfg domain.GenerateRequest
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Printf("E100: JSON parse failed: %v", err)
			os.Exit(exitBusiness)
		}
		mergeConfig(&req, &cfg)
	}

	resp := material.Generate(req)

	if jsonOutput {
		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
		if !resp.OK {
			os.Exit(exitBusiness)
		}
		return
	}

	if !resp.OK {
		for _, e := range resp.Errors {
			log.Printf("%s: %s", e.Code, e.Message)
		}
		os.Exit(exitBusiness)
	}

	for _, w := range resp.Warnings {
		log.Printf("%s: %s", w.Code, w.Message)
	}

	output := resp.T3D
	if noCRLF {
		output = strings.ReplaceAll(output, "\r\n", "\n")
	}

	writeOutput(outFile, output)

	if clipboard {
		if err := copyToClipboard(output); err != nil {
			log.Printf("W006: clipboard copy failed: %v", err)
		}
	}
}

func runInspect(inFile string, jsonOutput bool) {
	hlsl := readInput(inFile)
	resp := material.Inspect(hlsl)

	if jsonOutput {
		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
	} else {
		fmt.Printf("Output Type: %s\n", resp.EffectiveOutputType)
		fmt.Printf("Routing: %s\n", strings.Join(resp.EffectiveRouting, ", "))
		fmt.Printf("Inputs (%d):\n", len(resp.InferredInputs))
		for _, inp := range resp.InferredInputs {
			def := ""
			if inp.DefaultValue != "" {
				def = fmt.Sprintf(" (default: %s)", inp.DefaultValue)
			}
			fmt.Printf("  - %s: %s%s\n", inp.Name, inp.Type, def)
		}
		for _, w := range resp.Warnings {
			log.Printf("%s: %s", w.Code, w.Message)
		}
		for _, e := range resp.Errors {
			log.Printf("%s: %s", e.Code, e.Message)
		}
	}

	if !resp.OK {
		os.Exit(exitBusiness)
	}
}

func runValidate(inFile, configFile string, jsonOutput bool) {
	hlsl := readInput(inFile)
	var configJSON []byte
	if configFile != "" {
		var err error
		configJSON, err = os.ReadFile(configFile)
		if err != nil {
			log.Printf("E100: read config: %v", err)
			os.Exit(exitBusiness)
		}
	}

	resp := material.Validate(hlsl, configJSON)

	if jsonOutput {
		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
	} else {
		if resp.OK {
			fmt.Println("OK")
		}
		for _, w := range resp.Warnings {
			log.Printf("%s: %s", w.Code, w.Message)
		}
		for _, e := range resp.Errors {
			log.Printf("%s: %s", e.Code, e.Message)
		}
	}

	if !resp.OK {
		os.Exit(exitBusiness)
	}
	if len(resp.Warnings) > 0 {
		os.Exit(exitWarn)
	}
}

func readInput(path string) string {
	if path == "-" || path == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		return string(data)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func writeOutput(path, content string) {
	if path == "-" || path == "" {
		fmt.Print(content)
		return
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
}

func parseInputSpec(spec string) (domain.NodeInput, error) {
	parts := strings.SplitN(spec, ":", 4)
	if len(parts) < 2 {
		return domain.NodeInput{}, fmt.Errorf("invalid input spec %q (need name:type[:default[:rgb]])", spec)
	}
	inp := domain.NodeInput{
		Name: parts[0],
		Type: domain.ParamType(parts[1]),
	}
	if len(parts) >= 3 {
		inp.DefaultValue = parts[2]
	}
	if len(parts) >= 4 && strings.EqualFold(parts[3], "true") {
		inp.UseRGBMask = true
	}
	return inp, nil
}

func mergeConfig(req, cfg *domain.GenerateRequest) {
	if cfg.MaterialName != "" && req.MaterialName == "" {
		req.MaterialName = cfg.MaterialName
	}
	if cfg.OutputType != "" && req.OutputType == "" {
		req.OutputType = cfg.OutputType
	}
	if len(cfg.Inputs) > 0 && len(req.Inputs) == 0 {
		req.Inputs = cfg.Inputs
	}
	if len(cfg.Routing) > 0 && len(req.Routing) == 0 {
		req.Routing = cfg.Routing
	}
	if cfg.Seed != 0 && req.Seed == 0 {
		req.Seed = cfg.Seed
	}
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip.exe")
	default:
		// Try wl-copy first, then xclip
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			return fmt.Errorf("no clipboard command found")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// multiFlag allows repeatable string flags.
type multiFlag []string

func (f *multiFlag) String() string { return strings.Join(*f, ",") }
func (f *multiFlag) Set(val string) error {
	*f = append(*f, val)
	return nil
}
