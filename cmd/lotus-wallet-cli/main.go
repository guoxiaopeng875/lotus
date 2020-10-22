package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/build"
	types "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	lotuslog.SetupLogLevels()

	local := []*cli.Command{
		walletNew,
		walletList,
		walletExport,
		walletImport,
		walletDelete,
		walletSign,
	}

	app := &cli.App{
		Name:     "lotus_wallet_cli",
		Usage:    "Basic external wallet",
		Version:  build.UserVersion(),
		Commands: local,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "api",
				Usage: "specify lotus-wallet api",
				Value: "localhost:1777",
			},
		},
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

func NewWalletAPI(cctx *cli.Context) (api.WalletAPI, jsonrpc.ClientCloser, error) {
	addr := cctx.String("api")
	var wltAPI apistruct.WalletStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), "http://"+addr+"/rpc/v0", "Filecoin", []interface{}{&wltAPI.Internal}, nil)
	return &wltAPI, closer, err
}

var walletNew = &cli.Command{
	Name:      "new",
	Usage:     "Generate a new key of the given type",
	ArgsUsage: "[bls|secp256k1 (default secp256k1)]",
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		api, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		t := cctx.Args().First()
		if t == "" {
			t = "secp256k1"
		}

		nk, err := api.WalletNew(ctx, types.KeyType(t))
		if err != nil {
			return err
		}

		fmt.Println(nk.String())

		return nil
	},
}

var walletList = &cli.Command{
	Name:  "list",
	Usage: "List wallet address",
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		api, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		addrs, err := api.WalletList(ctx)
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			fmt.Println(addr.String())
		}

		return nil
	},
}

var walletExport = &cli.Command{
	Name:      "export",
	Usage:     "export keys",
	ArgsUsage: "[address]",
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		api, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() {
			return fmt.Errorf("must specify key to export")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ki, err := api.WalletExport(ctx, addr)
		if err != nil {
			return err
		}

		b, err := json.Marshal(ki)
		if err != nil {
			return err
		}

		fmt.Println(hex.EncodeToString(b))
		return nil
	},
}

var walletImport = &cli.Command{
	Name:      "import",
	Usage:     "import keys",
	ArgsUsage: "[<path> (optional, will read from stdin if omitted)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "format",
			Usage: "specify input format for key",
			Value: "hex-lotus",
		},
		&cli.BoolFlag{
			Name:  "as-default",
			Usage: "import the given key as your new default key",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		api, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		var inpdata []byte
		if !cctx.Args().Present() || cctx.Args().First() == "-" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter private key: ")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return err
			}
			inpdata = indata

		} else {
			fdata, err := ioutil.ReadFile(cctx.Args().First())
			if err != nil {
				return err
			}
			inpdata = fdata
		}

		var ki types.KeyInfo
		switch cctx.String("format") {
		case "hex-lotus":
			data, err := hex.DecodeString(strings.TrimSpace(string(inpdata)))
			if err != nil {
				return err
			}

			if err := json.Unmarshal(data, &ki); err != nil {
				return err
			}
		case "json-lotus":
			if err := json.Unmarshal(inpdata, &ki); err != nil {
				return err
			}
		case "gfc-json":
			var f struct {
				KeyInfo []struct {
					PrivateKey []byte
					SigType    int
				}
			}
			if err := json.Unmarshal(inpdata, &f); err != nil {
				return xerrors.Errorf("failed to parse go-filecoin key: %s", err)
			}

			gk := f.KeyInfo[0]
			ki.PrivateKey = gk.PrivateKey
			switch gk.SigType {
			case 1:
				ki.Type = types.KTSecp256k1
			case 2:
				ki.Type = types.KTBLS
			default:
				return fmt.Errorf("unrecognized key type: %d", gk.SigType)
			}
		default:
			return fmt.Errorf("unrecognized format: %s", cctx.String("format"))
		}

		addr, err := api.WalletImport(ctx, &ki)
		if err != nil {
			return err
		}

		fmt.Printf("imported key %s successfully!\n", addr)
		return nil
	},
}

// ./lotus-wallet-cli sign --from f3sdvcdvikf5gqtc2w2ihxqdjvsls453teo7hrrelloivzzvgx7jns6dmupmkztimyrrx4cbl7ckgnfs4kg4xa --nonce 7 --gas-premium 150908 --gas-feecap 151962 --gas-limit 546585 f3s5bwq3cxbaapfpzskoujl7txmprxb3fr3xgrp7hg2ozvxroagk7ukw4oegrslovtbycu2u4hbuhj5kykbj5a 0.00000001
var walletSign = &cli.Command{
	Name:      "sign",
	Usage:     "sign a tx message",
	ArgsUsage: "[targetAddress] [amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "specify the account to send funds from",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "specify gas price to use in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "specify gas fee cap to use in AttoFIL",
			Value: "0",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "specify gas limit",
			Value: 0,
		},
		&cli.Int64Flag{
			Name:  "nonce",
			Usage: "specify the nonce to use",
			Value: -1,
		},
		&cli.Uint64Flag{
			Name:  "method",
			Usage: "specify method to invoke",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		wAPI, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() || cctx.NArg() != 2 {
			return fmt.Errorf("must specify signing address and message to sign")
		}
		msg, err := parseMessageFromContext(cctx)
		if err != nil {
			return err
		}
		msg.Nonce = uint64(cctx.Int64("nonce"))
		mb, err := msg.ToStorageBlock()
		if err != nil {
			return xerrors.Errorf("serializing message: %w", err)
		}

		sig, err := wAPI.WalletSign(ctx, msg.From, mb.Cid().Bytes(), api.MsgMeta{
			Type:  api.MTChainMsg,
			Extra: mb.RawData(),
		})
		if err != nil {
			return xerrors.Errorf("failed to sign message: %w", err)
		}
		sm := &types.SignedMessage{
			Message:   *msg,
			Signature: *sig,
		}
		smBytes, err := sm.Serialize()
		if err != nil {
			return err
		}
		fmt.Println(hex.EncodeToString(smBytes))
		return nil
	},
}

func parseMessageFromContext(cctx *cli.Context) (*types.Message, error) {
	toAddr, err := address.NewFromString(cctx.Args().Get(0))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target address: %w", err)
	}

	val, err := types.ParseFIL(cctx.Args().Get(1))
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	var fromAddr address.Address
	from := cctx.String("from")
	addr, err := address.NewFromString(from)
	if err != nil {
		return nil, fmt.Errorf("failed to parse from address: %w", err)
	}

	fromAddr = addr

	gp, err := types.BigFromString(cctx.String("gas-premium"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas-premium: %w", err)
	}
	gfc, err := types.BigFromString(cctx.String("gas-feecap"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas-feecap: %w", err)
	}

	method := abi.MethodNum(cctx.Uint64("method"))

	msg := &types.Message{
		From:       fromAddr,
		To:         toAddr,
		Value:      types.BigInt(val),
		GasPremium: gp,
		GasFeeCap:  gfc,
		GasLimit:   cctx.Int64("gas-limit"),
		Method:     method,
		Params:     nil,
	}
	return msg, nil
}

var walletDelete = &cli.Command{
	Name:      "delete",
	Usage:     "Delete an account from the wallet",
	ArgsUsage: "<address> ",
	Action: func(cctx *cli.Context) error {
		ctx := context.Background()
		api, closer, err := NewWalletAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		if !cctx.Args().Present() || cctx.NArg() != 1 {
			return fmt.Errorf("must specify address to delete")
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		return api.WalletDelete(ctx, addr)
	},
}
