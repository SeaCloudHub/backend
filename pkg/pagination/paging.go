package pagination

import (
	"github.com/SeaCloudHub/backend/pkg/validation"
	"strings"
)

type Paging struct {
	Limit      int64  `json:"limit" query:"limit"`
	Cursor     string `json:"cursor" query:"cursor"`
	NextCursor string `json:"next_cursor" swaggerignore:"true"`
}

func (p *Paging) Validate() error {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	p.Cursor = strings.TrimSpace(p.Cursor)

	return validation.Validate().Struct(p)
}
