package postman

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
	"github.com/xxlv/ai-create-mcp/internal/adapters/shared"
)

func convert(postmanCollection *CollectionResponse) (*core.TemplateData, error) {
	if postmanCollection == nil {
		return nil, fmt.Errorf("no postman collection found")
	}
	oas := convertToOpenAPI(&postmanCollection.Collection)
	if oas == nil {
		return nil, fmt.Errorf("postman collection does not convert to oas3")
	}
	return shared.Convert(oas)
}

func convertToOpenAPI(collection *Collection) *openapi3.T {
	if collection == nil {
		return nil
	}
	comp := openapi3.NewComponents()
	oas := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:       collection.Info.Name,
			Description: collection.Info.Description,
			Version:     "1.0.0",
		},
		Paths:      &openapi3.Paths{},
		Components: &comp,
		Servers:    openapi3.Servers{},
	}

	processItems(collection.Item, "", oas)
	return oas
}

func processItems(items []Item, parentPath string, oas *openapi3.T) {
	for _, item := range items {
		if len(item.Item) > 0 {
			tag := &openapi3.Tag{
				Name:        item.Name,
				Description: "Operations in " + item.Name,
			}
			oas.Tags = append(oas.Tags, tag)

			newParentPath := parentPath
			if parentPath != "" {
				newParentPath += "." + item.Name
			} else {
				newParentPath = item.Name
			}
			processItems(item.Item, newParentPath, oas)
		} else if item.Request != nil {
			processRequest(item, parentPath, oas)
		}
	}
}

func processRequest(item Item, folderTag string, oas *openapi3.T) {
	req := item.Request

	parsedURL, err := url.Parse(req.URL.Raw)
	if err != nil {
		log.Printf("Warning: failed to parse URL %s: %v", req.URL.Raw, err)
		return
	}

	// Add server info if not already present
	if parsedURL.Host != "" {
		serverURL := parsedURL.Scheme + "://" + parsedURL.Host
		serverExists := false
		for _, s := range oas.Servers {
			if s.URL == serverURL {
				serverExists = true
				break
			}
		}
		if !serverExists {
			oas.Servers = append(oas.Servers, &openapi3.Server{URL: serverURL})
		}
	}

	pathTemplate := convertPathParams(parsedURL.Path)
	operation := &openapi3.Operation{
		Summary:     item.Name,
		Description: item.Name,
		OperationID: fmt.Sprintf("%s_%s", strings.ToLower(req.Method), sanitizeOperationID(pathTemplate+"_"+item.Name)),
		Responses:   openapi3.NewResponses(),
	}

	if folderTag != "" {
		operation.Tags = strings.Split(folderTag, ".")
	}

	processRequestHeaders(req, operation)
	processRequestBody(req, operation)
	processQueryParams(req, operation)
	processPathParams(pathTemplate, req, operation)
	processResponses(item.Response, operation)

	if oas.Paths.Find(pathTemplate) == nil {
		oas.Paths.Set(pathTemplate, &openapi3.PathItem{})
	}
	pathItem := oas.Paths.Find(pathTemplate)

	switch strings.ToUpper(req.Method) {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "HEAD":
		pathItem.Head = operation
	case "OPTIONS":
		pathItem.Options = operation
	case "TRACE":
		pathItem.Trace = operation
	}
}

func processRequestHeaders(req *Request, operation *openapi3.Operation) {
	for _, header := range req.Header {
		if strings.ToLower(header.Key) == "content-type" {
			continue
		}
		param := &openapi3.Parameter{
			Name:        header.Key,
			In:          "header",
			Description: "Header parameter " + header.Key,
			Required:    false,
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    &openapi3.Types{"string"},
					Example: header.Value,
				},
			},
		}
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
	}
}

func processRequestBody(req *Request, operation *openapi3.Operation) {
	if req.Body == nil {
		return
	}

	content := openapi3.NewContent()
	switch req.Body.Mode {
	case "raw":
		if req.Body.Raw == "" {
			return
		}
		contentType := "text/plain"
		for _, header := range req.Header {
			if strings.ToLower(header.Key) == "content-type" {
				contentType = header.Value
				break
			}
		}
		if req.Body.Options != nil && req.Body.Options.Raw != nil && req.Body.Options.Raw.Language == "json" {
			contentType = "application/json"
		}
		schema := &openapi3.Schema{Type: &openapi3.Types{"string"}}
		if strings.Contains(contentType, "json") {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(req.Body.Raw), &jsonData); err == nil {
				schema = inferSchemaFromJSON(jsonData)
			} else {
				log.Printf("Warning: failed to parse JSON body: %v", err)
			}
		}
		content[contentType] = &openapi3.MediaType{
			Schema:  &openapi3.SchemaRef{Value: schema},
			Example: req.Body.Raw,
		}
	}

	if len(content) > 0 {
		operation.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content:  content,
			},
		}
	}
}

