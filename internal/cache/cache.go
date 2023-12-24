package cache

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.etcd.io/bbolt"
)

type Cache struct {
	db *bbolt.DB
}

func NewCache(databasePath string) (*Cache, error) {
	db, err := bbolt.Open(databasePath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Cache{db: db}, nil
}

func (c *Cache) Close() error {
	return c.db.Close()
}

func (c *Cache) SetRepoStats(repo *v1.Repo, stats *v1.RepoStats) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("repo_stats"))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		return b.Put([]byte(repo.Id), protoutil.MustMarshal(stats))
	})
}
