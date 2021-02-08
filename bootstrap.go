package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"sync"

	"github.com/go-logr/logr"
	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	"github.com/iand/filecoin-kernel/networks"
)

func NewRoutedHost(ctx context.Context, addr string, randseed int64, ntwk networks.Network) (host.Host, routing.Routing, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(addr),
		libp2p.Identity(priv),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
	}

	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.Datastore(dstore),
		// dht.Validator(validator),
		dht.ProtocolPrefix(protocol.ID("/fil/kad/" + ntwk.Name() + "/kad/1.0.0")),
		dht.QueryFilter(dht.PublicQueryFilter),
		dht.RoutingTableFilter(dht.PublicRoutingTableFilter),
		dht.DisableProviders(),
		dht.DisableValues(),
	}

	// Make the DHT
	dht, err := dht.New(ctx, basicHost, dhtOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	// connect to the chosen ipfs nodes
	err = bootstrapConnect(ctx, routedHost, ntwk.BootstrapPeers())
	if err != nil {
		return nil, nil, err
	}

	// Bootstrap the host
	err = dht.Bootstrap(ctx)
	if err != nil {
		return nil, nil, err
	}

	return routedHost, dht, nil
}

// This code is borrowed from the go-ipfs bootstrap process
// TODO: replace with https://github.com/ipfs/go-ipfs/blob/master/core/bootstrap/bootstrap.go
func bootstrapConnect(ctx context.Context, ph host.Host, peers []peer.AddrInfo) error {
	logger := logr.FromContextOrDiscard(ctx)

	if len(peers) < 1 {
		return errors.New("not enough bootstrap peers")
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

		// performed asynchronously because when performed synchronously, if
		// one `Connect` call hangs, subsequent calls are more likely to
		// fail/abort due to an expiring context.
		// Also, performed asynchronously for dial speed.

		wg.Add(1)
		go func(p peer.AddrInfo) {
			defer wg.Done()
			ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				logger.Error(err, "bootstrap failed", "id", p.ID)
				errs <- err
				return
			}
			logger.Info("bootstrapped to peer", "id", p.ID)
		}(p)
	}
	wg.Wait()

	// our failure condition is when no connection attempt succeeded.
	// So drain the errs channel, counting the results.
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

type PeerList struct {
	peers []peer.ID
}

func NewPeerList(peers []string) (*PeerList, error) {
	pl := &PeerList{}

	for _, pstr := range peers {
		id, err := peer.Decode(pstr)
		if err != nil {
			return nil, fmt.Errorf("invalid peer id (%q): %w", pstr, err)
		}
		pl.peers = append(pl.peers, id)
	}

	return pl, nil
}

func (p *PeerList) Peers() []peer.ID {
	return p.peers
}

func StringSliceContains(haystack []string, needle string) bool {
	for _, p := range haystack {
		if p == needle {
			return true
		}
	}
	return false
}

func initBitswap(ctx context.Context, host host.Host, rt routing.Routing, bs blockstore.Blockstore) exchange.Interface {
	// prefix protocol for chain bitswap
	// (so bitswap uses /chain/ipfs/bitswap/1.0.0 internally for chain sync stuff)
	bitswapNetwork := network.NewFromIpfsHost(host, rt, network.Prefix("/chain"))
	bitswapOptions := []bitswap.Option{bitswap.ProvideEnabled(false)}

	// Use just exch.Close(), closing the context is not needed
	exch := bitswap.New(ctx, bitswapNetwork, bs, bitswapOptions...)
	return exch
}
