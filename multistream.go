// package multistream implements a peerstream transport using
// go-multistream to select the underlying stream muxer
package multistream

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/compression"

	mss "github.com/multiformats/go-multistream"
)

// DefaultNegotiateTimeout time spent to negotiate transport.
var DefaultNegotiateTimeout = time.Second * 60

// Transport implements the compressedTransport intercace
type Transport struct {
	mux mss.MultistreamMuxer

	tpts map[string]compression.CompressedTransport

	NegotiateTimeout time.Duration

	OrderPreference []string
}

// AddTransport adds a new transport.
func (t *Transport) AddTransport(path string, tpt compression.CompressedTransport) {
	if t.tpts == nil {
		t.tpts = make(map[string]compression.CompressedTransport, 1)
	}
	t.mux.AddHandler(path, nil)
	t.tpts[path] = tpt
	t.OrderPreference = append(t.OrderPreference, path)
}

// NewConn upgrades to a compressed connection the raw connection.
func (t *Transport) NewConn(nc net.Conn, isServer bool) (compression.CompressedConn, error) {
	if t.NegotiateTimeout != 0 {
		if err := nc.SetDeadline(time.Now().Add(t.NegotiateTimeout)); err != nil {
			return nil, err
		}
	}

	var proto string
	if isServer {
		selected, _, err := t.mux.Negotiate(nc)
		if err != nil {
			return nil, err
		}
		proto = selected
	} else {
		selected, err := mss.SelectOneOf(t.OrderPreference, nc)
		if err != nil {
			return nil, err
		}
		proto = selected
	}

	if t.NegotiateTimeout != 0 {
		if err := nc.SetDeadline(time.Time{}); err != nil {
			return nil, err
		}
	}

	tpt, ok := t.tpts[proto]
	if !ok {
		return nil, fmt.Errorf("selected protocol we don't have a transport for")
	}

	return tpt.NewConn(nc, isServer)
}
