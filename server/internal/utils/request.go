package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
)

const MaxBodyBytes = 1 << 20

func DecodeJSON(r *http.Request, dst any) error {
	err := json.NewDecoder(r.Body).Decode(dst)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return apperror.PayloadTooLarge("request body must be at most 1 MB")
	}
	if err != nil {
		return apperror.BadRequest("invalid JSON body")
	}
	return nil
}
