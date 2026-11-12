package cjk

import "testing"

func TestGB1PDFUnicodeIncludesVariantsAndSequences(t *testing.T) {
	for _, tc := range []struct {
		cid  uint32
		want string
	}{
		{17, "0"}, {830, "1"}, {852, "G"}, {2850, "目"},
		{22031, "郎\uFE00"}, {22032, "嗀\uFE00"},
		{22038, "礼\uFE00"}, {22040, "蘒\uFE00"},
	} {
		got, ok := Unicode("GB1", tc.cid)
		if !ok || string(got) != tc.want {
			t.Errorf("CID %d = %q, %v; want %q", tc.cid, got, ok, tc.want)
		}
	}
}
