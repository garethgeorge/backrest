package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/garethgeorge/resticui/gen/go/types"
	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/internal/orchestrator"
	"github.com/garethgeorge/resticui/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	*v1.UnimplementedResticUIServer
	orchestrator *orchestrator.Orchestrator

	reqId atomic.Uint64
	eventChannelsMu sync.Mutex
	eventChannels map[uint64]chan *v1.Event
}

var _ v1.ResticUIServer = &Server{}

func NewServer(orchestrator *orchestrator.Orchestrator) *Server {
	s := &Server{
		eventChannels: make(map[uint64]chan *v1.Event),
		orchestrator: orchestrator,
	}

	// go func() {
	// 	// TODO: disable this when proper event sources are implemented.
	// 	for {
	// 		time.Sleep(3 * time.Second)
	// 		s.PublishEvent(&v1.Event{
	// 			Event: &v1.Event_Log{
	// 				Log: &v1.LogEvent{
	// 					Message: fmt.Sprintf("event push test, it is %v", time.Now().Format(time.RFC3339)),
	// 				},
	// 			},
	// 		})
	// 	}
	// }()

	return s
}

// GetConfig implements GET /v1/config
func (s *Server) GetConfig(ctx context.Context, empty *emptypb.Empty) (*v1.Config, error) {
	return config.Default.Get()
}

// SetConfig implements POST /v1/config
func (s *Server) SetConfig(ctx context.Context, c *v1.Config) (*v1.Config, error) {
	existing, err := config.Default.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to check current config: %w", err)
	}

	// Compare and increment modno
	if existing.Modno != c.Modno {
		return nil, errors.New("config modno mismatch, reload and try again")
	}
	c.Modno += 1
	
	if err := config.Default.Update(c); err != nil {
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

	c = proto.Clone(c).(*v1.Config)
	c.Repos = append(c.Repos, repo)

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	r := restic.NewRepo(repo)
	 // use background context such that the init op can try to complete even if the connection is closed.
	if err := r.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	zap.S().Debug("Updating config")
	if err := config.Default.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return c, nil
}

// ListSnapshots implements GET /v1/snapshots/{repo.id}/{plan.id?}
func (s *Server) ListSnapshots(ctx context.Context, query *v1.ListSnapshotsRequest) (*v1.ResticSnapshotList, error) {
	repo, err := s.orchestrator.GetRepo(query.RepoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	var snapshots []*restic.Snapshot
	if query.PlanId != "" {
		var plan *v1.Plan
		plan, err = s.orchestrator.GetPlan(query.PlanId)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan %q: %w", query.PlanId, err)
		}

		snapshots, err = repo.SnapshotsForPlan(ctx, plan)
	} else {
		snapshots, err = repo.Snapshots(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Transform the snapshots and return them.
	var rs []*v1.ResticSnapshot
	for _, snapshot := range snapshots {
		rs = append(rs, snapshot.ToProto())
	}

	return &v1.ResticSnapshotList{
		Snapshots: rs,
	}, nil
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