package oplog

import (
	"os"
	"strconv"
)

type BigOpDataStore struct {
	path string
}

func NewBigOpDataStore(path string) *BigOpDataStore {
	return &BigOpDataStore{
		path: path,
	}
}

func (s *BigOpDataStore) DeleteAll(opId int64) error {
	dir := s.path + "/" + strconv.FormatInt(opId, 16)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, dirent := range files {
		if dirent.Name() == "." || dirent.Name() == ".." {
			continue
		}
		if err := os.Remove(dir + "/" + dirent.Name()); err != nil {
			return err
		}
	}
	return os.Remove(dir)
}

func (s *BigOpDataStore) Set(opId int64, key string, data []byte) error {
	filePath := s.path + "/" + strconv.FormatInt(opId, 16) + "/" + key
	return os.WriteFile(filePath, data, 0600)
}

func (s *BigOpDataStore) Get(opId int64, key string) ([]byte, error) {
	filePath := s.path + "/" + strconv.FormatInt(opId, 16) + "/" + key
	return os.ReadFile(filePath)
}
