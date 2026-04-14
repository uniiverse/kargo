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
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchProjectConfigRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchProjectConfigResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      project,
	}, &kargoapi.ProjectConfig{}); err != nil {
		return fmt.Errorf("get projectconfig: %w", err)
	}

	watchOpts := []client.ListOption{
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": project},
	}
	logger := logging.LoggerFromContext(ctx)

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		w, err := s.client.Watch(
			ctx,
			&kargoapi.ProjectConfigList{},
			watchOpts...,
		)
		if err != nil {
			return fmt.Errorf("watch ProjectConfig: %w", err)
		}

		if err := s.streamProjectConfigEvents(
			ctx, w, stream, keepaliveTicker,
		); err != nil {
			w.Stop()
			return err
		}
		w.Stop()
		logger.Debug("watch channel closed, re-establishing watch")
	}
}

func (s *server) streamProjectConfigEvents(
	ctx context.Context,
	w watch.Interface,
	stream *connect.ServerStream[svcv1alpha1.WatchProjectConfigResponse],
	keepaliveTicker *time.Ticker,
) error {
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return ctx.Err()
		case <-keepaliveTicker.C:
			if err := stream.Send(&svcv1alpha1.WatchProjectConfigResponse{
				Type: "KEEPALIVE",
			}); err != nil {
				return fmt.Errorf("send keepalive: %w", err)
			}
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			config, ok := e.Object.(*kargoapi.ProjectConfig)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if err := stream.Send(&svcv1alpha1.WatchProjectConfigResponse{
				ProjectConfig: config,
				Type:          string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
