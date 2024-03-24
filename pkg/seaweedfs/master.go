package seaweedfs

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
)

type Master struct {
	host   *url.URL
	client *resty.Client
}

func NewMaster(masterURL string) (*Master, error) {
	u, err := url.Parse(masterURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	return &Master{
		host:   u,
		client: resty.New().SetBaseURL(u.String()),
	}, nil
}

func (m *Master) SetDebug(debug bool) {
	m.client.SetDebug(debug)
}
