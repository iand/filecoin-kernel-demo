module github.com/iand/filecoin-kernel-demo

go 1.16

require (
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/go-logr/logr v0.3.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/iand/filecoin-kernel v0.0.0-00010101000000-000000000000
	github.com/iand/logfmtr v0.1.5
	github.com/ipfs/go-bitswap v0.3.3
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-blockservice v0.1.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ipfs-blockstore v1.0.3
	github.com/ipfs/go-ipfs-exchange-interface v0.0.1
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-core v0.8.0
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
)

replace github.com/iand/filecoin-kernel => ../filecoin-kernel
