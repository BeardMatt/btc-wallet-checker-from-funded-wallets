package worker

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"btcfind/cluster"
)

type Client struct {
	base    string
	token   string
	http    *http.Client
	backoff time.Duration
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.AuthToken == "" {
		cfg.AuthToken = os.Getenv("BTCFIND_AUTH_TOKEN")
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 30 * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSSkipVerify,
		},
	}
	if cfg.TLSCA != "" {
		caPEM, err := os.ReadFile(cfg.TLSCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("invalid TLS CA")
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}
	if cfg.ProxyURL != "" {
		u, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(u)
	}

	base := strings.TrimRight(cfg.CoordinatorURL, "/")
	return &Client{
		base:  base,
		token: cfg.AuthToken,
		http: &http.Client{
			Timeout:   cfg.DialTimeout + 60*time.Second,
			Transport: transport,
		},
		backoff: time.Second,
	}, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any) (int, error) {
	for {
		code, err := c.tryJSON(ctx, method, path, reqBody, respBody)
		if err == nil {
			c.backoff = time.Second
			return code, nil
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(c.backoff):
		}
		if c.backoff < 60*time.Second {
			c.backoff *= 2
		}
	}
}

func (c *Client) tryJSON(ctx context.Context, method, path string, reqBody any, respBody any) (int, error) {
	var body io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return 0, err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, fmt.Errorf("server error %d", resp.StatusCode)
	}
	if respBody != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return resp.StatusCode, err
		}
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (c *Client) Register(ctx context.Context, hostname string, threads int, version string) (cluster.RegisterResponse, error) {
	var resp cluster.RegisterResponse
	_, err := c.doJSON(ctx, http.MethodPost, "/api/v1/register", cluster.RegisterRequest{
		Hostname: hostname,
		Threads:  threads,
		Version:  version,
	}, &resp)
	return resp, err
}

func (c *Client) Heartbeat(ctx context.Context, workerID, state string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/api/v1/heartbeat", cluster.HeartbeatRequest{
		WorkerID: workerID,
		State:    state,
	}, nil)
	return err
}

func (c *Client) RunStatus(ctx context.Context) (cluster.RunResponse, error) {
	var resp cluster.RunResponse
	_, err := c.doJSON(ctx, http.MethodGet, "/api/v1/run", nil, &resp)
	return resp, err
}

func (c *Client) PostStats(ctx context.Context, req cluster.StatsRequest) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/api/v1/stats", req, nil)
	return err
}

func (c *Client) PostHit(ctx context.Context, req cluster.HitRequest) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/api/v1/hit", req, nil)
	return err
}

func (c *Client) DownloadCache(ctx context.Context, destPath, etag string) (string, error) {
	for {
		newETag, err := c.tryDownloadCache(ctx, destPath, etag)
		if err == nil {
			c.backoff = time.Second
			return newETag, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(c.backoff):
		}
		if c.backoff < 60*time.Second {
			c.backoff *= 2
		}
	}
}

func (c *Client) tryDownloadCache(ctx context.Context, destPath, etag string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/v1/cache", nil)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return etag, nil
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("cache download http %d", resp.StatusCode)
	}
	newETag := resp.Header.Get("ETag")
	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, resp.Body)
	closeErr := f.Close()
	if err != nil {
		os.Remove(tmp)
		return "", err
	}
	if closeErr != nil {
		os.Remove(tmp)
		return "", closeErr
	}
	if err := os.Rename(tmp, destPath); err != nil {
		return "", err
	}
	return newETag, nil
}