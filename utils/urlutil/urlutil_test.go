package urlutil

import "testing"

var urlTable = []struct {
	addr   string
	expect string
}{
	{
		addr:   ":5501",
		expect: "http://127.0.0.1:5501",
	},
	{
		addr:   "127.0.0.1:5502",
		expect: "http://127.0.0.1:5502",
	},
}

func TestMakeURL(t *testing.T) {
	for _, item := range urlTable {
		got := MakeURL(item.addr)
		if got != item.expect {
			t.Errorf("MakeURL failed, expect: %v, got: %v\n", item.expect, got)
		}
	}
}
