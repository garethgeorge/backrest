package api

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	*v1.UnimplementedResticUIServer
}

var _ v1.ResticUIServer = &server{}

func (s *server) GetConfig(ctx context.Context, empty *emptypb.Empty) (*v1.Config, error) {
	return config.Default.Get()
}

func (s *server) SetConfig(ctx context.Context, c *v1.Config) (*v1.Config, error) {
	err := config.Default.Update(c)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return config.Default.Get()
}

func (s *server) GetEvents(_ *emptypb.Empty, stream v1.ResticUI_GetEventsServer) error {
	for {
		zap.S().Info("Sending event")
		stream.Send(&v1.Event{
			Timestamp: 0,
			Event: &v1.Event_BackupStatusChange{
				BackupStatusChange: &v1.BackupStatusEvent{
					Status: v1.Status_IN_PROGRESS,
					Percent: 0,
					Plan: "myplan",
				},
			},
		})

		timer := time.NewTimer(time.Second * 1)

		select {
		case <-stream.Context().Done():
			zap.S().Info("Get events hangup")
			return nil
		case <-timer.C:
		}
	}
}