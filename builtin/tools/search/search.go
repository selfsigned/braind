package search

import (
	"context"
	"encoding/json"

	"github.com/braind/braind/internal/interfaces"
)

type SearchTool struct{}

func New() *SearchTool { return &SearchTool{} }

func (t *SearchTool) Name() string { return "search" }

func (t *SearchTool) Description() string {
	return "Semantic search over the vault's embedding store. Returns ranked chunks with source file references."
}

func (t *SearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query text"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of results to return",
				"default": 10
			}
		}
	}`)
}

func (t *SearchTool) Execute(ctx context.Context, req interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	return nil, nil
}
