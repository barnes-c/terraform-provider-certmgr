package certMgr

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

type Client struct {
	HTTPClient *spnego.Client
	Host       string
	Port       int
}

func loadKrb5Config() (*config.Config, error) {
	path := os.Getenv("KRB5_CONFIG")
	if path == "" {
		path = "/etc/krb5.conf" // Default path for krb5.conf
	}
	return config.Load(path)
}

func loadCCache() (*credentials.CCache, error) {
	ccachePath := os.Getenv("KRB5CCNAME")
	if ccachePath == "" {
		return nil, fmt.Errorf("KRB5CCNAME environment variable not set")
	}

	ccachePath = strings.TrimPrefix(ccachePath, "FILE:")
	return credentials.LoadCCache(ccachePath)
}

func NewClient(host, port string) (*Client, error) {
	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 || p > 65535 {
		return nil, fmt.Errorf("invalid port: %q", port)
	}

	krbConf, err := loadKrb5Config()
	if err != nil {
		return nil, fmt.Errorf("failed to load krb5.conf: %w", err)
	}

	ccache, err := loadCCache()
	if err != nil {
		return nil, fmt.Errorf("failed to load credential cache: %w", err)
	}

	krbClient, err := client.NewFromCCache(ccache, krbConf)
	if err != nil {
		return nil, fmt.Errorf("failed to create kerberos client: %w", err)
	}

	fqdn, err := resolveFQDN(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve fqdn for host %q: %w", host, err)
	}

	httpClient := spnego.NewClient(krbClient, nil, "")

	return &Client{
		Host:       fqdn,
		Port:       p,
		HTTPClient: httpClient,
	}, nil
}

func resolveFQDN(host string) (string, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve IP for hostname %s: %w", host, err)
	}

	for _, ip := range ips {
		if ip.To4() == nil {
			continue
		}
		ptrs, err := net.LookupAddr(ip.String())
		if err != nil {
			return "", fmt.Errorf("reverse lookup failed for IP %s: %w", ip, err)
		}
		if len(ptrs) > 0 {
			return strings.TrimSuffix(ptrs[0], "."), nil
		}
	}
	return "", fmt.Errorf("no valid IPv4 PTR record found for host %s", host)
}

func (c *Client) doRequest(method, url string, payload []byte) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", cerr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}
