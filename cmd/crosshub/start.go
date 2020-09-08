package main

import (
	"fmt"
	"github.com/simplechain-org/crosshub/chainview"
	"github.com/simplechain-org/crosshub/fabric/courier"
	"github.com/simplechain-org/crosshub/fabric/courier/client"

	"github.com/simplechain-org/crosshub/core"
	"github.com/simplechain-org/crosshub/repo"
	"github.com/simplechain-org/crosshub/swarm"

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

	eventCh := make(chan *core.CrossTransaction, 4096)
	if s, err := swarm.New(repo, eventCh); err != nil {
		log.Error("swarm.New", "err", err)
		return err
	} else {
		if err := s.Start(); err != nil {
			log.Error("s.Start", "err", err)
			return err
		}
	}

	if repo.Config.Role == 1 {
		if v, err := chainview.New(repo, eventCh); err != nil {
			log.Error("chainview.New", "err", err)
			return err
		} else {
			if err := v.Start(); err != nil {
				log.Error("s.Start", "err", err)
				return err
			}
		}
	} else {
		courierHandler, err := courier.New(client.InitConfig(repo.Config.Fabric))
		if err != nil {
			log.Error("[courier.Handler] new handler", "err", err)
		}

		courierHandler.Start()
		defer courierHandler.Stop()
	}

	log.Info("new config", "store", repo.Config.Fabric)

	//fabricView.New(repo,eventCh)
	<-ch
	return nil
}
