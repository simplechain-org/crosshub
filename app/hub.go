package app

import (
	"context"
	"github.com/simplechain-org/crosshub/repo"
	"github.com/simplechain-org/crosshub/swarm"
)

type Hub struct {


	PeerManger swarm.Swarm

	repo   *repo.Repo
	ctx    context.Context
	cancel context.CancelFunc
}
