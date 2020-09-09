package client

import (
	"fmt"

	"github.com/simplechain-org/crosshub/fabric/courier/utils"
	"github.com/simplechain-org/crosshub/repo"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
)

type Config struct {
	// fabric client config
	core.ConfigProvider
	channel.RequestOption
	FilterEvents []string

	user        string
	channelID   string
	chaincodeID string
	dataDir     string
}

func checkConfig(cfg repo.Fabric) error {
	var elem string
	switch {
	case cfg.DataDir == "":
		elem = "datadir"
	case cfg.ChannelId == "":
		elem = "channelid"
	case cfg.ChaincodeId == "":
		elem = "chaincodeid"
	case cfg.User == "":
		elem = "user"
	case cfg.PeerUrl == nil:
		elem = "peerurl"
	case cfg.Events == nil:
		elem = "events"
	case cfg.ConfigPath == "":
		elem = "configpath"
	}

	if elem != "" {
		return fmt.Errorf("%s not set", elem)
	}
	return nil
}

// InitConfig initializes the configuration
func InitConfig(fabric repo.Fabric) *Config {

	if err := checkConfig(fabric); err != nil {
		utils.Fatalf("[courier.Config] err: %v", err)
	}

	cnfg := config.FromRaw(utils.ReplacePathInFile(fabric.ConfigPath), "yaml")

	cfg := &Config{
		ConfigProvider: cnfg,
		RequestOption:  channel.WithTargetEndpoints(fabric.PeerUrl...),
		FilterEvents:   fabric.Events,
		user:           fabric.User,
		chaincodeID:    fabric.ChaincodeId,
		channelID:      fabric.ChannelId,
		dataDir:        fabric.DataDir,
	}

	return cfg
}

// InitUserName initializes the user name from the provided arguments
func (c *Config) UserName() string {
	return c.user
}

// ChannelID returns the channel ID
func (c *Config) ChannelID() string {
	return c.channelID
}

// ChainCodeID returns the chaicode ID
func (c *Config) ChainCodeID() string {
	return c.chaincodeID
}

// DataDir returns the courier data directory
func (c *Config) DataDir() string {
	return c.dataDir
}
