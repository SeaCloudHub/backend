package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
)

func init() {
	validate = newValidate()
}

func newValidate() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	// register function to get tag name from json tags.
	validate.RegisterTagNameFunc(jsonTagName)

	return validate
}

func Validate() *validator.Validate {
	return validate
}

func jsonTagName(field reflect.StructField) string {
	name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}

	return name
}
