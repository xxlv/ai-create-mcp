package postman

import (
	"fmt"
	urlPkg "net/url"
	"regexp"
	"strings"

	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
)

func convert(postmanCollection *CollectionResponse) (*core.TemplateData, error) {
	if postmanCollection == nil {
		return nil, fmt.Errorf("no postman collection found")
	}
	vars := convertPostmanToTemplateData(*postmanCollection)
	return &vars, nil
}

func convertPostmanToTemplateData(collection CollectionResponse) core.TemplateData {
	// Initialize core.TemplateData structure
	templateData := core.TemplateData{
		MissBaseURL:       true, // Default to true, we'll set it to false if we find server info
		BinaryName:        collection.Collection.Info.Name,
		ServerName:        collection.Collection.Info.Name,
		ServerVersion:     "1.0.0", // Default version
		ServerDescription: collection.Collection.Info.Description,
		ServerDirectory:   convertToValidDirectoryName(collection.Collection.Info.Name),
	}

	// Extract endpoints, resources, and tools from Postman items
	endpoints := make([]string, 0)
	resources := make([]core.Resource, 0)
	tools := make([]core.Tool, 0)

	// Process collection variables for server URL
	baseURL := ""
	for _, variable := range collection.Collection.Variable {
		if variable.Key == "baseUrl" || variable.Key == "base_url" || variable.Key == "host" {
			baseURL = variable.Value
			templateData.MissBaseURL = false
			break
		}
	}

	// Process all items recursively
	processItems(collection.Collection.Item, baseURL, &endpoints, &resources, &tools)

	// Assign the processed data to core.TemplateData
	templateData.Endpoints = endpoints
	templateData.Resources = resources
	templateData.Tools = tools
	templateData.Prompts = []core.Prompt{} // Postman doesn't have an equivalent for Prompts

	return templateData
}

// Helper function to process items recursively
func processItems(items []Item, baseURL string, endpoints *[]string, resources *[]core.Resource, tools *[]core.Tool) {
	for _, item := range items {
		// If item has nested items, it's a folder
		if len(item.Item) > 0 {
			processItems(item.Item, baseURL, endpoints, resources, tools)
			continue
		}

		// Skip items without requests
		if item.Request == nil {
			continue
		}

		// Add endpoint
		endpoint := item.Request.URL.Raw
		*endpoints = append(*endpoints, endpoint)

		// If it has a body and it's a GET request, it might be a resource
		if item.Request.Body != nil && item.Request.Method == "GET" {
			mimeType := "application/json" // Default MIME type

			// Try to determine MIME type from body options
			if item.Request.Body.Options != nil && item.Request.Body.Options.Raw != nil {
				language := item.Request.Body.Options.Raw.Language
				if language == "json" {
					mimeType = "application/json"
				} else if language == "xml" {
					mimeType = "application/xml"
				}
				// Add more MIME type mappings as needed
			}

			resource := core.Resource{
				Name:        item.Name,
				Description: item.Name, // Using name as description if no dedicated description field exists
				URI:         item.Request.URL.Raw,
				MimeType:    mimeType,
			}
			*resources = append(*resources, resource)
		}

		// For non-GET requests, create a Tool
		if item.Request.Method != "GET" {
			tool := core.Tool{
				Name:        item.Name,
				Description: item.Name, // Using name as description
				Method:      item.Request.Method,
				Path:        endpoint,
				Arguments:   extractArguments(item.Request),
			}
			*tools = append(*tools, tool)
		}
	}
}

// Helper function to extract Arguments from Request
func extractArguments(request *Request) []core.Argument {
	arguments := make([]core.Argument, 0)

	// Add URL query parameters as arguments
	for _, query := range request.URL.Query {
		arg := core.Argument{
			Name:        query.Key,
			Description: "URL parameter: " + query.Key,
			Required:    false, // Defaulting to false as Postman doesn't indicate if a parameter is required
		}
		arguments = append(arguments, arg)
	}

	// If the request has a body, try to extract fields from JSON
	if request.Body != nil && request.Body.Mode == "raw" && request.Body.Raw != "" {
		// If we have JSON body options, treat it as a JSON object and extract fields
		if request.Body.Options != nil && request.Body.Options.Raw != nil &&
			request.Body.Options.Raw.Language == "json" {
			bodyArgs := extractArgumentsFromJSONBody(request.Body.Raw)
			arguments = append(arguments, bodyArgs...)
		}
	}

	return arguments
}

// Helper function to extract arguments from JSON body
func extractArgumentsFromJSONBody(jsonBody string) []core.Argument {
	arguments := make([]core.Argument, 0)

	// In a real implementation, you would parse the JSON and extract field names
	// Here's a simplified placeholder approach
	// Note: Actual implementation would require proper JSON parsing

	// For this simplified version, we'll just return a placeholder argument
	// indicating that the entire JSON body is expected
	arg := core.Argument{
		Name:        "body",
		Description: "Request body",
		Required:    true,
	}
	arguments = append(arguments, arg)

	return arguments
}

// Helper function to get an endpoint from URL
func getEndpoint(url URL) string {
	if len(url.Path) > 0 {
		return "/" + strings.Join(url.Path, "/")
	}

	// If no path segments, try to extract path from raw URL
	if url.Raw != "" {
		parsedURL, err := urlPkg.Parse(url.Raw)
		if err == nil && parsedURL.Path != "" {
			return parsedURL.Path
		}
	}

	return "/"
}

// Helper function to convert a string to a valid directory name
func convertToValidDirectoryName(name string) string {
	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(strings.ToLower(name), "_")
}
