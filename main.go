package quev2

import (
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	MaxItemToRetrieve = 10
	cache             *dbCache
)

func init() {
	cache = NewDBCache()
}

type dbCache struct {
	m map[string]map[string]string
}

func NewDBCache() *dbCache {
	return &dbCache{
		m: make(map[string]map[string]string),
	}
}

type item struct {
	V    string
	when *time.Time
}

type Quedb struct {
	b *bolt.DB
}

func (qdb *Quedb) Set(listname, data string) error {
	return qdb.b.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(listname))
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(data), []byte(""))
		if err != nil {
			return err
		}
		return nil
	})
}

func (qdb *Quedb) Remove(from, item string) error {
	return qdb.b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(from))
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(item))
	})
}

func (qdb *Quedb) Move(from, to, item string) error {
	return qdb.b.Update(func(tx *bolt.Tx) error {
		fromB, _ := tx.CreateBucketIfNotExists([]byte(from))
		toB, _ := tx.CreateBucketIfNotExists([]byte(to))

		fromB.Delete([]byte(item))
		toB.Put([]byte(item), []byte(""))
		return nil
	})
}

func (qdb *Quedb) GetAndMove(from, to string) ([]string, error) {

	items, err := qdb.Get(from)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		qdb.Move(from, to, item)
	}

	return items, nil
}

func (qdb *Quedb) Get(listname string) ([]string, error) {
	var result []string
	qdb.b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(listname))
		if bucket == nil {
			return nil
		}

		var x = 0
		bucket.ForEach(func(k, v []byte) error {
			result = append(result, string(k))
			if MaxItemToRetrieve == x {
				return errors.New("max reached")
			}
			x++
			return nil
		})
		return nil
	})
	//TODO: Delete Befoe Exit
	return result, nil
}

func (qdb *Quedb) Close() {
	if qdb != nil && qdb.b != nil {
		qdb.Close()
	}
}

func NewQDB(path ...string) (*Quedb, error) {
	var dbPath = "./que.dat"
	if len(path) > 0 {
		dbPath = path[0]
	}

	db, err := bolt.Open(dbPath, 0777, &bolt.Options{})
	if err != nil {
		return nil, err
	}
	return &Quedb{b: db}, nil
}
