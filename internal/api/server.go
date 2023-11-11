package api

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
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

// SetConfig implements PUT /v1/config
func (s *Server) SetConfig(ctx context.Context, c *v1.Config) (*v1.Config, error) {
	err := config.Default.Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return config.Default.Get()
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

// PublishEvent publishes an event to all GetEvents streams. It is effectively a multicast.
func (s *Server) PublishEvent(event *v1.Event) {
	zap.S().Debug("Publishing event", zap.Any("event", event))
	s.eventChannelsMu.Lock()
	defer s.eventChannelsMu.Unlock()
	for _, ch := range s.eventChannels {
		ch <- event
	}
}