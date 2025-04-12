// Copyright (c) Christopher Barnes <christopher@barnes.biz>
// SPDX-License-Identifier: MPL-2.0

package certMgr

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Certificate struct {
	ID        int    `json:"id"`
	Hostname  string `json:"hostname"`
	Requestor string `json:"requestor"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

func (c *Client) CreateCertificate(hostname string) (*Certificate, error) {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/", fqdn, c.Port)
	payload, _ := json.Marshal(map[string]string{"hostname": hostname})

	body, _, err := c.doRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}

	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
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
	body, _, err := c.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	type stagedResponse struct {
		Meta    map[string]interface{} `json:"meta"`
		Objects []Certificate          `json:"objects"`
	}

	var staged stagedResponse
	if err := json.Unmarshal(body, &staged); err != nil {
		return nil, fmt.Errorf("failed unmarshaling staged certs: %w", err)
	}

	if len(staged.Objects) == 0 {
		return nil, fmt.Errorf("no certificates found for hostname %s", hostname)
	}

	latestCert := staged.Objects[len(staged.Objects)-1]

	return &latestCert, nil
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
	if _, _, err := c.doRequest(http.MethodPost, url, data); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteCertificate(hostname string) error {
	fqdn, err := c.resolveFQDN()
	if err != nil {
		return fmt.Errorf("FQDN resolution failed: %w", err)
	}

	urlList := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", fqdn, c.Port, hostname)
	body, _, err := c.doRequest(http.MethodGet, urlList, nil)
	if err != nil {
		return fmt.Errorf("failed listing staged events: %w", err)
	}

	var events struct {
		Objects []struct {
			ID int `json:"id"`
		} `json:"objects"`
	}

	if err := json.Unmarshal(body, &events); err != nil {
		return fmt.Errorf("json parse error: %w", err)
	}

	for _, event := range events.Objects {
		urlDel := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%d/", fqdn, c.Port, event.ID)
		if _, _, err := c.doRequest(http.MethodDelete, urlDel, nil); err != nil {
			return fmt.Errorf("delete failed for event %d: %w", event.ID, err)
		}
	}
	return nil
}
