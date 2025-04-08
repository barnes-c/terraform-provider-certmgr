// Copyright (c) Christopher Barnes <christopher@barnes.biz>
// SPDX-License-Identifier: MPL-2.0

package certMgr

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	HTTPClient *http.Client
	Host       string
	Port       int
}

type Certificate struct {
	ID        int    `json:"id"`
	Hostname  string `json:"hostname"`
	Requestor string `json:"requestor"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

func NewClient(host, port *string) (*Client, error) {
	c := &Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Host:       "",   // default
		Port:       8008, // default
	}

	if host != nil && *host != "" {
		c.Host = *host
	}

	if port != nil && *port != "" {
		p, err := strconv.Atoi(*port)
		if err != nil || p <= 0 || p > 65535 {
			return nil, fmt.Errorf("invalid port: %q", *port)
		}
		c.Port = p
	}

	return c, nil
}

func (c *Client) resolveFQDN() (string, error) {
	ips, err := net.LookupIP(c.Host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve IP for hostname %s: %w", c.Host, err)
	}

	for _, ip := range ips {
		if ip.To4() == nil {
			continue
		}

		ptrs, err := net.LookupAddr(ip.String())
		if err != nil {
			return "", fmt.Errorf("reverse lookup failed for IP %s: %w", ip.String(), err)
		}

		if len(ptrs) == 0 {
			return "", fmt.Errorf("no PTR record found for IP %s", ip.String())
		}

		ptr := strings.TrimSuffix(ptrs[0], ".")
		if strings.HasPrefix(ptr, "baby") {
			return ptr, nil
		}
		return "", fmt.Errorf("PTR record %q does not start with 'baby'", ptr)
	}

	return "", fmt.Errorf("no IPv4 address found for host %s", c.Host)
}

func (c *Client) CreateCertificate(hostname string) (*Certificate, error) {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return nil, fmt.Errorf("FQDN resolution failed: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/", fqdn, c.Port)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", fmt.Sprintf(`{"hostname": "%s"}`, hostname),
		url,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("curl failed: %v\nOutput: %s", err, out)
	}

	var cert Certificate
	if err := json.Unmarshal(out, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse certificate response: %w\nRaw: %s", err, out)
	}

	return &cert, nil
}

func (c *Client) GetCertificate(hostname string) (*Certificate, error) {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve FQDN: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", fqdn, c.Port, hostname)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "GET", url)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("curl failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute curl: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("empty response body from API")
	}

	type certificateListResponse struct {
		Objects []Certificate `json:"objects"`
	}

	var list certificateListResponse
	if err := json.Unmarshal(output, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON list: %w\nRaw: %s", err, output)
	}

	if len(list.Objects) == 0 {
		return nil, fmt.Errorf("no certificates found for hostname %s", hostname)
	}

	return &list.Objects[len(list.Objects)-1], nil
}


func (c *Client) DeleteCertificate(id int64)  error {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return  fmt.Errorf("failed to resolve FQDN: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%d", fqdn, c.Port, id)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "DELETE",
		"-w", "%{http_code}", "-o", "/dev/stdout", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return  fmt.Errorf("curl failed: %w; output: %s", err, output)
	}
	return  err
}

func (c *Client) UpdateCertificate(hostname string, cert Certificate) (*Certificate, error) {
	certPtr, err := c.GetCertificate(hostname)
	if err != nil {
		return nil, err
	}
	return certPtr, nil
}
