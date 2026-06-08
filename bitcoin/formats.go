package bitcoin

import (
	"fmt"
	"strings"
)

type FormatMask struct {
	LegacyCompressed   bool
	LegacyUncompressed bool
	Segwit             bool
	P2SH               bool
	Taproot            bool
}

func AllFormats() FormatMask {
	return FormatMask{
		LegacyCompressed:   true,
		LegacyUncompressed: true,
		Segwit:             true,
		P2SH:               true,
		Taproot:            true,
	}
}

func (m FormatMask) Any() bool {
	return m.LegacyCompressed || m.LegacyUncompressed || m.Segwit || m.P2SH || m.Taproot
}

func (m FormatMask) NeedsCompressedPubkeyHash() bool {
	return m.LegacyCompressed || m.Segwit || m.P2SH
}

func ParseFormatMask(value string) (FormatMask, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "all" {
		return AllFormats(), nil
	}

	var mask FormatMask
	for _, tok := range strings.Split(value, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		switch tok {
		case "legacy":
			mask.LegacyCompressed = true
			mask.LegacyUncompressed = true
		case "legacy-compressed":
			mask.LegacyCompressed = true
		case "legacy-uncompressed":
			mask.LegacyUncompressed = true
		case "segwit":
			mask.Segwit = true
		case "p2sh":
			mask.P2SH = true
		case "taproot":
			mask.Taproot = true
		case "all":
			return AllFormats(), nil
		default:
			return FormatMask{}, fmt.Errorf("unknown format %q (want legacy, legacy-compressed, legacy-uncompressed, segwit, p2sh, taproot, all)", tok)
		}
	}

	if !mask.Any() {
		return FormatMask{}, fmt.Errorf("--formats requires at least one format")
	}
	return mask, nil
}

func (m FormatMask) String() string {
	if m == AllFormats() {
		return "all"
	}
	var parts []string
	if m.LegacyCompressed && m.LegacyUncompressed {
		parts = append(parts, "legacy")
	} else {
		if m.LegacyCompressed {
			parts = append(parts, "legacy-compressed")
		}
		if m.LegacyUncompressed {
			parts = append(parts, "legacy-uncompressed")
		}
	}
	if m.Segwit {
		parts = append(parts, "segwit")
	}
	if m.P2SH {
		parts = append(parts, "p2sh")
	}
	if m.Taproot {
		parts = append(parts, "taproot")
	}
	return strings.Join(parts, ",")
}

func InjectBucketAllowed(mask FormatMask, bucket string) bool {
	switch bucket {
	case "legacy":
		return mask.LegacyCompressed
	case "legacy-uncompressed":
		return mask.LegacyUncompressed
	case "segwit":
		return mask.Segwit
	case "p2sh":
		return mask.P2SH
	case "taproot":
		return mask.Taproot
	default:
		return false
	}
}