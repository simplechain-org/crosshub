package hubnet

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type Config struct {
	localAddr  string
	privKey    crypto.PrivKey
	protocolID protocol.ID
}

type Option func(*Config)

func WithPrivateKey(privKey crypto.PrivKey) Option {
	return func(config *Config) {
		config.privKey = privKey
	}
}

func WithLocalAddr(addr string) Option {
	return func(config *Config) {
		config.localAddr = addr
	}
}

func WithProtocolID(id protocol.ID) Option {
	return func(config *Config) {
		config.protocolID = id
	}
}

func checkConfig(config *Config) error {
	if config.localAddr == "" {
		return fmt.Errorf("empty local address")
	}

	return nil
}

func generateConfig(opts ...Option) (*Config, error) {
	conf := &Config{}
	for _, opt := range opts {
		opt(conf)
	}

	if err := checkConfig(conf); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	return conf, nil
}
