package main

import (
	"context"
	"fmt"

	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

type MemBlockstore struct {
	data map[cid.Cid]block.Block
}

func NewMemBlockstore() *MemBlockstore {
	return &MemBlockstore{make(map[cid.Cid]block.Block)}
}

func (mb *MemBlockstore) Get(c cid.Cid) (block.Block, error) {
	d, ok := mb.data[c]
	if ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found")
}

func (mb *MemBlockstore) Put(b block.Block) error {
	mb.data[b.Cid()] = b
	return nil
}

func (mb *MemBlockstore) Close() error {
	return nil
}

func (mb *MemBlockstore) Has(c cid.Cid) (bool, error) {
	_, ok := mb.data[c]
	return ok, nil
}

func (mb *MemBlockstore) GetSize(c cid.Cid) (int, error) {
	d, ok := mb.data[c]
	if ok {
		return len(d.RawData()), nil
	}
	return 0, fmt.Errorf("not found")
}

func (mb *MemBlockstore) PutMany(blocks []block.Block) error {
	for _, b := range blocks {
		mb.data[b.Cid()] = b
	}
	return nil
}

func (mb *MemBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	ch := make(chan cid.Cid)
	go func() {
		defer close(ch)

		for c := range mb.data {
			ch <- c
		}
	}()

	return ch, nil
}

func (mb *MemBlockstore) DeleteBlock(c cid.Cid) error {
	delete(mb.data, c)
	return nil
}

func (mb *MemBlockstore) HashOnRead(_ bool) {
	// ignore
}

func (mb *MemBlockstore) View(c cid.Cid, callback func([]byte) error) error {
	d, ok := mb.data[c]
	if !ok {
		fmt.Errorf("not found")
	}
	return callback(d.RawData())
}

type MemDatastore struct {
	data map[datastore.Key][]byte
}

func NewMemDatastore() *MemDatastore {
	return &MemDatastore{make(map[datastore.Key][]byte)}
}

func (md *MemDatastore) Get(key datastore.Key) ([]byte, error) {
	d, ok := md.data[key]
	if ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found")
}

func (md *MemDatastore) Put(key datastore.Key, v []byte) error {
	md.data[key] = v
	return nil
}

func (md *MemDatastore) Delete(key datastore.Key) error {
	delete(md.data, key)
	return nil
}
