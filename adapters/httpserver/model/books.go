package model

import "errors"

type CreateBookRequest struct {
	ISBN string `json:"isbn"`
	Name string `json:"name"`
}

func (r CreateBookRequest) Validate() error {
	if r.ISBN == "" || r.Name == "" {
		return errors.New("invalid request")
	}
	return nil
}
