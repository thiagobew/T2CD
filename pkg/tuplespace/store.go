package tuplespace

import (
	"context"
	"fmt"

	"github.com/atomix/go-sdk/pkg/atomix"
	_map "github.com/atomix/go-sdk/pkg/primitive/map"
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

type AtomixStore struct {
}

func NewDistributedStore() *AtomixStore {
	minIdCounter, err := atomix.Counter("minId").Get(context.Background())
	defer minIdCounter.Close(context.Background())
	if err != nil {
		fmt.Println(err)
		return nil
	}

	maxIdCounter, err := atomix.Counter("maxId").Get(context.Background())
	defer maxIdCounter.Close(context.Background())
	if err != nil {
		fmt.Println(err)
		return nil
	}

	err = minIdCounter.Set(context.Background(), 0)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	err = maxIdCounter.Set(context.Background(), 0)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	fmt.Printf("Initialize AtomixStore with minId: %d, maxId: %d\n", minIdCounter, maxIdCounter)

	return &AtomixStore{}
}

func (store *AtomixStore) getMinMaxIds() (int64, int64) {
	minIdCounter, err := atomix.Counter("minId").Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1, -1
	}
	defer func() {
		if err := minIdCounter.Close(context.Background()); err != nil {
			fmt.Println(err)
		}
	}()

	maxIdCounter, err := atomix.Counter("maxId").Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1, -1
	}
	defer func() {
		if err := maxIdCounter.Close(context.Background()); err != nil {
			fmt.Println(err)
		}
	}()

	minId, err := minIdCounter.Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1, -1
	}

	maxId, err := maxIdCounter.Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return -1, -1
	}

	return minId, maxId
}

func (store *AtomixStore) Read(tuple Tuple) opt.Maybe[Tuple] {
	minId, maxId := store.getMinMaxIds()

	if minId == -1 || maxId == -1 {
		fmt.Println("[Read] Error reading minId and maxId")
		return opt.NewNothing[Tuple]()
	}
	fmt.Printf("[Read] Read minId: %d, maxId: %d\n", minId, maxId)
	for i := minId; i <= maxId; i++ {
		fmt.Printf("[Read] Reading tuple with id: %d\n", i)
		tuples, err := atomix.Map[int64, []byte]("tuplespace").Get(context.Background())
		if err != nil {
			fmt.Println(err)
			return opt.NewNothing[Tuple]()
		}
		defer func() {
			if closeErr := tuples.Close(context.Background()); closeErr != nil {
				fmt.Println(closeErr)
			}
		}()

		mapEntry, err := tuples.Get(context.Background(), i)
		if err != nil {
			fmt.Println(err)
			return opt.NewNothing[Tuple]()
		}

		var tupleValue []byte = mapEntry.Value

		tuple := DecodeTuple(tupleValue)
		fmt.Printf("[Read] Read tuple: %v\n", tuple)
		if tuple.IsMatching(tuple) {
			return opt.NewJust(tuple)
		}
	}

	return opt.NewNothing[Tuple]()
}

func (store *AtomixStore) Get(query Tuple) opt.Maybe[Tuple] {
	minId, maxId := store.getMinMaxIds()

	if minId == -1 || maxId == -1 {
		fmt.Println("[Get] Error reading minId and maxId")
		return opt.NewNothing[Tuple]()
	}

	for i := minId; i <= maxId; i++ {
		fmt.Printf("[Get] Reading tuple with id: %d\n", i)
		tuples, err := atomix.Map[int64, []byte]("tuplespace").Get(context.Background())
		if err != nil {
			fmt.Println(err)
			return opt.NewNothing[Tuple]()
		}
		defer func() {
			if closeErr := tuples.Close(context.Background()); closeErr != nil {
				fmt.Println(closeErr)
			}
		}()

		mapEntry, err := tuples.Get(context.Background(), i)
		if err != nil {
			fmt.Println(err)
			return opt.NewNothing[Tuple]()
		}

		var tupleValue []byte = mapEntry.Value

		tuple := DecodeTuple(tupleValue)
		fmt.Printf("[Get] Read tuple: %v\n", tuple)
		if tuple.IsMatching(query) {
			_, err := tuples.Remove(context.Background(), i, _map.IfVersion(mapEntry.Version))
			if err != nil {
				fmt.Println(err)
				return opt.NewNothing[Tuple]()
			}

			// Update minId
			minIdCounter, err := atomix.Counter("minId").Get(context.Background())
			if err != nil {
				fmt.Println(err)
				return opt.NewNothing[Tuple]()
			}

			if i == minId {
				err = minIdCounter.Set(context.Background(), i+1)
				if err != nil {
					fmt.Println(err)
				}
			}

			minIdCounter.Close(context.Background())
			return opt.NewJust(tuple)
		}
	}

	return opt.NewNothing[Tuple]()
}

func (store *AtomixStore) Write(tuple Tuple) bool {

	maxIdCounter, err := atomix.Counter("maxId").Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer func() {
		if closeErr := maxIdCounter.Close(context.Background()); closeErr != nil {
			fmt.Println(closeErr)
		}
	}()

	tuples, err := atomix.Map[int64, []byte]("tuplespace").Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return false
	}

	defer func() {
		if closeErr := tuples.Close(context.Background()); closeErr != nil {
			fmt.Println(closeErr)
		}
	}()

	// Update maxId
	newId, err := maxIdCounter.Increment(context.Background(), 1)
	if err != nil {
		fmt.Println(err)
		return false
	}

	_, err = tuples.Put(context.Background(), newId, EncodeTuple(tuple))
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
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
