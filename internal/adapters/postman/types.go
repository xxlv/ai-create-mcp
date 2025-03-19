package postman

type CollectionResponse struct {
	Collection Collection `json:"collection"`
}

type Event struct {
	Listen string  `json:"listen"`
	Script *Script `json:"script,omitempty"`
}

type Script struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Collection struct {
	Info     Info       `json:"info"`
	Item     []Item     `json:"item"`
	Event    []Event    `json:"event,omitempty"`
	Variable []Variable `json:"variable,omitempty"`
}

// Info contains metadata about the collection
type Info struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Description string `json:"description,omitempty"`
}

// Item represents an item in the collection, which can be a request or a folder
type Item struct {
	Name     string     `json:"name"`
	Request  *Request   `json:"request,omitempty"`
	Response []Response `json:"response,omitempty"`
	Item     []Item     `json:"item,omitempty"` // Nested items (for folders)
}

// Request represents an HTTP request
type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header,omitempty"`
	Body   *Body    `json:"body,omitempty"`
	URL    URL      `json:"url"`
}

// Header represents an HTTP header
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Body represents the request body
type Body struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
	// Adding options field to support JSON format info
	Options *BodyOptions `json:"options,omitempty"`
}

// BodyOptions represents options for the request body
type BodyOptions struct {
	Raw *RawOptions `json:"raw,omitempty"`
}

// RawOptions represents options for raw body
type RawOptions struct {
	Language string `json:"language,omitempty"`
}

// URL represents the request URL
type URL struct {
	Raw      string   `json:"raw"`
	Protocol string   `json:"protocol,omitempty"`
	Host     []string `json:"host,omitempty"`
	Path     []string `json:"path,omitempty"`
	Query    []Query  `json:"query,omitempty"`
}

// Query represents query parameters
type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Response represents an example response
type Response struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Code   int      `json:"code"`
	Header []Header `json:"header,omitempty"`
	Body   string   `json:"body,omitempty"`
}
