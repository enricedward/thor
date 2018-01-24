package statedb

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/vechain/thor/stackedmap"
	"github.com/vechain/thor/thor"
)

// StateDB is facade for account.Manager, snapshot.Snapshot and Log.
// It implements evm.StateDB, only adapt to evm.
type StateDB struct {
	state State
	repo  *stackedmap.StackedMap
}

type (
	suicideFlagKey     common.Address
	refundKey          struct{}
	preimageKey        common.Hash
	logKey             struct{}
	affectedAddressKey common.Address
	createdContractKey common.Address
)

// New create a statedb object.
func New(state State) *StateDB {
	getter := func(k interface{}) (interface{}, bool) {
		switch k.(type) {
		case suicideFlagKey:
			return false, true
		case refundKey:
			return &big.Int{}, true
		}
		panic(fmt.Sprintf("unknown type of key %+v", k))
	}

	repo := stackedmap.New(getter)
	return &StateDB{
		state,
		repo,
	}
}

// GetRefund returns total refund during VM life-cycle.
func (s *StateDB) GetRefund() *big.Int {
	v, _ := s.repo.Get(refundKey{})
	return v.(*big.Int)
}

// GetOutputs callback ouputs include logs, new addresses and preimages.
// Merge callbacks for performance reasons.
func (s *StateDB) GetOutputs(
	logCB func(*types.Log) bool,
	affectedAddressCB func(common.Address) bool,
	createdContractCB func(common.Address) bool,
	preimagesCB func(common.Hash, []byte) bool,
) {
	s.repo.Journal(func(k, v interface{}) bool {
		switch key := k.(type) {
		case logKey:
			return logCB(v.(*types.Log))
		case affectedAddressKey:
			return affectedAddressCB(common.Address(key))
		case createdContractKey:
			return createdContractCB(common.Address(key))
		case preimageKey:
			return preimagesCB(common.Hash(key), v.([]byte))
		}
		return true
	})
}

// ForEachStorage see state.State.ForEachStorage.
func (s *StateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) {
	s.state.ForEachStorage(thor.Address(addr), func(k thor.Hash, v thor.Hash) bool {
		return cb(common.Hash(k), common.Hash(v))
	})
}

// CreateAccount stub.
func (s *StateDB) CreateAccount(addr common.Address) {}

// GetBalance stub.
func (s *StateDB) GetBalance(addr common.Address) *big.Int {
	return s.state.GetBalance(thor.Address(addr))
}

// SubBalance stub.
func (s *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.repo.Put(affectedAddressKey(addr), nil)
	balance := s.state.GetBalance(thor.Address(addr))
	s.state.SetBalance(thor.Address(addr), new(big.Int).Sub(balance, amount))
}

// AddBalance stub.
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.repo.Put(affectedAddressKey(addr), nil)
	balance := s.state.GetBalance(thor.Address(addr))
	s.state.SetBalance(thor.Address(addr), new(big.Int).Add(balance, amount))
}

// GetNonce stub.
func (s *StateDB) GetNonce(addr common.Address) uint64 { return 0 }

// SetNonce stub.
func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {}

// GetCodeHash stub.
func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(s.state.GetCodeHash(thor.Address(addr)))
}

// GetCode stub.
func (s *StateDB) GetCode(addr common.Address) []byte {
	return s.state.GetCode(thor.Address(addr))
}

// GetCodeSize stub.
func (s *StateDB) GetCodeSize(addr common.Address) int {
	return len(s.state.GetCode(thor.Address(addr)))
}

// SetCode stub.
func (s *StateDB) SetCode(addr common.Address, code []byte) {
	if len(code) > 0 {
		s.repo.Put(createdContractKey(addr), nil)
	}
	s.state.SetCode(thor.Address(addr), code)
}

// HasSuicided stub.
func (s *StateDB) HasSuicided(addr common.Address) bool {
	// only check suicide flag here
	v, _ := s.repo.Get(suicideFlagKey(addr))
	return v.(bool)
}

// Suicide stub.
// We do two things:
// 1, delete account
// 2, set suicide flag
func (s *StateDB) Suicide(addr common.Address) bool {
	if !s.state.Exists(thor.Address(addr)) {
		return false
	}
	s.state.Delete(thor.Address(addr))
	s.repo.Put(suicideFlagKey(addr), true)
	return true
}

// GetState stub.
func (s *StateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(s.state.GetStorage(thor.Address(addr), thor.Hash(key)))
}

// SetState stub.
func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	s.state.SetStorage(thor.Address(addr), thor.Hash(key), thor.Hash(value))
}

// Exist stub.
func (s *StateDB) Exist(addr common.Address) bool {
	return s.state.Exists(thor.Address(addr))
}

// Empty stub.
func (s *StateDB) Empty(addr common.Address) bool {
	return !s.state.Exists(thor.Address(addr))
}

// AddRefund stub.
func (s *StateDB) AddRefund(gas *big.Int) {
	v, _ := s.repo.Get(refundKey{})
	total := new(big.Int).Add(v.(*big.Int), gas)
	s.repo.Put(refundKey{}, total)
}

// AddPreimage stub.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	s.repo.Put(preimageKey(hash), preimage)
}

// AddLog stub.
func (s *StateDB) AddLog(vmlog *types.Log) {
	s.repo.Put(logKey{}, vmlog)
}

// Snapshot stub.
func (s *StateDB) Snapshot() int {
	s.state.NewCheckpoint()
	rev := s.repo.Push()
	return rev
}

// RevertToSnapshot stub.
func (s *StateDB) RevertToSnapshot(rev int) {
	if rev < 0 || rev > s.repo.Depth() {
		panic(fmt.Sprintf("invalid snapshot revision %d (depth:%d)", rev, s.repo.Depth()))
	}
	revertCount := s.repo.Depth() - rev
	for i := 0; i < revertCount; i++ {
		s.state.Revert()
	}
	s.repo.PopTo(rev)
}
