package utils

import (
	"os"

	"github.com/simplechain-org/go-simplechain/log"
)

var Logger = log.New()

func init() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(log.LvlDebug)
	Logger.SetHandler(glogger)
}
