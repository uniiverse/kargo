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

func (s *server) WatchPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      name,
	}, &kargoapi.Promotion{}); err != nil {
		return fmt.Errorf("get promotion: %w", err)
	}

	watchOpts := []client.ListOption{
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	}
	logger := logging.LoggerFromContext(ctx)

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		w, err := s.client.Watch(
			ctx,
			&kargoapi.PromotionList{},
			watchOpts...,
		)
		if err != nil {
			return fmt.Errorf("watch promotion: %w", err)
		}

		if err := s.streamPromotionEvents(
			ctx, w, stream, keepaliveTicker,
		); err != nil {
			w.Stop()
			return err
		}
		w.Stop()
		logger.Debug("watch channel closed, re-establishing watch")
	}
}

func (s *server) streamPromotionEvents(
	ctx context.Context,
	w watch.Interface,
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionResponse],
	keepaliveTicker *time.Ticker,
) error {
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return ctx.Err()
		case <-keepaliveTicker.C:
			if err := stream.Send(&svcv1alpha1.WatchPromotionResponse{
				Type: "KEEPALIVE",
			}); err != nil {
				return fmt.Errorf("send keepalive: %w", err)
			}
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			promotion, ok := e.Object.(*kargoapi.Promotion)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if err := stream.Send(&svcv1alpha1.WatchPromotionResponse{
				Promotion: promotion,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
