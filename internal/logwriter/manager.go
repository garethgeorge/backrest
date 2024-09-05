package logwriter

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

type LogManager struct {
	llm *LiveLog
	rlm *RotatingLog
}

func NewLogManager(dir string, maxLogFiles int) (*LogManager, error) {
	ll, err := NewLiveLogger(path.Join(dir, ".live"))
	if err != nil {
		return nil, err
	}

	rl := NewRotatingLog(path.Join(dir), maxLogFiles)
	if err != nil {
		return nil, err
	}

	return &LogManager{
		llm: ll,
		rlm: rl,
	}, nil
}

func (lm *LogManager) NewLiveWriter(opID int64) (string, io.WriteCloser, error) {
	id := fmt.Sprintf("%d.livelog", opID)
	w, err := lm.llm.NewWriter(id)
	return id, w, err
}

func (lm *LogManager) Subscribe(id string) (chan []byte, error) {
	if strings.HasSuffix(id, ".livelog") {
		return lm.llm.Subscribe(id)
	} else {
		// TODO: implement streaming from rotating log storage
		ch := make(chan []byte, 1)
		data, err := lm.rlm.Read(id)
		if err != nil {
			return nil, err
		}
		ch <- data
		close(ch)
		return ch, nil
	}
}

func (lm *LogManager) Unsubscribe(id string, ch chan []byte) {
	lm.llm.Unsubscribe(id, ch)
}

// LiveLogIDs returns the list of IDs of live logs e.g. with writes in progress.
func (lm *LogManager) LiveLogIDs() []string {
	return lm.llm.ListIDs()
}

func (lm *LogManager) Finalize(id string) (frozenID string, err error) {
	if lm.llm.IsAlive(id) {
		return "", errors.New("live log still being written")
	}

	ch, err := lm.llm.Subscribe(id)
	if err != nil {
		return "", err
	}

	bytes := make([]byte, 0)
	for data := range ch {
		bytes = append(bytes, data...)
	}

	return lm.rlm.Write(bytes)
}
