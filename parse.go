package main

import "bytes"

func parseTSVLine(line []byte) (addr []byte, balance int, ok bool) {
	tab := bytes.IndexByte(line, '\t')
	if tab < 0 {
		return nil, 0, false
	}

	balance, ok = parseIntBytes(line[tab+1:])
	if !ok {
		return nil, 0, false
	}

	return line[:tab], balance, true
}

func parseIntBytes(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}

	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}