package quev2

import (
	"errors"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
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

func (qdb *Quedb) Move(to, item string) error {
	return qdb.b.Update(func(tx *bolt.Tx) error {
		fromB, _ := tx.CreateBucketIfNotExists([]byte("in_flight"))
		toB, _ := tx.CreateBucketIfNotExists([]byte(to))

		fromB.Delete([]byte(item))
		toB.Put([]byte(item), []byte(""))
		return nil
	})
}

func (qdb *Quedb) List(skip, limit int, listname string) ([]*Item, error) {
	var result []*Item

	err := qdb.b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(listname))
		if b == nil {
			return errors.New("channel does not exist")
		}

		c := b.Cursor()
		x := 0
		for {
			k, v := c.Next()
			if k == nil {
				return nil
			}

			if skip > 0 && x < skip {
				x++
				continue
			}

			result = append(result, &Item{Listname: listname, Key: string(k), Value: string(v)})
			if limit > 0 && len(result) == limit {
				return nil
			}
		}
	})
	return result, err
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

	for _, item := range result {
		qdb.Set("in_flight", item)
		qdb.Remove(listname, item)
	}

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

type Item struct {
	Listname string
	Key      string
	Value    string
}

type OutgoingMessage struct {
	Action  string      `json:"action"`
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

func NewV4() string {
	v4 := uuid.NewV4()
	return strings.Replace(v4.String(), "-", "", -1)
}
