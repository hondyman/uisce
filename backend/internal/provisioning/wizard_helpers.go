package provisioning

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func writeJSONStatus(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
