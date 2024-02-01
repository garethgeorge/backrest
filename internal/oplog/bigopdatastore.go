package oplog

import (
	"bytes"
	"os"
	"strconv"

	"github.com/natefinch/atomic"
	"go.uber.org/zap"
)

type BigOpDataStore struct {
	path string
}

func NewBigOpDataStore(path string) *BigOpDataStore {
	return &BigOpDataStore{
		path: path,
	}
}

func (s *BigOpDataStore) resolvePath(opId int64) string {
	return s.path + "/" + strconv.FormatInt(opId&0xFF, 16) + "/" + strconv.FormatInt(opId, 16)
}

func (s *BigOpDataStore) DeleteOperationData(opId int64) error {
	dir := s.resolvePath(opId)
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
			zap.S().Errorf("deleting big operation data for operation %v failed to delete %v: %v", opId, dirent.Name(), err)
			continue
		}
	}
	return os.Remove(dir)
}

func (s *BigOpDataStore) SetBigData(opId int64, key string, data []byte) error {
	dir := s.resolvePath(opId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return atomic.WriteFile(dir+"/"+key, bytes.NewReader(data))
}

func (s *BigOpDataStore) GetBigData(opId int64, key string) ([]byte, error) {
	return os.ReadFile(s.resolvePath(opId) + "/" + key)
}

func (s *BigOpDataStore) DeleteBigData(opId int64, key string) error {
	return os.Remove(s.resolvePath(opId) + "/" + key)
}
