package imap

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

// Client é um wrapper sobre imapclient.Client.
type Client struct {
	*imapclient.Client
}

// Connect abre uma conexão IMAP e autentica com LOGIN.
func Connect(host string, port int, useTLS bool, timeout time.Duration, user, pass string, debug bool) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	dialer := &net.Dialer{Timeout: timeout}

	var inner *imapclient.Client
	
	opts := &imapclient.Options{}
	if debug {
		opts.DebugWriter = os.Stdout
	}

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
