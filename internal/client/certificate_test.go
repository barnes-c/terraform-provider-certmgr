// Copyright (c) Christopher Barnes <christopher@barnes.biz>
// SPDX-License-Identifier: MPL-2.0

package certMgr_test

import (
	"fmt"
	"testing"
	"time"

	certMgr "certMgr/internal/client"

	"github.com/stretchr/testify/require"
)

func TestCertificateCRUD(t *testing.T) {
	host := "hector.cern.ch"
	port := 8008

	cli, err := certMgr.NewClient(host, port)
	require.NoError(t, err)

	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	last5 := timestamp[len(timestamp)-5:]

	hostname := fmt.Sprintf("tf-test-cert-%s.cern.ch", last5)

	t.Logf("Creating certificate for hostname: %s", hostname)
	createdCert, err := cli.CreateCertificate(hostname)
	require.NoError(t, err)
	require.Equal(t, hostname, createdCert.Hostname)

	t.Log("Reading certificate...")
	readCert, err := cli.GetCertificate(hostname)
	require.NoError(t, err)
	require.Equal(t, createdCert.Hostname, readCert.Hostname)

	defer func() {
		t.Logf("Deleting certificate for hostname: %s", hostname)
		err := cli.DeleteCertificate(hostname)
		require.NoError(t, err)
	}()

	t.Log("Updating certificate...")
	readCert.Requestor = "terraform-test"
	err = cli.UpdateCertificate(*readCert)
	require.NoError(t, err)

	t.Log("Final read to confirm update...")
	finalCert, err := cli.GetCertificate(hostname)
	require.NoError(t, err)
	require.Equal(t, "terraform-test", finalCert.Requestor)
}
