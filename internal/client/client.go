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

func NewClient(host, port string) (*Client, error) {
	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 || p > 65535 {
		return nil, fmt.Errorf("invalid port: %q", port)
	}

	return &Client{Host: host, Port: p}, nil
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
			return "", fmt.Errorf("reverse lookup failed for IP %s: %w", ip, err)
		}
		if len(ptrs) > 0 {
			return strings.TrimSuffix(ptrs[0], "."), nil
		}
	}
	return "", fmt.Errorf("no valid IPv4 PTR record found for host %s", c.Host)
}

func (c *Client) CreateCertificate(hostname string) (*Certificate, error) {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/", fqdn, c.Port)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", fmt.Sprintf(`{"hostname":"%s"}`, hostname), url)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("curl failed: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(out, &cert); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}
	return &cert, nil
}

func (c *Client) GetCertificate(hostname string) (*Certificate, error) {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", fqdn, c.Port, hostname)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "GET", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("curl failed: %w", err)
	}

	type stagedResponse struct {
		Meta    map[string]interface{} `json:"meta"`
		Objects []Certificate          `json:"objects"`
	}

	var staged stagedResponse
	if err := json.Unmarshal(output, &staged); err != nil {
		return nil, fmt.Errorf("failed unmarshaling staged certs: %w", err)
	}

	if len(staged.Objects) == 0 {
		return nil, fmt.Errorf("no certificates found for hostname %s", hostname)
	}

	latestCert := staged.Objects[len(staged.Objects)-1]

	return &latestCert, nil
}

func (c *Client) DeleteCertificate(hostname string) error {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return fmt.Errorf("FQDN resolution failed: %w", err)
	}

	urlList := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", fqdn, c.Port, hostname)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "GET", urlList)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed listing staged events: %w", err)
	}

	var events struct {
		Objects []struct {
			ID int `json:"id"`
		} `json:"objects"`
	}

	if err := json.Unmarshal(output, &events); err != nil {
		return fmt.Errorf("json parse error: %w", err)
	}

	for _, event := range events.Objects {
		urlDel := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%d/", fqdn, c.Port, event.ID)
		delCmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "DELETE", urlDel)
		if out, err := delCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("delete failed for event %d: %w, output: %s", event.ID, err, out)
		}
	}
	return nil
}

func (c *Client) UpdateCertificate(cert Certificate) error {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cert)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/certificate/", fqdn, c.Port)
	cmd := exec.Command("curl", "-s", "--negotiate", "-u", ":", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", string(data), url)

	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("curl failed: %w", err)
	}

	return nil
}
