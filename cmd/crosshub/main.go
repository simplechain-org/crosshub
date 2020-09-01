package main

import (
	"os"
	"time"

	"github.com/urfave/cli"
	"github.com/simplechain-org/go-simplechain/log"
)

func main() {
	app := cli.NewApp()
	app.Name = "CrossHub"
	app.Usage = "A leading inter-blockchain platform"
	app.Compiled = time.Now()

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	// global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "repo",
			Usage: "CrossHub storage repo path",
		},
	}

	app.Commands = []cli.Command{
		configCMD(),
		initCMD(),
		startCMD(),
		keyCMD(),
		//versionCMD(),
		certCMD,
		//client.LoadClientCMD(),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error("Run","err",err)
	}
}
