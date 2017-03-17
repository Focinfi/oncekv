package node

import (
	"fmt"
	"strings"
)

func competeAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return fmt.Sprintf("http://localhost%s", addr)
	}

	return ""
}
