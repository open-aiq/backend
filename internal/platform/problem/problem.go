package problem

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

const docsBase = "https://docs.air-iq.net/reference/errors/#"

type Kind struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Description string `json:"description"`
}

var (
	MalformedRequest = Kind{"malformed-request", "Malformed request", http.StatusBadRequest, "The request could not be decoded."}
	Unauthorized     = Kind{"unauthorized", "Unauthorized", http.StatusUnauthorized, "Valid credentials are required."}
	NotFound         = Kind{"not-found", "Resource not found", http.StatusNotFound, "The requested resource was not found."}
	MethodNotAllowed = Kind{"method-not-allowed", "Method not allowed", http.StatusMethodNotAllowed, "The request method is not supported for this resource."}
	NoReadings       = Kind{"no-readings", "No readings available", http.StatusNotFound, "The device has not reported any readings."}
	ContentTooLarge  = Kind{"content-too-large", "Request content too large", http.StatusRequestEntityTooLarge, "The request body exceeds the accepted size."}
	UnsupportedMedia = Kind{"unsupported-media-type", "Unsupported media type", http.StatusUnsupportedMediaType, "The request uses an unsupported media type."}
	Validation       = Kind{"validation-error", "Request validation failed", http.StatusUnprocessableEntity, "One or more request values are invalid."}
	Internal         = Kind{"internal-error", "Internal server error", http.StatusInternalServerError, "The server could not complete the request."}
)

func Catalog() []Kind {
	return []Kind{MalformedRequest, Unauthorized, NotFound, MethodNotAllowed, NoReadings, ContentTooLarge, UnsupportedMedia, Validation, Internal}
}

type Violation struct {
	In     string `json:"in" example:"body" enums:"body,query,path"`
	Name   string `json:"name" example:"name"`
	Code   string `json:"code" example:"required"`
	Detail string `json:"detail" example:"name is required"`
}

type Problem struct {
	Type     string      `json:"type" example:"https://docs.air-iq.net/reference/errors/#validation-error"`
	Title    string      `json:"title" example:"Request validation failed"`
	Status   int         `json:"status" example:"422"`
	Detail   string      `json:"detail" example:"One or more request values are invalid."`
	Instance string      `json:"instance,omitempty" example:"/api/v1/devices"`
	Errors   []Violation `json:"errors,omitempty"`
}

func TypeURL(kind Kind) string { return docsBase + kind.Slug }

func Write(c *gin.Context, kind Kind, detail string, violations ...Violation) {
	if detail == "" {
		detail = kind.Description
	}
	c.Header("Content-Type", "application/problem+json")
	c.JSON(kind.Status, Problem{
		Type: TypeURL(kind), Title: kind.Title, Status: kind.Status,
		Detail: detail, Instance: c.Request.URL.Path, Errors: violations,
	})
}

func Abort(c *gin.Context, kind Kind, detail string, violations ...Violation) {
	Write(c, kind, detail, violations...)
	c.Abort()
}

func WriteHTTP(w http.ResponseWriter, r *http.Request, kind Kind, detail string) {
	if detail == "" {
		detail = kind.Description
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(kind.Status)
	_ = json.NewEncoder(w).Encode(Problem{Type: TypeURL(kind), Title: kind.Title, Status: kind.Status, Detail: detail, Instance: r.URL.Path})
}
