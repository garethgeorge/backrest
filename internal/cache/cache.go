package cache

import (
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog/serializationutil"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

var (
	bucketRepoStats = []byte("repo_stats")
)

var (
	ErrNotFound = errors.New("not found")
)

type Cache struct {
	db *bbolt.DB
}

func NewCache(databasePath string) (*Cache, error) {
	db, err := bbolt.Open(databasePath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range [][]byte{bucketRepoStats} {
			_, err := tx.CreateBucketIfNotExists(bucket)
			return err
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return &Cache{db: db}, nil
}

func (c *Cache) Close() error {
	return c.db.Close()
}

func (c *Cache) SetRepoStats(repoId string, stats *v1.RepoStats) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketRepoStats)
		statsBytes, err := proto.Marshal(stats)
		if err != nil {
			return fmt.Errorf("marshal stats: %w", err)
		}
		return b.Put(serializationutil.BytesToKey([]byte(repoId)), statsBytes)
	})
}

func (c *Cache) GetRepoStats(repoId string) (*v1.RepoStats, error) {
	var stats v1.RepoStats
	if err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketRepoStats)
		statsBytes := b.Get(serializationutil.BytesToKey([]byte(repoId)))
		if statsBytes == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(statsBytes, &stats)
	}); err != nil {
		return nil, err
	}
	return &stats, nil
}
