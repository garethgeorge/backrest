package oplog

import (
	"bytes"
	"os"
	"strconv"

	"github.com/natefinch/atomic"
)

type BigOpDataStore struct {
	path string
}

func NewBigOpDataStore(path string) *BigOpDataStore {
	return &BigOpDataStore{
		path: path,
	}
}

func (s *BigOpDataStore) DeleteBigData(opId int64) error {
	dir := s.path + "/" + strconv.FormatInt(opId, 16)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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

func (s *BigOpDataStore) SetBigData(opId int64, key string, data []byte) error {
	if err := os.MkdirAll(s.path+"/"+strconv.FormatInt(opId, 16), 0755); err != nil {
		return err
	}
	filePath := s.path + "/" + strconv.FormatInt(opId, 16) + "/" + key
	return atomic.WriteFile(filePath, bytes.NewReader(data))
}

func (s *BigOpDataStore) GetBigData(opId int64, key string) ([]byte, error) {
	filePath := s.path + "/" + strconv.FormatInt(opId, 16) + "/" + key
	return os.ReadFile(filePath)
}
