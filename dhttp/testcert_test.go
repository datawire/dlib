package dhttp_test

import (
	"os"
	"path/filepath"
)

func testCertFiles() (certFile string, keyFile string, cleanup func(), err error) {
	tmpdir, err := os.MkdirTemp("", "dhttp-test.")
	if err != nil {
		return "", "", nil, err
	}

	certFile = filepath.Join(tmpdir, "cert.pem")
	keyFile = filepath.Join(tmpdir, "key.pem")
	cleanup = func() {
		_ = os.RemoveAll(tmpdir)
	}

	if err := os.WriteFile(certFile, LocalhostCert, 0600); err != nil {
		cleanup()
		return "", "", nil, err
	}
	if err := os.WriteFile(keyFile, LocalhostKey, 0600); err != nil {
		cleanup()
		return "", "", nil, err
	}

	return certFile, keyFile, cleanup, nil
}
