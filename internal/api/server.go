package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/garethgeorge/resticui/gen/go/types"
	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	*v1.UnimplementedResticUIServer

	reqId atomic.Uint64
	eventChannelsMu sync.Mutex
	eventChannels map[uint64]chan *v1.Event
}

var _ v1.ResticUIServer = &Server{}

func NewServer() *Server {
	s := &Server{
		eventChannels: make(map[uint64]chan *v1.Event),
	}

	go func() {
		for {
			time.Sleep(3 * time.Second)
			s.PublishEvent(&v1.Event{
				Event: &v1.Event_Log{
					Log: &v1.LogEvent{
						Message: fmt.Sprintf("event push test, it is %v", time.Now().Format(time.RFC3339)),
					},
				},
			})
		}
	}()

	return s
}

// GetConfig implements GET /v1/config
func (s *Server) GetConfig(ctx context.Context, empty *emptypb.Empty) (*v1.Config, error) {
	return config.Default.Get()
}

// SetConfig implements POST /v1/config
func (s *Server) SetConfig(ctx context.Context, c *v1.Config) (*v1.Config, error) {
	err := config.Default.Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return config.Default.Get()
}

// AddRepo implements POST /v1/config/repo, it includes validation that the repo can be initialized.
func (s *Server) AddRepo(ctx context.Context, repo *v1.Repo) (*v1.Config, error) {
	c, err := config.Default.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	r := restic.NewRepo(repo)
	 // use background context such that the init op can try to complete even if the connection is closed.
	if err := r.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}
	
	c.Repos = append(c.Repos, repo)

	if err := config.Default.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return c, nil
}


// GetEvents implements GET /v1/events
func (s *Server) GetEvents(_ *emptypb.Empty, stream v1.ResticUI_GetEventsServer) error {
	reqId := s.reqId.Add(1)

	// Register a channel to receive events for this invocation
	s.eventChannelsMu.Lock()
	eventChan := make(chan *v1.Event, 3)
	s.eventChannels[reqId] = eventChan
	s.eventChannelsMu.Unlock()
	defer delete(s.eventChannels, reqId)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-eventChan:
			stream.Send(event)
		}
	}
}

func (s *Server) PathAutocomplete(ctx context.Context, path *types.StringValue) (*types.StringList, error) {
	ents, err := os.ReadDir(path.Value)
	if errors.Is(err, os.ErrNotExist) {
		return &types.StringList{}, nil
	} else if err != nil {
		return nil, err
	}

	var paths []string
	for _, ent := range ents {
		paths = append(paths, ent.Name())
	}

	return &types.StringList{Values: paths}, nil
}


// PublishEvent publishes an event to all GetEvents streams. It is effectively a multicast.
func (s *Server) PublishEvent(event *v1.Event) {
	zap.S().Debug("Publishing event", zap.Any("event", event))
	s.eventChannelsMu.Lock()
	defer s.eventChannelsMu.Unlock()
	for _, ch := range s.eventChannels {
		ch <- event
	}
}