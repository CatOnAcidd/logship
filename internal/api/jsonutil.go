package api

import (
	"encoding/json"
	"io"
)

func jsonEnc(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
