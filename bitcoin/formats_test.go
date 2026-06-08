package bitcoin

import "testing"

func TestParseFormatMaskAll(t *testing.T) {
	for _, in := range []string{"", "all"} {
		mask, err := ParseFormatMask(in)
		if err != nil {
			t.Fatal(err)
		}
		if mask != AllFormats() {
			t.Fatalf("want all formats for %q: %+v", in, mask)
		}
	}
}

func TestParseFormatMaskTaprootOnly(t *testing.T) {
	mask, err := ParseFormatMask("taproot")
	if err != nil {
		t.Fatal(err)
	}
	if mask.Taproot != true || mask.Any() != true {
		t.Fatal(mask)
	}
	if mask.LegacyCompressed || mask.Segwit || mask.P2SH {
		t.Fatal("expected taproot only", mask)
	}
}

func TestParseFormatMaskInvalid(t *testing.T) {
	if _, err := ParseFormatMask("bc1"); err == nil {
		t.Fatal("expected error")
	}
}