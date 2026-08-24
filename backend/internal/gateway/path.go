package gateway

import "strings"

// StripInvokePrefix maps /api/v1/run/{name}/foo?x=1 onto the user-visible path /foo.
func StripInvokePrefix(fullPath, functionName string) string {
	prefix := "/api/v1/run/" + functionName
	if !strings.HasPrefix(fullPath, prefix) {
		if fullPath == "" {
			return "/"
		}
		return fullPath
	}
	rest := strings.TrimPrefix(fullPath, prefix)
	if rest == "" {
		return "/"
	}
	if !strings.HasPrefix(rest, "/") {
		return "/" + rest
	}
	return rest
}
