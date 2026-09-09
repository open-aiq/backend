package input

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"

	"go-aiq-backend/internal/platform/problem"
)

const MaxJSONBodyBytes int64 = 64 << 10

var (
	transformer = modifiers.New()
	validate    = newValidator()
	providerRE  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,49}$`)
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			if name := strings.Split(field.Tag.Get(tag), ",")[0]; name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})
	_ = v.RegisterValidation("provider", func(fl validator.FieldLevel) bool {
		return providerRE.MatchString(fl.Field().String())
	})
	return v
}

func BindJSON(c *gin.Context, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		problem.Write(c, problem.UnsupportedMedia, "Content-Type must be application/json.")
		return false
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxJSONBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeDecodeError(c, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeDecodeError(c, err)
			return false
		}
		problem.Write(c, problem.MalformedRequest, "The request body must contain exactly one JSON value.")
		return false
	}
	return Validate(c, "body", dst)
}

func Validate(c *gin.Context, location string, dst any) bool {
	if err := transformer.Struct(context.Background(), dst); err != nil {
		problem.Write(c, problem.MalformedRequest, "The request values could not be normalized.")
		return false
	}
	if err := validate.Struct(dst); err != nil {
		problem.Write(c, problem.Validation, "One or more request values are invalid.", violations(location, err)...)
		return false
	}
	return true
}

func RejectUnknownQuery(c *gin.Context, allowed ...string) bool {
	known := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		known[key] = struct{}{}
	}
	var errs []problem.Violation
	for key := range c.Request.URL.Query() {
		if _, ok := known[key]; !ok {
			errs = append(errs, problem.Violation{In: "query", Name: key, Code: "unknown", Detail: fmt.Sprintf("%s is not an accepted query parameter", key)})
		}
	}
	if len(errs) != 0 {
		problem.Write(c, problem.Validation, "One or more query parameters are invalid.", errs...)
		return false
	}
	return true
}

func writeDecodeError(c *gin.Context, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		problem.Write(c, problem.ContentTooLarge, fmt.Sprintf("The request body must not exceed %d bytes.", MaxJSONBodyBytes))
		return
	}
	if errors.Is(err, io.EOF) {
		problem.Write(c, problem.MalformedRequest, "The request body must not be empty.")
		return
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		name := typeErr.Field
		if name == "" {
			name = "body"
		}
		problem.Write(c, problem.Validation, "One or more request values are invalid.", problem.Violation{In: "body", Name: name, Code: "invalid_type", Detail: fmt.Sprintf("%s has the wrong JSON type", name)})
		return
	}
	if field, ok := unknownField(err); ok {
		problem.Write(c, problem.Validation, "One or more request values are invalid.", problem.Violation{In: "body", Name: field, Code: "unknown", Detail: fmt.Sprintf("%s is not an accepted field", field)})
		return
	}
	problem.Write(c, problem.MalformedRequest, "The request body contains malformed JSON.")
}

func unknownField(err error) (string, bool) {
	const prefix = "json: unknown field \""
	message := err.Error()
	if !strings.HasPrefix(message, prefix) || !strings.HasSuffix(message, "\"") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(message, prefix), "\""), true
}

func violations(location string, err error) []problem.Violation {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return []problem.Violation{{In: location, Name: "request", Code: "invalid", Detail: "request is invalid"}}
	}
	out := make([]problem.Violation, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		name := fieldErr.Namespace()
		if dot := strings.IndexByte(name, '.'); dot >= 0 {
			name = name[dot+1:]
		}
		out = append(out, problem.Violation{In: location, Name: name, Code: code(fieldErr.Tag()), Detail: message(name, fieldErr)})
	}
	return out
}

func code(tag string) string {
	switch tag {
	case "oneof":
		return "one_of"
	case "datetime", "uuid4", "provider":
		return "format"
	case "required_without_all":
		return "required"
	default:
		return tag
	}
}

func message(name string, err validator.FieldError) string {
	switch err.Tag() {
	case "required", "required_without_all":
		return fmt.Sprintf("%s is required", name)
	case "min":
		return fmt.Sprintf("%s must be at least %s", name, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", name, err.Param())
	case "gte":
		return fmt.Sprintf("%s must be at least %s", name, err.Param())
	case "lte":
		return fmt.Sprintf("%s must be at most %s", name, err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", name, strings.ReplaceAll(err.Param(), " ", ", "))
	case "uuid4":
		return fmt.Sprintf("%s must be a UUIDv4", name)
	case "datetime":
		return fmt.Sprintf("%s must use YYYY-MM-DD", name)
	case "provider":
		return fmt.Sprintf("%s must be a lowercase provider token", name)
	default:
		return fmt.Sprintf("%s is invalid", name)
	}
}
