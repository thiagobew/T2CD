package tuplespace

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/btree"

	opt "github.com/micutio/goptional"
)

// The Store defines an interface that any concrete implementation of a tuple space has to follow.
// The tuplespace assumes the store implementation to be thread-safe in order to allow concurrent
// access.
type Store interface {

	// Read a tuple that matches the argument and remove it from the space.
	Get(query Tuple) opt.Maybe[Tuple]

	// Read a tuple that matches the argument.
	Read(query Tuple) opt.Maybe[Tuple]

	// Write a tuple into the tuple space.
	Write(tuple Tuple) bool
}

// The BTreeStore is a simple in-memory implementation of a store.
type BTreeStore struct {
	tree *btree.BTreeG[Tuple]
}

// NewSimpleStore creates an empty store instance which is ready for use.
func NewSimpleStore() *BTreeStore {
	return &BTreeStore{tree: btree.NewBTreeG(TupleOrder)}
}

// Get implements the `Get` function of the `Store` interface.
func (store *BTreeStore) Get(query Tuple) opt.Maybe[Tuple] {
	tuple, found := store.tree.Get(query)
	if found {
		if tuple.IsMatching(query) {
			store.tree.Delete(tuple)
			return opt.NewJust(tuple)
		} else {
			fmt.Printf("[Get] tuple %v does not match query %v\n", tuple, query)
		}
	}
	return opt.NewNothing[Tuple]()
}

// Read implements the `Read` function of the `Store` interface.
func (store *BTreeStore) Read(query Tuple) opt.Maybe[Tuple] {
	tuple, found := store.tree.Get(query)
	if found {
		if tuple.IsMatching(query) {
			return opt.NewJust(tuple)
		} else {
			fmt.Printf("[Read] tuple %v does not match query %v\n", tuple, query)
		}
	}
	return opt.NewNothing[Tuple]()
}

// Write implements the `Write` function of the `Store` interface
// Returns `true` if the tuple was inserted, false otherwise
func (store *BTreeStore) Write(tuple Tuple) bool {
	if !tuple.IsDefined() {
		fmt.Printf("[Write] Warning: attempt to store undefined tuple %v \n", tuple)
		return false
	} else {
		store.tree.Set(tuple)
		return true
	}
}

func (store *BTreeStore) Copy() *BTreeStore {
	clone := NewSimpleStore()

	// Get root node
	root, empty := store.tree.GetAt(0)
	if empty {
		return clone
	}

	store.tree.Ascend(root, func(i Tuple) bool {
		clone.Write(i)
		fmt.Printf("Copied tuple: %v\n", i)
		return true
	})

	return clone
}

func (store *BTreeStore) MarshalJSON() ([]byte, error) {
	var tuples []Tuple

	root, empty := store.tree.GetAt(0)
	if empty {
		return nil, nil
	}

	store.tree.Ascend(root, func(i Tuple) bool {
		tuples = append(tuples, i)
		return true
	})

	result, err := json.Marshal(tuples)
	if err != nil {
		return nil, err
	}

	fmt.Printf("MarshalJSON: %s\n", result)

	return result, nil
}

func (store *BTreeStore) MarshalBinary() ([]byte, error) {
	return store.MarshalJSON()
}
