package utils

import (
	"os"

	"github.com/simplechain-org/go-simplechain/log"
)

var Logger = log.New()

func Verbosity(lvl string) {
	var l = log.LvlDebug
	switch lvl {
	case "info":
		l = log.LvlInfo
	case "warn":
		l = log.LvlWarn
	case "error":
		l = log.LvlError
	default:
		l = log.LvlDebug
	}

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(l)
	Logger.SetHandler(glogger)
}
