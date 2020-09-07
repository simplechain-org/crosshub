// Copyright 2016 The go-simplechain Authors
// This file is part of the go-simplechain library.
//
// The go-simplechain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-simplechain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-simplechain library. If not, see <http://www.gnu.org/licenses/>.

package repo

import (
	"encoding/json"
	"github.com/simplechain-org/go-simplechain/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

//import (
//	"github.com/simplechain-org/go-simplechain/common"
//	"github.com/simplechain-org/go-simplechain/cross/backend/synchronise"
//)
//
//const (
//	LogDir   = "crosslog"
//	TxLogDir = "crosstxlog"
//	DataDir  = "crossdata"
//)
//
//type Config struct {
//	MainContract common.Address       `json:"mainContract"`
//	SubContract  common.Address       `json:"subContract"`
//	Signer       common.Address       `json:"signer"`
//	Anchors      []common.Address     `json:"anchors"`
//	SyncMode     synchronise.SyncMode `json:"syncMode"`
//}
//
//var DefaultConfig = Config{
//	SyncMode: synchronise.ALL,
//}
//
//func (config *Config) Sanitize() Config {
//	cfg := Config{
//		MainContract: config.MainContract,
//		SubContract:  config.SubContract,
//		Signer:       config.Signer,
//	}
//	set := make(map[common.Address]struct{})
//	for _, anchor := range config.Anchors {
//		if _, ok := set[anchor]; !ok {
//			cfg.Anchors = append(cfg.Anchors, anchor)
//			set[anchor] = struct{}{}
//		}
//	}
//	return cfg
//}

const (
	// defaultPathName is the default config dir name
	defaultPathName = ".crosshub"
	// defaultPathRoot is the path to the default config dir location.
	defaultPathRoot = "~/" + defaultPathName
	// envDir is the environment variable used to change the path root.
	envDir = "CROSSHUB_PATH"
	// Config name
	configName = "crosshub.toml"
	// key name
	KeyName = "key.json"
	// API name
	APIName = "api"
)

type Config struct {
	Title    string `toml:"title" json:"title"`
	RepoRoot string `toml:"repo_root" json:"repo_root"`
	Contract string `toml:"contract" json:"contract"` //跨链合约地址
	RpcIp   string `toml:"rpcip" json:"rpc_ip"`
	RpcPort string `toml:"rpcport" json:"rpc_port"`
	Port     `toml:"port" json:"port"`
	Gateway  `toml:"gateway" json:"gateway"`
	Cert     `toml:"cert" json:"cert"`
}

type Port struct {
	Grpc    int64 `toml:"grpc" json:"grpc"`
	Gateway int64 `toml:"gateway" json:"gateway"`
}

type Gateway struct {
	AllowedOrigins []string `toml:"allowed_origins" mapstructure:"allowed_origins"`
}

type Cert struct {
	Verify bool `toml:"verify" json:"verify"`
}

func (c *Config) Bytes() ([]byte, error) {
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func DefaultConfig() (*Config, error) {
	return &Config{
		Title: "CrossHub configuration file",
		//Contract: "0xf7bea9e8a0c8e99af6e52ff5e41ec9cac6e6c314",
		//RpcUrl: "http://112.124.0.14:58545",
		Port: Port{
			Grpc:    60011,
			Gateway: 9091,
		},
		Gateway: Gateway{AllowedOrigins: []string{"*"}},
		Cert: Cert{Verify: true},
	}, nil
}

func UnmarshalConfig(repoRoot string) (*Config, error) {
	viper.SetConfigFile(filepath.Join(repoRoot, configName))
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CROSSHUB")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	//config, err := DefaultConfig()
	//if err != nil {
	//	return nil, err
	//}
	var config Config

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	log.Info("UnmarshalConfig","config",config)
	config.RepoRoot = repoRoot

	return &config, nil
}

func ReadConfig(path, configType string, config interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(configType)
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	if err := v.Unmarshal(config); err != nil {
		return err
	}

	return nil
}

func PathRoot() (string, error) {
	dir := os.Getenv(envDir)
	var err error
	if len(dir) == 0 {
		dir, err = homedir.Expand(defaultPathRoot)
	}
	return dir, err
}

func PathRootWithDefault(path string) (string, error) {
	if len(path) == 0 {
		return PathRoot()
	}

	return path, nil
}