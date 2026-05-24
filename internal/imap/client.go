// Package imap provides a thin wrapper around the go-imap/v2 client library,
// exposing higher-level methods for mailbox management, message operations, and quota queries.
package imap

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

// Client wraps imapclient.Client with higher-level convenience methods.
type Client struct {
	*imapclient.Client
}

// Connect dials an IMAP server (TLS or plain), authenticates with LOGIN, and returns a Client.
// When debug is true, raw IMAP commands and responses are written to stdout.
func Connect(host string, port int, useTLS bool, timeout time.Duration, user, pass string, debug bool) (*Client, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: timeout}

	opts := &imapclient.Options{}
	if debug {
		opts.DebugWriter = os.Stdout
	}

	var inner *imapclient.Client
	if useTLS {
		tlsCfg := &tls.Config{ServerName: host}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("tls dial: %w", err)
		}
		inner = imapclient.New(conn, opts)
	} else {
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("dial: %w", err)
		}
		inner = imapclient.New(conn, opts)
	}

	if err := inner.Login(user, pass).Wait(); err != nil {
		inner.Close()
		return nil, fmt.Errorf("login: %w", err)
	}

	return &Client{Client: inner}, nil
}
