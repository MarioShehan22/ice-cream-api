package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = func() *validator.Validate {
	v := validator.New()
	// report errors using the JSON field names
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	return v
}()

// Struct returns nil when valid, otherwise a map of field -> message.
func Struct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return map[string]string{"_": err.Error()}
	}
	out := make(map[string]string, len(ves))
	for _, fe := range ves {
		out[fe.Field()] = message(fe)
	}
	return out
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s (characters/value)", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s (characters/value)", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	default:
		return "is invalid"
	}
}
