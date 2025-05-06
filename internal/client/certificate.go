// Copyright (c) Christopher Barnes <christopher.barnes@cern.ch>
// SPDX-License-Identifier: GPL-3.0-or-later

package certMgr

import (
	"encoding/json"
	"errors"
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

var ErrNoCertificates = errors.New("no certificates found")

func (c *Client) CreateCertificate(hostname string) (*Certificate, error) {
	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/", c.Host, c.Port)
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
	url := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", c.Host, c.Port, hostname)
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
		return nil, ErrNoCertificates
	}

	latestCert := staged.Objects[len(staged.Objects)-1]

	return &latestCert, nil
}

func (c *Client) UpdateCertificate(cert Certificate) error {
	data, err := json.Marshal(cert)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	url := fmt.Sprintf("https://%s:%d/krb/certmgr/certificate/", c.Host, c.Port)
	if _, _, err := c.doRequest(http.MethodPost, url, data); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteCertificate(hostname string) error {
	urlList := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/?hostname=%s", c.Host, c.Port, hostname)
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
		urlDel := fmt.Sprintf("https://%s:%d/krb/certmgr/staged/%d/", c.Host, c.Port, event.ID)
		if _, _, err := c.doRequest(http.MethodDelete, urlDel, nil); err != nil {
			return fmt.Errorf("delete failed for event %d: %w", event.ID, err)
		}
	}
	return nil
}
