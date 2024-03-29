package seaweedfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-resty/resty/v2"
)

type Filer struct {
	host   *url.URL
	client *resty.Client
}

func NewFiler(filerURL string) (*Filer, error) {
	u, err := url.Parse(filerURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	return &Filer{
		host:   u,
		client: resty.New().SetBaseURL(u.String()),
	}, nil
}

func (f *Filer) SetDebug(debug bool) {
	f.client.SetDebug(debug)
}

func (f *Filer) GetMetadata(ctx context.Context, in *GetMetadataRequest) (*Entry, error) {
	var result Entry

	resp, err := f.client.R().SetContext(ctx).SetQueryParam("metadata", "true").SetResult(&result).
		Get(in.FullPath)
	if err != nil {
		return nil, fmt.Errorf("get metadata: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrNotFound
	}

	return &result, nil
}

func (f *Filer) ListEntries(ctx context.Context, in *ListEntriesRequest) (*ListEntriesResponse, error) {
	var result ListEntriesResponse

	req := f.client.R().SetContext(ctx)

	if in.Limit > 0 {
		req = req.SetQueryParam("limit", fmt.Sprint(in.Limit))
	}

	if len(in.LastFileName) > 0 {
		req = req.SetQueryParam("lastFileName", in.LastFileName)
	}

	if len(in.NamePattern) > 0 {
		req = req.SetQueryParam("namePattern", in.NamePattern)
	}

	if len(in.NamePatternExclude) > 0 {
		req = req.SetQueryParam("namePatternExclude", in.NamePatternExclude)
	}

	resp, err := req.SetHeader("Accept", "application/json").SetResult(&result).
		Get(in.DirPath)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrNotFound
	}

	return &result, nil
}

func (f *Filer) DownloadFile(ctx context.Context, in *DownloadFileRequest) (io.ReadCloser, error) {
	resp, err := f.client.R().SetContext(ctx).SetDoNotParseResponse(true).
		Get(in.FullPath)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrNotFound
	}

	return resp.RawBody(), nil
}

func (f *Filer) UploadFile(ctx context.Context, in *UploadFileRequest) (*UploadFileResponse, error) {
	var result UploadFileResponse

	resp, err := f.client.R().SetContext(ctx).SetFileReader("file", "", in.Content).
		SetResult(&result).
		Post(in.FullFileName)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return &result, nil
}

func (f *Filer) CreateDirectory(ctx context.Context, in *CreateDirectoryRequest) error {
	resp, err := f.client.R().SetContext(ctx).
		Post(in.DirPath)
	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

func (f *Filer) Delete(ctx context.Context, in *DeleteRequest) error {
	resp, err := f.client.R().SetContext(ctx).
		Delete(in.FullPath)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	return nil
}
