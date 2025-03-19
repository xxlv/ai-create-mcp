package postman

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestConvertToOpenAPI(t *testing.T) {
	t.Run("Basic collection conversion", func(t *testing.T) {
		collection := Collection{
			Info: Info{
				Name:        "Test API",
				Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
				Description: "Test API description",
			},
			Item: []Item{
				{
					Name: "Get Users",
					Request: &Request{
						Method: "GET",
						URL: URL{
							Raw:      "https://api.example.com/users",
							Protocol: "https",
							Host:     []string{"api", "example", "com"},
							Path:     []string{"users"},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		assert.Equal(t, "3.0.0", oas.OpenAPI)
		assert.Equal(t, "Test API", oas.Info.Title)
		assert.Equal(t, "Test API description", oas.Info.Description)
		assert.NotNil(t, oas.Paths.Find("/users"))
		assert.NotNil(t, oas.Paths.Find("/users").Get)
	})

	// Test with folders
	t.Run("Collection with folders", func(t *testing.T) {
		collection := Collection{
			Info: Info{
				Name:   "Test API with Folders",
				Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
			},
			Item: []Item{
				{
					Name: "Users",
					Item: []Item{
						{
							Name: "Get All Users",
							Request: &Request{
								Method: "GET",
								URL: URL{
									Raw:  "https://api.example.com/users",
									Path: []string{"users"},
								},
							},
						},
						{
							Name: "Get User by ID",
							Request: &Request{
								Method: "GET",
								URL: URL{
									Raw:  "https://api.example.com/users/:id",
									Path: []string{"users", ":id"},
								},
							},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		assert.Equal(t, "Test API with Folders", oas.Info.Title)
		assert.NotNil(t, oas.Paths.Find("/users"))
		assert.NotNil(t, oas.Paths.Find("/users/{id}"))
		assert.Equal(t, 1, len(oas.Tags))
		assert.Equal(t, "Users", oas.Tags[0].Name)
	})

	// Test with query parameters
	t.Run("Request with query parameters", func(t *testing.T) {
		collection := Collection{
			Info: Info{
				Name: "Test API with Query Params",
			},
			Item: []Item{
				{
					Name: "Get Users with Filter",
					Request: &Request{
						Method: "GET",
						URL: URL{
							Raw:  "https://api.example.com/users?role=admin&active=true",
							Path: []string{"users"},
							Query: []Query{
								{Key: "role", Value: "admin"},
								{Key: "active", Value: "true"},
							},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		pathItem := oas.Paths.Find("/users")
		assert.NotNil(t, pathItem)
		assert.NotNil(t, pathItem.Get)
		assert.Equal(t, 2, len(pathItem.Get.Parameters))
		assert.Equal(t, "role", pathItem.Get.Parameters[0].Value.Name)
		assert.Equal(t, "active", pathItem.Get.Parameters[1].Value.Name)
	})

	// Test with headers
	t.Run("Request with headers", func(t *testing.T) {
		collection := Collection{
			Info: Info{
				Name: "Test API with Headers",
			},
			Item: []Item{
				{
					Name: "Get Users with Auth",
					Request: &Request{
						Method: "GET",
						URL: URL{
							Raw:  "https://api.example.com/users",
							Path: []string{"users"},
						},
						Header: []Header{
							{Key: "Authorization", Value: "Bearer token123"},
							{Key: "Accept", Value: "application/json"},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		pathItem := oas.Paths.Find("/users")
		assert.NotNil(t, pathItem)
		assert.NotNil(t, pathItem.Get)
		assert.Equal(t, 2, len(pathItem.Get.Parameters))
		assert.Equal(t, "Authorization", pathItem.Get.Parameters[0].Value.Name)
		assert.Equal(t, "Accept", pathItem.Get.Parameters[1].Value.Name)
	})

	// Test with request body
	t.Run("Request with JSON body", func(t *testing.T) {
		jsonBody := `{"name":"John Doe","email":"john@example.com"}`
		collection := Collection{
			Info: Info{
				Name: "Test API with Request Body",
			},
			Item: []Item{
				{
					Name: "Create User",
					Request: &Request{
						Method: "POST",
						URL: URL{
							Raw:  "https://api.example.com/users",
							Path: []string{"users"},
						},
						Header: []Header{
							{Key: "Content-Type", Value: "application/json"},
						},
						Body: &Body{
							Mode: "raw",
							Raw:  jsonBody,
							Options: &BodyOptions{
								Raw: &RawOptions{
									Language: "json",
								},
							},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		pathItem := oas.Paths.Find("/users")
		assert.NotNil(t, pathItem)
		assert.NotNil(t, pathItem.Post)
		assert.NotNil(t, pathItem.Post.RequestBody)
		assert.NotNil(t, pathItem.Post.RequestBody.Value.Content["application/json"])
		assert.Equal(t, jsonBody, pathItem.Post.RequestBody.Value.Content["application/json"].Example)
	})

	// Test with path parameters
	t.Run("Request with path parameters", func(t *testing.T) {
		collection := Collection{
			Info: Info{
				Name: "Test API with Path Params",
			},
			Item: []Item{
				{
					Name: "Get User by ID",
					Request: &Request{
						Method: "GET",
						URL: URL{
							Raw:  "https://api.example.com/users/:id",
							Path: []string{"users", ":id"},
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		pathItem := oas.Paths.Find("/users/{id}")
		assert.NotNil(t, pathItem)
		assert.NotNil(t, pathItem.Get)
		assert.Equal(t, 1, len(pathItem.Get.Parameters))
		assert.Equal(t, "id", pathItem.Get.Parameters[0].Value.Name)
		assert.Equal(t, "path", pathItem.Get.Parameters[0].Value.In)
		assert.True(t, pathItem.Get.Parameters[0].Value.Required)
	})

	t.Run("Request with example responses", func(t *testing.T) {
		responseBody := `{"id":1,"name":"John Doe","email":"john@example.com"}`
		collection := Collection{
			Info: Info{
				Name: "Test API with Responses",
			},
			Item: []Item{
				{
					Name: "Get User",
					Request: &Request{
						Method: "GET",
						URL: URL{
							Raw:  "https://api.example.com/users/1",
							Path: []string{"users", "1"},
						},
					},
					Response: []Response{
						{
							Name:   "Successful Response",
							Status: "OK",
							Code:   200,
							Header: []Header{
								{Key: "Content-Type", Value: "application/json"},
							},
							Body: responseBody,
						},
					},
				},
			},
		}

		oas := convertToOpenAPI(&collection)

		pathItem := oas.Paths.Find("/users/1")
		assert.NotNil(t, pathItem)
		assert.NotNil(t, pathItem.Get)
		assert.NotNil(t, pathItem.Get.Responses)

		// 使用Value方法而不是Get方法
		resp200 := pathItem.Get.Responses.Value("200")
		assert.NotNil(t, resp200)
		assert.Equal(t, "Successful Response", *resp200.Value.Description)
		assert.NotNil(t, resp200.Value.Content["application/json"])
		assert.Equal(t, responseBody, resp200.Value.Content["application/json"].Example)
	})

}

func TestInferSchemaFromJSON(t *testing.T) {
	t.Run("Infer schema from JSON object", func(t *testing.T) {
		jsonStr := `{
			"id": 1,
			"name": "John Doe",
			"email": "john@example.com",
			"active": true,
			"tags": ["user", "admin"],
			"address": {
				"street": "123 Main St",
				"city": "Anytown"
			}
		}`

		var data interface{}
		err := json.Unmarshal([]byte(jsonStr), &data)
		assert.NoError(t, err)

		schema := inferSchemaFromJSON(data)

		assert.Equal(t, openapi3.Types{"object"}, *schema.Type)
		assert.NotNil(t, schema.Properties["id"])
		assert.Equal(t, openapi3.Types{"integer"}, *schema.Properties["id"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Properties["name"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Properties["email"].Value.Type)
		assert.Equal(t, openapi3.Types{"boolean"}, *schema.Properties["active"].Value.Type)
		assert.Equal(t, openapi3.Types{"array"}, *schema.Properties["tags"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Properties["tags"].Value.Items.Value.Type)
		assert.Equal(t, openapi3.Types{"object"}, *schema.Properties["address"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Properties["address"].Value.Properties["street"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Properties["address"].Value.Properties["city"].Value.Type)
	})

	t.Run("Infer schema from JSON array", func(t *testing.T) {
		jsonStr := `[
			{"id": 1, "name": "John"},
			{"id": 2, "name": "Alice"}
		]`

		var data interface{}
		err := json.Unmarshal([]byte(jsonStr), &data)
		assert.NoError(t, err)

		schema := inferSchemaFromJSON(data)

		assert.Equal(t, openapi3.Types{"array"}, *schema.Type)
		assert.NotNil(t, schema.Items)
		assert.Equal(t, openapi3.Types{"object"}, *schema.Items.Value.Type)
		assert.Equal(t, openapi3.Types{"integer"}, *schema.Items.Value.Properties["id"].Value.Type)
		assert.Equal(t, openapi3.Types{"string"}, *schema.Items.Value.Properties["name"].Value.Type)
	})
}

func TestSanitizeOperationID(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Get Users", "Get_Users"},
		{"Create User (POST)", "Create_User_POST"},
		{"Delete-User", "Delete_User"},
		{"user profile info", "user_profile_info"},
		{"123 Test", "123_Test"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeOperationID(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertPathParams(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/{id}"},
		{"/users/:id/profile", "/users/{id}/profile"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/users", "/users"},
		{"/users/:id/:action", "/users/{id}/{action}"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := convertPathParams(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProcessRequestHeaders(t *testing.T) {
	req := &Request{
		Header: []Header{
			{Key: "Authorization", Value: "Bearer token123"},
			{Key: "Accept", Value: "application/json"},
			{Key: "Content-Type", Value: "application/json"},
		},
	}

	operation := &openapi3.Operation{
		Parameters: []*openapi3.ParameterRef{},
	}

	processRequestHeaders(req, operation)

	// Should have 2 headers (Content-Type is skipped as it's handled separately)
	assert.Equal(t, 2, len(operation.Parameters))
	assert.Equal(t, "Authorization", operation.Parameters[0].Value.Name)
	assert.Equal(t, "Accept", operation.Parameters[1].Value.Name)
}

func TestProcessQueryParams(t *testing.T) {
	req := &Request{
		URL: URL{
			Query: []Query{
				{Key: "page", Value: "1"},
				{Key: "limit", Value: "10"},
				{Key: "sort", Value: "name"},
			},
		},
	}

	operation := &openapi3.Operation{
		Parameters: []*openapi3.ParameterRef{},
	}

	processQueryParams(req, operation)

	assert.Equal(t, 3, len(operation.Parameters))
	assert.Equal(t, "page", operation.Parameters[0].Value.Name)
	assert.Equal(t, "limit", operation.Parameters[1].Value.Name)
	assert.Equal(t, "sort", operation.Parameters[2].Value.Name)
	assert.Equal(t, "query", operation.Parameters[0].Value.In)
}

func TestProcessPathParams(t *testing.T) {
	pathTemplate := "/users/{id}/posts/{postId}"
	req := &Request{} // Create a dummy Request object as required
	operation := &openapi3.Operation{
		Parameters: []*openapi3.ParameterRef{},
	}

	processPathParams(pathTemplate, req, operation)

	assert.Equal(t, 2, len(operation.Parameters))
	assert.Equal(t, "id", operation.Parameters[0].Value.Name)
	assert.Equal(t, "postId", operation.Parameters[1].Value.Name)
	assert.Equal(t, "path", operation.Parameters[0].Value.In)
	assert.True(t, operation.Parameters[0].Value.Required)
}

func TestProcessRequestBody(t *testing.T) {
	t.Run("JSON body", func(t *testing.T) {
		jsonBody := `{"name":"John Doe","email":"john@example.com"}`
		req := &Request{
			Body: &Body{
				Mode: "raw",
				Raw:  jsonBody,
				Options: &BodyOptions{
					Raw: &RawOptions{
						Language: "json",
					},
				},
			},
			Header: []Header{
				{Key: "Content-Type", Value: "application/json"},
			},
		}

		operation := &openapi3.Operation{}

		processRequestBody(req, operation)

		assert.NotNil(t, operation.RequestBody)
		assert.NotNil(t, operation.RequestBody.Value.Content["application/json"])
		assert.Equal(t, jsonBody, operation.RequestBody.Value.Content["application/json"].Example)
		assert.Equal(t, openapi3.Types{"object"}, *operation.RequestBody.Value.Content["application/json"].Schema.Value.Type)
	})

	t.Run("Plain text body", func(t *testing.T) {
		textBody := "Hello, world!"
		req := &Request{
			Body: &Body{
				Mode: "raw",
				Raw:  textBody,
			},
			Header: []Header{
				{Key: "Content-Type", Value: "text/plain"},
			},
		}

		operation := &openapi3.Operation{}

		processRequestBody(req, operation)

		assert.NotNil(t, operation.RequestBody)
		assert.NotNil(t, operation.RequestBody.Value.Content["text/plain"])
		assert.Equal(t, textBody, operation.RequestBody.Value.Content["text/plain"].Example)
		assert.Equal(t, openapi3.Types{"string"}, *operation.RequestBody.Value.Content["text/plain"].Schema.Value.Type)
	})
}

func TestProcessResponses(t *testing.T) {
	t.Run("JSON response", func(t *testing.T) {
		jsonBody := `{"id":1,"name":"John Doe"}`
		responses := []Response{
			{
				Name:   "Successful Response",
				Status: "OK",
				Code:   200,
				Header: []Header{
					{Key: "Content-Type", Value: "application/json"},
				},
				Body: jsonBody,
			},
			{
				Name:   "Not Found",
				Status: "Not Found",
				Code:   404,
				Body:   `{"error":"User not found"}`,
			},
		}

		operation := &openapi3.Operation{
			Responses: openapi3.NewResponses(),
		}

		processResponses(responses, operation)

		resp200 := operation.Responses.Value("200")
		resp404 := operation.Responses.Value("404")

		assert.NotNil(t, resp200)
		assert.NotNil(t, resp404)
		assert.Equal(t, "Successful Response", *resp200.Value.Description)
		assert.Equal(t, "Not Found", *resp404.Value.Description)
		assert.NotNil(t, resp200.Value.Content["application/json"])
		assert.Equal(t, jsonBody, resp200.Value.Content["application/json"].Example)
	})
}
