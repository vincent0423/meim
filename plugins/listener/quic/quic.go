package quic

import (
	"errors"
	"net"

	"github.com/ipiao/meim/comect"
	quicconn "github.com/marten-seemann/quic-conn"
)

func init() {
	comect.RegisterMakeListener("quic", quicMakeListener)
}

func quicMakeListener(cfg *comect.ListenerConfig) (ln net.Listener, err error) {
	if cfg.TLSConfig == nil {
		return nil, errors.New("TLSConfig must be configured in cfg.Options")
	}
	return quicconn.Listen("udp", cfg.Address, cfg.TLSConfig)
}
