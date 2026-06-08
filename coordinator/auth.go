package coordinator

import (
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"
)

func bearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "auth not configured", http.StatusInternalServerError)
			return
		}
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		got := strings.TrimPrefix(hdr, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func tlsConfig(cfg Config) (*tls.Config, error) {
	if cfg.MTLSCA == "" && !cfg.MTLSRequireClientCert {
		return &tls.Config{MinVersion: tls.VersionTLS12}, nil
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if cfg.MTLSCA != "" {
		caPEM, err := os.ReadFile(cfg.MTLSCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errInvalidCA
		}
		tlsCfg.ClientCAs = pool
	}
	if cfg.MTLSRequireClientCert {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsCfg, nil
}

var errInvalidCA = &apiError{msg: "invalid mTLS CA", code: 500}