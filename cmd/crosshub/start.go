package main

import (
	"fmt"
	"github.com/simplechain-org/crosshub/chainview"
	"github.com/simplechain-org/crosshub/fabric/courier"
	"github.com/simplechain-org/crosshub/fabric/courier/client"
	"github.com/simplechain-org/crosshub/repo"
	"github.com/simplechain-org/crosshub/swarm"

	"github.com/simplechain-org/go-simplechain/crypto/ecdsa"
	"github.com/simplechain-org/go-simplechain/log"
	"github.com/urfave/cli"
)

func startCMD() cli.Command {
	return cli.Command{
		Name:   "start",
		Usage:  "Start a long-running start process",
		Action: start,
	}
}

func start(ctx *cli.Context) error {
	ch := make(chan bool)
	log.Info("start")
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		log.Error("PathRootWithDefault", "err", err)
		return fmt.Errorf("get repo path: %w", err)
	}

	repo, err := repo.Load(repoRoot)
	if err != nil {
		log.Error("repo.Load", "err", err)
		return fmt.Errorf("repo load: %w", err)
	}

	//eventCh := make(chan *core.CrossTransaction, 4096)
	//rtxCh  := make(chan *core.ReceptTransaction,4096)
	eventCh := make(chan interface{}, 4096)
	messageCh := make(chan interface{}, 4096)
	s, err := swarm.New(repo, messageCh, eventCh)
	if err != nil {
		log.Error("swarm.New", "err", err)
		return err
	}

	if err := s.Start(); err != nil {
		log.Error("s.Start", "err", err)
		return err
	}

	switch repo.Config.Role {
	case 1:
		v, err := chainview.New(repo, eventCh, messageCh)
		if err != nil {
			log.Error("chainview.New", "err", err)
			return err
		}

		if err := v.Start(); err != nil {
			log.Error("s.Start", "err", err)
			return err
		}
	default:
		courierHandler, err := courier.New(client.InitConfig(repo.Config.Fabric), &courier.CrossChannel{
			eventCh,
			messageCh,
		})

		if err != nil {
			log.Error("[courier.Handler] new handler", "err", err)
			return err
		}

		// set private key
		courierHandler.SetPrivateKey(repo.Key.PrivKey.(*ecdsa.PrivateKey))
		// accept cross request from simplechain
		courierHandler.SetOutChainFlag(true)

		courierHandler.Start()
		defer courierHandler.Stop()
	}

	log.Info("new config", "store", repo.Config.Fabric)

	//fabricView.New(repo,eventCh)
	<-ch
	return nil
}