func inferSchemaFromJSON(data interface{}) *openapi3.Schema {
	switch v := data.(type) {
	case map[string]interface{}:
		schema := &openapi3.Schema{
			Type:       &openapi3.Types{"object"},
			Properties: make(map[string]*openapi3.SchemaRef),
		}
		for key, val := range v {
			schema.Properties[key] = &openapi3.SchemaRef{Value: inferSchemaFromJSON(val)}
		}
		return schema
	case []interface{}:
		if len(v) == 0 {
			return &openapi3.Schema{
				Type:  &openapi3.Types{"array"},
				Items: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
			}
		}
		return &openapi3.Schema{
			Type:  &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{Value: inferSchemaFromJSON(v[0])},
		}
	case string:
		return &openapi3.Schema{Type: &openapi3.Types{"string"}}
	case float64:
		if v == float64(int64(v)) {
			return &openapi3.Schema{Type: &openapi3.Types{"integer"}}
		}
		return &openapi3.Schema{Type: &openapi3.Types{"number"}}
	case bool:
		return &openapi3.Schema{Type: &openapi3.Types{"boolean"}}
	case nil:
		return &openapi3.Schema{Type: &openapi3.Types{"null"}}
	default:
		return &openapi3.Schema{Type: &openapi3.Types{"string"}}
	}
}

func processQueryParams(req *Request, operation *openapi3.Operation) {
	for _, query := range req.URL.Query {
		param := &openapi3.Parameter{
			Name:        query.Key,
			In:          "query",
			Description: "Query parameter " + query.Key,
			Required:    false,
			Schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:    &openapi3.Types{"string"},
					Example: query.Value,
				},
			},
		}
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
	}
}

func convertPathParams(pathTemplate string) string {
	var sb strings.Builder
	segments := strings.Split(pathTemplate, "/")
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, ":") {
			sb.WriteString("/{")
			sb.WriteString(segment[1:])
			sb.WriteString("}")
		} else {
			sb.WriteString("/")
			sb.WriteString(segment)
		}
	}
	if sb.Len() == 0 {
		return "/"
	}
	return sb.String()
}

func processPathParams(pathTemplate string, req *Request, operation *openapi3.Operation) {
	segments := strings.Split(pathTemplate, "/")
	for _, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			var defaultValue interface{}
			for _, variable := range req.URL.Query { // Assuming 'Query' is the correct field
				if variable.Key == paramName {
					defaultValue = variable.Value
					break
				}
			}
			param := &openapi3.Parameter{
				Name:        paramName,
				In:          "path",
				Description: "Path parameter " + paramName,
				Required:    true,
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:    &openapi3.Types{"string"},
						Default: defaultValue,
					},
				},
			}
			operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
		}
	}
}

func processResponses(responses []Response, operation *openapi3.Operation) {
	for _, resp := range responses {
		statusCode := strconv.Itoa(resp.Code)
		description := resp.Name
		response := &openapi3.Response{
			Description: &description,
			Content:     openapi3.NewContent(),
		}

		contentType := "text/plain"
		for _, header := range resp.Header {
			if strings.ToLower(header.Key) == "content-type" {
				contentType = header.Value
				break
			}
		}

		if resp.Body != "" {
			schema := &openapi3.Schema{Type: &openapi3.Types{"string"}}
			if strings.Contains(contentType, "json") {
				var jsonData interface{}
				if err := json.Unmarshal([]byte(resp.Body), &jsonData); err == nil {
					schema = inferSchemaFromJSON(jsonData)
				} else {
					log.Printf("Warning: failed to parse response JSON: %v", err)
				}
			}
			response.Content[contentType] = &openapi3.MediaType{
				Schema:  &openapi3.SchemaRef{Value: schema},
				Example: resp.Body,
			}
		}
		operation.Responses.Set(statusCode, &openapi3.ResponseRef{Value: response})
	}
}

func sanitizeOperationID(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	words := strings.Split(sb.String(), "_")
	var filteredWords []string
	for _, word := range words {
		if word != "" {
			filteredWords = append(filteredWords, word)
		}
	}
	return strings.Join(filteredWords, "_")
}
