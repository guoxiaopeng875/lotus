package main

import (
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	cli2 "github.com/filecoin-project/lotus/cli"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli/v2"
	"os"

	"github.com/filecoin-project/lotus/build"
)

func main() {

	local := []*cli.Command{
		pushCmd,
	}

	app := &cli.App{
		Name:     "mpool-push",
		Usage:    "mpool-push",
		Version:  build.UserVersion(),
		Commands: local,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:  "msg",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			api, closer, err := cli2.GetFullNodeAPI(c)
			defer closer()
			ctx := cli2.ReqContext(c)
			msgBytes, err := hex.DecodeString(c.String("msg"))
			if err != nil {
				panic(err)
			}
			sm, err := types.DecodeSignedMessage(msgBytes)
			if err != nil {
				panic(err)
			}
			cid, err := api.MpoolPush(ctx, sm)
			if err != nil {
				panic(err)
			}
			fmt.Println(cid.String())
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
		return
	}
}

var pushCmd = &cli.Command{
	Name: "watch-head",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "threshold",
			Value: 3,
			Usage: "number of times head remains unchanged before failing health check",
		},
		&cli.IntFlag{
			Name:  "interval",
			Value: int(build.BlockDelaySecs),
			Usage: "interval in seconds between chain head checks",
		},
		&cli.StringFlag{
			Name:  "systemd-unit",
			Value: "lotus-daemon.service",
			Usage: "systemd unit name to restart on health check failure",
		},
		&cli.IntFlag{
			Name: "api-timeout",
			// TODO: this default value seems spurious.
			Value: int(build.BlockDelaySecs),
			Usage: "timeout between API retries",
		},
		&cli.IntFlag{
			Name:  "api-retries",
			Value: 8,
			Usage: "number of API retry attempts",
		},
	},
	Action: func(c *cli.Context) error {
		return nil
	},
}
