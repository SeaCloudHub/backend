package seaweedfs

import (
	"context"
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

func (f *Master) DirStatus(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}

	_, err := f.client.R().SetContext(ctx).SetResult(&result).Get("/dir/status")
	if err != nil {
		return result, fmt.Errorf("get dir status: %w", err)
	}

	return result, nil
}

func (f *Master) VolStatus(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}

	_, err := f.client.R().SetContext(ctx).SetResult(&result).Get("/vol/status")
	if err != nil {
		return result, fmt.Errorf("get vol status: %w", err)
	}

	return result, nil
}
