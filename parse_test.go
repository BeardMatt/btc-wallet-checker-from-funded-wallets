package main

import "testing"

func TestParseTSVLine(t *testing.T) {
	cases := []struct {
		line        string
		wantAddr    string
		wantBalance int
		wantOK      bool
	}{
		{"34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo\t24859759051607", "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo", 24859759051607, true},
		{"bc1ql49ydapnjafl5t2cp9zqpjwe6pdgmxy98859v2\t14006298532556", "bc1ql49ydapnjafl5t2cp9zqpjwe6pdgmxy98859v2", 14006298532556, true},
		{"34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo\t24859759051607\n", "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo", 24859759051607, true},
		{"no-tab-line", "", 0, false},
		{"addr\tnotdigits", "", 0, false},
	}
	for _, tc := range cases {
		addr, bal, ok := parseTSVLine([]byte(tc.line))
		if ok != tc.wantOK {
			t.Fatalf("line %q: ok=%v want %v", tc.line, ok, tc.wantOK)
		}
		if !tc.wantOK {
			continue
		}
		if string(addr) != tc.wantAddr || bal != tc.wantBalance {
			t.Fatalf("line %q: got addr=%q bal=%d", tc.line, addr, bal)
		}
	}
}