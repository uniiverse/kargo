package server

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchClusterConfig(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.WatchClusterConfigRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchClusterConfigResponse],
) error {
	if err := s.client.Get(ctx, client.ObjectKey{
		Name: api.ClusterConfigName,
	}, &kargoapi.ClusterConfig{}); err != nil {
		return fmt.Errorf("get ClusterConfig: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		w, err := s.client.Watch(
			ctx,
			&kargoapi.ClusterConfigList{},
			client.MatchingFields{"metadata.name": api.ClusterConfigName},
		)
		if err != nil {
			return fmt.Errorf("watch ClusterConfig: %w", err)
		}

		if err := s.streamClusterConfigEvents(
			ctx, w, stream, keepaliveTicker,
		); err != nil {
			w.Stop()
			return err
		}
		w.Stop()
		logger.Debug("watch channel closed, re-establishing watch")
	}
}

func (s *server) streamClusterConfigEvents(
	ctx context.Context,
	w watch.Interface,
	stream *connect.ServerStream[svcv1alpha1.WatchClusterConfigResponse],
	keepaliveTicker *time.Ticker,
) error {
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return ctx.Err()
		case <-keepaliveTicker.C:
			if err := stream.Send(&svcv1alpha1.WatchClusterConfigResponse{
				Type: "KEEPALIVE",
			}); err != nil {
				return fmt.Errorf("send keepalive: %w", err)
			}
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			config, ok := e.Object.(*kargoapi.ClusterConfig)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if err := stream.Send(&svcv1alpha1.WatchClusterConfigResponse{
				ClusterConfig: config,
				Type:          string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
