package core

type TemplateData struct {
	MissBaseURL       bool // when openapi miss server set True
	BinaryName        string
	Endpoints         []string
	ServerName        string
	ServerVersion     string
	Resources         []Resource
	Prompts           []Prompt
	Tools             []Tool
	ServerDescription string
	ServerDirectory   string
}

type Resource struct {
	Name        string
	Description string
	URI         string
	MimeType    string
}

type Prompt struct {
	Name        string
	Description string
	Arguments   []Argument
}

type Argument struct {
	Name        string
	Description string
	Required    bool
}

type Tool struct {
	Name        string
	Description string
	Arguments   []Argument
	Method      string
	Path        string
}
