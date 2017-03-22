package urlutil

import (
	"fmt"
	"strings"
)

// MakeURL make a url for some abbreviation addr like ":12345"
func MakeURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("http://localhost%s", addr)
	}

	return strings.TrimSuffix(addr, "/")
}
