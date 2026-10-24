package types

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
)

var (
	validate             = newValidator()
	workspaceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{2,62}$`)
	localHosts           = map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonFieldName)
	mustRegister(v, "workspace_name", func(fl validator.FieldLevel) bool {
		return workspaceNamePattern.MatchString(fl.Field().String())
	})
	mustRegister(v, "local_url", func(fl validator.FieldLevel) bool {
		u, err := url.Parse(fl.Field().String())
		return err == nil && localHosts[u.Hostname()]
	})
	// min/max count runes; maxbytes guards byte limits such as bcrypt's 72.
	mustRegister(v, "maxbytes", func(fl validator.FieldLevel) bool {
		limit, err := strconv.Atoi(fl.Param())
		return err == nil && len(fl.Field().String()) <= limit
	})
	return v
}

func mustRegister(v *validator.Validate, tag string, fn validator.Func) {
	if err := v.RegisterValidation(tag, fn); err != nil {
		panic(fmt.Sprintf("registering %q validation: %v", tag, err))
	}
}

func jsonFieldName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

func validateStruct(s any) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return fmt.Errorf("validating request: %w", err)
	}

	messages := make([]string, 0, len(fieldErrs))
	for _, fe := range fieldErrs {
		messages = append(messages, fieldMessage(fe))
	}
	return apperror.BadRequest(strings.Join(messages, "; "))
}

func fieldMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required", "required_if":
		return field + " is required"
	case "excluded_if":
		return field + " is not allowed for this connection_type"
	case "oneof":
		return field + " must be one of: " + fe.Param()
	case "min":
		return field + " must be at least " + fe.Param() + " characters"
	case "maxbytes":
		return field + " must be at most " + fe.Param() + " bytes"
	case "url":
		return field + " must be a valid URL"
	case "local_url":
		return field + " must point to localhost, 127.0.0.1 or ::1"
	case "workspace_name":
		return field + " must be 3-63 letters, digits or hyphens, starting with a letter or digit"
	default:
		return field + " is invalid"
	}
}
