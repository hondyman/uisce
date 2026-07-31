package rulefabric

import "encoding/json"

// jsonMarshal / jsonUnmarshal are thin wrappers to keep vm_manager.go's
// dependency surface tight. Tests can stub these for deterministic input.
var (
	jsonMarshal   = json.Marshal
	jsonUnmarshal = json.Unmarshal
)