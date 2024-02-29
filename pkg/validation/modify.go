package validation

import (
	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
)

var (
	modify *mold.Transformer
)

func init() {
	modify = modifiers.New()
}

func Modify() *mold.Transformer {
	return modify
}
