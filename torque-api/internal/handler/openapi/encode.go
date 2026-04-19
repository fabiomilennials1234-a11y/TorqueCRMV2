package openapi

import "encoding/json"

// jsonEncode centralizes the JSON encoding of the parsed YAML tree. Split
// into its own file so the encoding/json import stays adjacent to the call
// site a reviewer expects.
func jsonEncode(v any) ([]byte, error) {
	return json.Marshal(v)
}
