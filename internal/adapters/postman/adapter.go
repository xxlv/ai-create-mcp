package postman

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xxlv/ai-create-mcp/internal/adapters/core"
)

var apiURL = "https://api.getpostman.com/collections/%s"

type PostmanAdapter struct {
	collectionId string
	apiKey       string
}

func New(collectionId string, apiKey string) *PostmanAdapter {
	return &PostmanAdapter{
		collectionId: collectionId,
		apiKey:       apiKey,
	}
}
func (a *PostmanAdapter) ToTemplateData() (*core.TemplateData, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(apiURL, a.collectionId), nil)
	if err != nil {
		return nil, err
	}

	if a.apiKey != "" {
		req.Header.Set("X-API-Key", a.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch collection: %s", resp.Status)
	}

	var collection map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, err
	}
	return convert(collection)

}

func (a *PostmanAdapter) GetSourceType() string {
	return "postman"
}

var _ core.Adapter = new(PostmanAdapter)
