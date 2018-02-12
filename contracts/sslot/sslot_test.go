package sslot_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vechain/thor/lvldb"
	"github.com/vechain/thor/state"

	"github.com/vechain/thor/thor"

	"github.com/vechain/thor/contracts/sslot"
)

func TestSSlot(t *testing.T) {
	kv, _ := lvldb.NewMem()
	st, _ := state.New(thor.Hash{}, kv)

	addr := thor.BytesToAddress([]byte("acc"))
	ss := sslot.New(addr, 0)

	ss.SaveStructed(st, thor.Hash{1})
	var v thor.Hash
	ss.LoadStructed(st, &v)
	assert.Equal(t, thor.Hash{1}, v)
}
