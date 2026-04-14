package server

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/watch"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchWarehousesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchWarehousesResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	name := req.Msg.GetName()

	if name != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: project,
			Name:      name,
		}, &kargoapi.Warehouse{}); err != nil {
			return fmt.Errorf("get warehouse: %w", err)
		}
	}

	watchOpts := []libClient.ListOption{libClient.InNamespace(project)}
	if name != "" {
		watchOpts = append(watchOpts, libClient.MatchingFields{"metadata.name": name})
	}
	logger := logging.LoggerFromContext(ctx)

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		w, err := s.client.Watch(ctx, &kargoapi.WarehouseList{}, watchOpts...)
		if err != nil {
			return fmt.Errorf("watch warehouse: %w", err)
		}

		if err := s.streamWarehouseEvents(
			ctx, w, stream, keepaliveTicker,
		); err != nil {
			w.Stop()
			return err
		}
		w.Stop()
		logger.Debug("watch channel closed, re-establishing watch")
	}
}

func (s *server) streamWarehouseEvents(
	ctx context.Context,
	w watch.Interface,
	stream *connect.ServerStream[svcv1alpha1.WatchWarehousesResponse],
	keepaliveTicker *time.Ticker,
) error {
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return ctx.Err()
		case <-keepaliveTicker.C:
			if err := stream.Send(&svcv1alpha1.WatchWarehousesResponse{
				Type: "KEEPALIVE",
			}); err != nil {
				return fmt.Errorf("send keepalive: %w", err)
			}
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			warehouse, ok := e.Object.(*kargoapi.Warehouse)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			// Necessary because serializing a Warehouse as part of a protobuf
			// message does not apply custom marshaling. The call to this helper
			// compensates for that.
			if err := prepareOutboundWarehouse(warehouse); err != nil {
				return fmt.Errorf("prepare outbound warehouse: %w", err)
			}
			if err := stream.Send(&svcv1alpha1.WatchWarehousesResponse{
				Warehouse: warehouse,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
