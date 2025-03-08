package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	_ "embed"

	"github.com/Masterminds/semver/v3"
	"github.com/pelletier/go-toml"
)

//go:embed templates/__init__.py.tmpl
var initTpl string

//go:embed templates/serveroas.py.tmpl
var serverTpl string

//go:embed templates/README.md.tmpl
var readmeTpl string

const (
	MinUVVersion = "0.4.10"
)

type PyProject struct {
	Data *toml.Tree
}

func NewPyProject(path string) (*PyProject, error) {
	data, err := toml.LoadFile(path)
	if err != nil {
		return nil, err
	}
	return &PyProject{Data: data}, nil
}

func (p *PyProject) Name() string {
	return p.Data.Get("project.name").(string)
}

func (p *PyProject) FirstBinary() string {
	scripts, ok := p.Data.Get("project.scripts").(map[string]interface{})
	if !ok || len(scripts) == 0 {
		return ""
	}
	for k := range scripts {
		return k
	}
	return ""
}

func checkUVVersion(requiredVersion string) (string, error) {
	cmd := exec.Command("uv", "--version")
	output, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.Error); ok {
			return "", nil // uv not found
		}
		return "", fmt.Errorf("failed to check uv version: %v", err)
	}

	version := strings.TrimSpace(string(output))
	re := regexp.MustCompile(`uv (\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 2 {
		return "", nil
	}

	versionNum := matches[1]
	reqVer, _ := semver.NewVersion(requiredVersion)
	curVer, _ := semver.NewVersion(versionNum)
	if curVer.Compare(reqVer) >= 0 {
		return version, nil
	}
	return "", nil
}

func ensureUVInstalled() error {
	version, err := checkUVVersion(MinUVVersion)
	if err != nil || version == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: uv >= %s is required but not installed.\n", MinUVVersion)
		fmt.Fprintf(os.Stderr, "To install, visit: https://github.com/astral-sh/uv\n")
		os.Exit(1)
	}
	return nil
}

func getClaudeConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var path string
	switch os.Getenv("GOOS") {
	case "windows":
		path = filepath.Join(home, "AppData", "Roaming", "Claude")
	case "darwin":
		path = filepath.Join(home, "Library", "Application Support", "Claude")
	default:
		return "", nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}
	return path, nil
}

func hasClaudeApp() bool {
	path, _ := getClaudeConfigPath()
	return path != ""
}

func updateClaudeConfig(projectName, projectPath string) bool {
	configDir, err := getClaudeConfigPath()
	if err != nil || configDir == "" {
		return false
	}

	configFile := filepath.Join(configDir, "claude_desktop_config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read Claude.app configuration: %v\n", err)
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to parse Claude.app configuration: %v\n", err)
		return false
	}

	if _, ok := config["mcpServers"]; !ok {
		config["mcpServers"] = make(map[string]interface{})
	}

	mcpServers := config["mcpServers"].(map[string]interface{})
	if _, exists := mcpServers[projectName]; exists {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: %s already exists in Claude.app configuration\n", projectName)
		fmt.Fprintf(os.Stderr, "Settings file location: %s\n", configFile)
		return false
	}

	mcpServers[projectName] = map[string]interface{}{
		"command": "uv",
		"args":    []string{"--directory", projectPath, "run", projectName},
	}

	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to marshal Claude.app configuration: %v\n", err)
		return false
	}

	if err := os.WriteFile(configFile, updatedData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write Claude.app configuration: %v\n", err)
		return false
	}

	fmt.Printf("✅ Added %s to Claude.app configuration\n", projectName)
	fmt.Printf("Settings file location: %s\n", configFile)
	return true
}

func getPackageDirectory(path string) (string, error) {
	srcDir := filepath.Join(path, "src")
	matches, err := filepath.Glob(filepath.Join(srcDir, "*", "__init__.py"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("could not find __init__.py in src directory")
	}
	return filepath.Dir(matches[0]), nil
}

func copyTemplate(path, name, description, version, oasPath string) error {
	targetDir, err := getPackageDirectory(path)
	if err != nil {
		return err
	}

	pyproject, err := NewPyProject(filepath.Join(path, "pyproject.toml"))
	if err != nil {
		return fmt.Errorf("failed to load pyproject.toml: %v", err)
	}
	templateVars := TemplateData{
		BinaryName:        pyproject.FirstBinary(),
		ServerName:        name,
		ServerVersion:     "1.0.0",
		ServerDescription: description,
		ServerDirectory:   path,
		Resources: []Resource{
			{Name: "Note1", Description: "A simple note", URI: "note://internal/note1", MimeType: "text/plain"},
		},
		Prompts: []Prompt{
			{
				Name:        "summarize-notes",
				Description: "Summarize all notes",
				Arguments: []Argument{
					{Name: "style", Description: "Summary style", Required: false},
				},
			},
		},
		Tools: []Tool{
			{
				Name:        "add-note",
				Description: "Add a new note",
				Arguments: []Argument{
					{Name: "name", Description: "Note name", Required: true},
					{Name: "content", Description: "Note content", Required: true},
				},
			},

			{
				Name:        "add-note-2",
				Description: "Add a new note",
				Arguments: []Argument{
					{Name: "name", Description: "Note name", Required: true},
					{Name: "content", Description: "Note content", Required: true},
				},
			},
		},
	}

	// doc, err := openapi3.NewLoader().LoadFromFile(oasPath)
	// templateVars, err := ConvertOAStoTemplateData(doc)
	// if err != nil {
	// 	return fmt.Errorf("failed to load oas file: %v", err)
	// }

	templates := []struct {
		name      string
		content   string
		outputDir string
	}{
		{
			name:      "__init__.py",
			content:   initTpl,
			outputDir: targetDir,
		},
		{
			name:      "server.py",
			content:   serverTpl,
			outputDir: targetDir,
		},
		{
			name:      "README.md",
			content:   readmeTpl,
			outputDir: path,
		},
	}

	for _, t := range templates {
		tmpl, err := template.New(t.name).Parse(t.content) // In practice, load from file or embed
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %v", t.name, err)
		}

		outPath := filepath.Join(t.outputDir, t.name)
		file, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", outPath, err)
		}
		defer file.Close()

		if err := tmpl.Execute(file, templateVars); err != nil {
			return fmt.Errorf("failed to render template %s: %v", t.name, err)
		}
	}

	return nil
}

func checkPackageName(name string) bool {
	if name == "" {
		fmt.Fprintln(os.Stderr, "❌ Project name cannot be empty")
		return false
	}
	if strings.Contains(name, " ") {
		fmt.Fprintln(os.Stderr, "❌ Project name must not contain spaces")
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			fmt.Fprintln(os.Stderr, "❌ Project name must consist of ASCII letters, digits, underscores, hyphens, and periods")
			return false
		}
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, "_") || strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		fmt.Fprintln(os.Stderr, "❌ Project name must not start or end with an underscore, hyphen, or period")
		return false
	}
	return true
}

func createProject(path, name, description, version, oasPath string, useClaude bool) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	cmd := exec.Command("uv", "init", "--name", name, "--package", "--app", "--quiet")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize project: %v", err)
	}

	cmd = exec.Command("uv", "add", "mcp")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add mcp dependency: %v", err)
	}

	if err := copyTemplate(path, name, description, version, oasPath); err != nil {
		return fmt.Errorf("failed to copy templates: %v", err)
	}

	if useClaude && hasClaudeApp() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nClaude.app detected. Would you like to install the server into Claude.app now? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(response)) != "n" {
			updateClaudeConfig(name, path)
		}
	}
	basePath, _ := os.Getwd()
	relPath, _ := filepath.Rel(basePath, path)
	fmt.Printf("✅ Created project %s in %s\n", name, relPath)
	fmt.Printf("ℹ️ To install dependencies run:\n")
	fmt.Printf("   cd %s\n", relPath)
	fmt.Println("   uv sync --dev --all-extras")
	return compileDep(relPath)
}

func compileDep(workspacePath string) error {
	fmt.Printf("ℹ️ Installing dependencies...\n")
	cmd := exec.Command("uv", "sync", "--dev", "--all-extras")
	cmd.Dir = workspacePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to sync dependencies: %v", err)
	}
	return nil
}

func updatePyProjectSettings(projectPath, version, description string) error {
	pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
	data, err := toml.LoadFile(pyprojectPath)
	if err != nil {
		return fmt.Errorf("pyproject.toml not found: %v", err)
	}

	if version != "" {
		data.Set("project.version", version)
	}
	if description != "" {
		data.Set("project.description", description)
	}

	file, err := os.Create(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to open pyproject.toml: %v", err)
	}
	defer file.Close()
	return toml.NewEncoder(file).Encode(data)
}
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux, bsd, etc
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// runInspector use mcp inspcector package
func runInspector(projectPath, projectName string) error {
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector", "uv", "--directory", projectPath, "run", projectName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	go tryOpenBrowser()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run inspector: %v", err)
	}
	return nil
}

func tryOpenBrowser() {
	url := "http://localhost:5173"
	timeout := time.After(5 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return
		case <-tick:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if err := openBrowser(url); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ Warning: Inspector is running but failed to open browser: %v\n", err)
				}
				fmt.Printf("\n🔍 MCP Inspector is up and running at %s 🚀\n", url)
				return
			}
		}
	}
}

func main() {
	var (
		path        string
		name        string
		oasPath     string
		version     string
		description string
		claudeApp   bool
		inspector   bool
	)
	flag.StringVar(&path, "path", "", "Directory to create project in")
	flag.StringVar(&name, "name", "", "Project name")
	flag.StringVar(&oasPath, "oaspath", "", "Oas path")
	flag.StringVar(&version, "version", "0.1.0", "Server version")
	flag.BoolVar(&inspector, "inspector", true, "Open inspector")
	flag.StringVar(&description, "description", "Simple mcp", "Project description")
	flag.BoolVar(&claudeApp, "claudeapp", true, "Enable/disable Claude.app integration")

	flag.Parse()

	if err := ensureUVInstalled(); err != nil {
		os.Exit(1)
	}
	fmt.Println("Creating a new MCP server project using uv.")
	fmt.Println("This will set up a Python project with MCP dependency.")
	fmt.Println("\nLet's begin!")

	reader := bufio.NewReader(os.Stdin)
	if name == "" {
		fmt.Print("Project name (required): ")
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)
	}

	if !checkPackageName(name) {
		os.Exit(1)
	}

	if description == "" {
		fmt.Print("Project description [A MCP server project]: ")
		description, _ = reader.ReadString('\n')
		description = strings.TrimSpace(description)
		if description == "" {
			description = "A MCP server project"
		}
	}

	if version == "" {
		fmt.Print("Project version [0.1.0]: ")
		version, _ = reader.ReadString('\n')
		version = strings.TrimSpace(version)
		if version == "" {
			version = "0.1.0"
		}
	}
	if oasPath == "" {
		fmt.Print("Oas path(required): ")
		oasPath, _ = reader.ReadString('\n')
		oasPath = strings.TrimSpace(oasPath)
	}

	if _, err := semver.NewVersion(version); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: Version must be a valid semantic version (e.g. 1.0.0): %v\n", err)
		os.Exit(1)
	}
	basePath, _ := os.Getwd()
	projectPath := filepath.Join(basePath, name)
	if path != "" {
		projectPath = path
	} else {
		// fmt.Printf("Project will be created at: %s\n", projectPath)
		// fmt.Print("Is this correct? [Y/n]: ")
		// response, _ := reader.ReadString('\n')
		// if strings.TrimSpace(strings.ToLower(response)) == "n" {
		// 	fmt.Print("Enter the correct path: ")
		// 	projectPath, _ = reader.ReadString('\n')
		// 	projectPath = strings.TrimSpace(projectPath)
		// }
		// for debug
		// projectPath, _ = reader.ReadString('\n')
		projectPath = strings.TrimSpace(projectPath)
	}

	projectPath = filepath.Clean(projectPath)
	if err := createProject(projectPath, name, description, version, oasPath, claudeApp); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	if err := updatePyProjectSettings(projectPath, version, description); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error updating pyproject.toml: %v\n", err)
		os.Exit(1)
	}

	if inspector {
		if err := runInspector(projectPath, name); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error running inspector: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Inspector executed successfully")
	}
	os.Exit(0)
}
