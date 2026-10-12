package utils

import (
	"encoding/json"
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
)

func DecodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return apperror.BadRequest("invalid JSON body")
	}
	return nil
}
