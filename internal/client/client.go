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
		Host:       "hector.cern.ch", // default
		Port:       8008,             // default
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
	fqdn, err := net.LookupCNAME(c.Host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve FQDN for host %s: %w", c.Host, err)
	}
	return fqdn, nil
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

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%s", fqdn, c.Port, hostname)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "GET", url)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("curl failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute curl: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(output, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &cert, nil
}

func (c *Client) DeleteCertificate(id int) error {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return fmt.Errorf("failed to resolve FQDN: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%d", fqdn, c.Port, id)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "DELETE",
		"-w", "%{http_code}", "-o", "/dev/stdout", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("curl failed: %w; output: %s", err, output)
	}
	return err
}

func (c *Client) UpdateCertificate(hostname string, cert Certificate) (*Certificate, error) {
	certPtr, err := c.GetCertificate(hostname)
	if err != nil {
		return nil, err
	}
	return certPtr, nil
}
