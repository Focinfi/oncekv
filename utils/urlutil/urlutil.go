package urlutil

import (
	"fmt"
	"strings"
)

// MakeURL make a url for some abbreviation addr like ":12345"
func MakeURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("http://127.0.0.1%s", addr)

	} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = fmt.Sprintf("http://%s", addr)
	}

	return strings.TrimSuffix(addr, "/")
}
