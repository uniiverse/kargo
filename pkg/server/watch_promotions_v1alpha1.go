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

func (s *server) WatchPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionsRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionsResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	stage := req.Msg.GetStage()

	if stage != "" {
		if err := s.client.Get(ctx, client.ObjectKey{
			Namespace: project,
			Name:      stage,
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	logger := logging.LoggerFromContext(ctx)

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		w, err := s.client.Watch(
			ctx,
			&kargoapi.PromotionList{},
			client.InNamespace(project),
		)
		if err != nil {
			return fmt.Errorf("watch promotion: %w", err)
		}

		if err = s.streamPromotionsEvents(
			ctx, w, stream, keepaliveTicker, stage,
		); err != nil {
			w.Stop()
			return err
		}
		w.Stop()
		logger.Debug("watch channel closed, re-establishing watch")
	}
}

func (s *server) streamPromotionsEvents(
	ctx context.Context,
	w watch.Interface,
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionsResponse],
	keepaliveTicker *time.Ticker,
	stage string,
) error {
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return ctx.Err()
		case <-keepaliveTicker.C:
			if err := stream.Send(&svcv1alpha1.WatchPromotionsResponse{
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
			// FIXME: Current (dynamic) client doesn't support filtering with indexed field by indexer,
			// so manually filter stage here.
			if stage != "" && stage != promotion.Spec.Stage {
				continue
			}
			if err := stream.Send(&svcv1alpha1.WatchPromotionsResponse{
				Promotion: promotion,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
