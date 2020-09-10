package api

import (
	"github.com/simplechain-org/crosshub/core"
	db "github.com/simplechain-org/crosshub/database"
)

func QueryByPage(idb *db.IndexDB,localSize, localPage, remoteSize, remotePage int) (
	locals map[uint8][]*core.CrossTransaction, remotes map[uint8][]*core.CrossTransaction) {
	orderBy := []db.FieldName{db.PriceIndex}
	locals = map[uint8][]*core.CrossTransaction{2:nil}
	remotes = map[uint8][]*core.CrossTransaction{5:idb.Query(localSize,localPage,orderBy,false)}
	return
}