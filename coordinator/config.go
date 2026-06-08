package coordinator

import "btcfind/bitcoin"

// Config holds btcfind-coordinator settings.
type Config struct {
	Listen    string
	TLSCert   string
	TLSKey    string
	AuthToken string
	Insecure  bool

	Formats    bitcoin.FormatMask
	MinBalance uint64

	AutoStart bool
	Quiet     bool
	NoColor   bool
	Verbose   bool

	// mTLS (optional)
	MTLSCA               string
	MTLSRequireClientCert bool

	// Abuse limits (optional)
	MaxWorkers      int
	AllowCIDR       []string
	RateRegister    int // per minute per IP, 0 = default 10
	RateCache       int // per hour per IP, 0 = default 2
	RateAPI         int // per minute per IP, 0 = default 120
}