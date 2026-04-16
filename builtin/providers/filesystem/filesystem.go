package filesystem

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/braind/braind/internal/interfaces"
)

type Provider struct{}

func (p *Provider) Name() string                                                      { return "filesystem" }
func (p *Provider) Ingest(ctx context.Context, db *sql.DB, cfg json.RawMessage) error { return nil }
func (p *Provider) Sources() ([]interfaces.ProviderSource, error)                     { return nil, nil }
