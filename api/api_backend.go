package api

import (
	"github.com/simplechain-org/crosshub/core"
	db "github.com/simplechain-org/crosshub/database"
)

func (s *CrossQueryApi)QueryByPage(localSize, localPage, remoteSize, remotePage int) (
	locals map[uint8][]*core.CrossTransaction, remotes map[uint8][]*core.CrossTransaction) {
	orderBy := []db.FieldName{db.PriceIndex}
	locals = map[uint8][]*core.CrossTransaction{2:s.localDb.Query(localSize,localPage,orderBy,false)}
	remotes = map[uint8][]*core.CrossTransaction{5:s.remoteDb.Query(localSize,localPage,orderBy,false)}
	return
}