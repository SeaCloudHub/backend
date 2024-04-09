package pagination

import "github.com/SeaCloudHub/backend/pkg/util"

type Pager struct {
	Page  int
	Limit int
	total int64
}

type PageInfo struct {
	TotalItems   int64 `json:"total_items"`
	TotalPages   int   `json:"total_pages"`
	CurrentPage  int   `json:"current_page"`
	NextPage     *int  `json:"next_page"`
	PreviousPage *int  `json:"previous_page"`
	FirstPage    int   `json:"first_page"`
	LastPage     int   `json:"last_page"`
	Limit        int   `json:"limit"`
}

func NewPager(page int, limit int) *Pager {
	return &Pager{
		Page:  page,
		Limit: limit,
	}
}

// Do returns the offset and limit
func (p *Pager) Do() (int, int) {
	offset := (p.Page - 1) * p.Limit

	return offset, p.Limit
}

func (p *Pager) SetTotal(total int64) {
	p.total = total
}

func (p *Pager) PageInfo() PageInfo {
	totalPages := ceil(p.total, int64(p.Limit))

	var nextPage *int
	if p.Page < totalPages {
		nextPage = util.Int(p.Page + 1)
	}

	var previousPage *int
	if p.Page > 1 {
		previousPage = util.Int(p.Page - 1)
	}

	return PageInfo{
		TotalItems:   p.total,
		TotalPages:   totalPages,
		CurrentPage:  p.Page,
		NextPage:     nextPage,
		PreviousPage: previousPage,
		FirstPage:    1,
		LastPage:     totalPages,
		Limit:        p.Limit,
	}
}

func ceil(a, b int64) int {
	return int((a + b - 1) / b)
}
