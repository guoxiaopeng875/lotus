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

	app := &cli.App{
		Name:    "mpool-push",
		Usage:   "mpool-push",
		Version: build.UserVersion(),
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
			if err != nil {
				return err
			}
			defer closer()
			ctx := cli2.ReqContext(c)
			msgBytes, err := hex.DecodeString(c.String("msg"))
			if err != nil {
				return err
			}
			sm, err := types.DecodeSignedMessage(msgBytes)
			if err != nil {
				return err
			}
			cid, err := api.MpoolPush(ctx, sm)
			if err != nil {
				return err
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
