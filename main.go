package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/iand/logfmtr"
	bserv "github.com/ipfs/go-blockservice"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	"github.com/iand/filecoin-kernel/chain"
	"github.com/iand/filecoin-kernel/networks/mainnet"
	filsync "github.com/iand/filecoin-kernel/sync"
)

func main() {
	app := &cli.App{
		Name: "filecoin-kernel-demo",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "log-level",
				Aliases: []string{"ll"},
				Usage:   "Set verbosity of logs to `LEVEL` (-1: off, 0: info, 1:debug)",
				Value:   0,
			},
			&cli.StringFlag{
				Name:    "peerfile",
				Aliases: []string{"pf"},
				Usage:   "Path to file containing list of peers to connect to.",
			},
		},
		Action:          run,
		HideHelpCommand: true,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(cc *cli.Context) error {
	loggerOpts := logfmtr.DefaultOptions()
	loggerOpts.Humanize = true
	loggerOpts.Colorize = true
	logfmtr.UseOptions(loggerOpts)

	logger := logr.Discard()
	ll := cc.Int("log-level")
	if ll >= 0 {
		logfmtr.SetVerbosity(ll)
		logger = logfmtr.New()
	}

	initialPeers, err := readInitialPeers(cc)
	if err != nil {
		return fmt.Errorf("read peer file: %w", err)
	}

	ctx := logr.NewContext(cc.Context, logger)

	ntwk := mainnet.Network

	ha, r, err := NewRoutedHost(ctx, fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 33110), 5, ntwk)
	if err != nil {
		return err
	}
	defer ha.Close()
	logger.Info("Connected to filecoin dht", "peer_id", ha.ID())

	bstore := NewMemBlockstore()
	mstore := NewMemDatastore()

	cs, err := chain.NewStore(bstore, mstore)
	if err != nil {
		return err
	}

	// Set up genesis
	genesisTs, err := cs.Import(ctx, ntwk.GenesisData())
	if err != nil {
		return err
	}

	logger.Info("loaded genesis", "cid", genesisTs.Cids[0])

	err = cs.SetGenesis(ctx, genesisTs.Cids[0])
	if err != nil {
		return err
	}

	err = cs.SetHeaviestTipSet(ctx, genesisTs)
	if err != nil {
		return err
	}

	// Create ipfs block service backed by bitswap
	exch := initBitswap(ctx, ha, r, bstore)
	bsrv := bserv.New(bstore, exch)

	coord := filsync.NewCoordinator(cs, ha, ntwk, bsrv)

	logger.Info("running chain sync coordinator")

	go func() {
		// Run blocks until an error or the context is cancelled
		if err := coord.Run(ctx); err != nil {
			logger.Error(err, "sync coordinator failed")
		}
	}()

	for _, addr := range initialPeers {
		logger.Info("connecting to peer", "addr", addr)
		if err := ha.Connect(ctx, addr); err != nil {
			logger.Error(err, "connect to peer", "addr", addr)
		}
	}

	select {
	case <-ctx.Done():
	}
	return nil
}

func readInitialPeers(cc *cli.Context) ([]peer.AddrInfo, error) {
	if cc.String("peerfile") == "" {
		return []peer.AddrInfo{}, nil
	}

	file, err := os.Open(cc.String("peerfile"))
	if err != nil {
		return nil, fmt.Errorf("open peerfile: %w", err)
	}
	defer file.Close()

	addrs := make([]peer.AddrInfo, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		maddr := ma.StringCast(scanner.Text())
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse peer address %q: %w", maddr, err)
		}

		addrs = append(addrs, *p)
	}

	return addrs, nil
}
