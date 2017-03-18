package urlutil

import (
	"fmt"
	"strings"
)

// MakeURL make a url for some abbreviation addr like ":12345"
func MakeURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return fmt.Sprintf("http://localhost%s", addr)
	}

	return addr
}
