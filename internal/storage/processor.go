package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	// Replace this with your actual module path matching your go.mod
	pb "github.com/lysz210/dozob/pkg/storage"

	"google.golang.org/grpc"
)

type StorageProcessor struct {
	client pb.StorageBridgeClient
}

func NewProcessor(conn *grpc.ClientConn) *StorageProcessor {
	return &StorageProcessor{
		client: pb.NewStorageBridgeClient(conn),
	}
}

func (sp *StorageProcessor) HandleDeviceEvent(
	ctx context.Context,
	event DeviceEvent,
) error {
	var action pb.DeviceAction

	switch event.Action {
	case StorageDeviceAdd:
		action = pb.DeviceAction_DEVICE_ACTION_ADD
	case StorageDeviceRemove:
		action = pb.DeviceAction_DEVICE_ACTION_REMOVE
	default:
		return fmt.Errorf("unknown device action: %s", event.Action)
	}

	device := &pb.Device{
		DevName:   event.Device.DevName,
		Bus:       event.Device.Bus,
		FsUuid:    event.Device.FsUuid,
		FsLabel:   event.Device.FsLabel,
		FsSize:    event.Device.FsSize,
		FsType:    event.Device.FsType,
		FsVersion: event.Device.FsVersion,
		Model:     event.Device.Model,
	}

	req := &pb.DeviceEvent{
		Action: action,
		Device: device,
	}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := sp.client.ReportUdevEvent(callCtx, req)
	if err != nil {
		return fmt.Errorf("failed to report udev event: %w", err)
	}

	log.Println("Successfully reported udev event to server")
	return nil
}

func (sp *StorageProcessor) StartCommandStream(
	ctx context.Context,
	mountDeviceChan chan<- Device,
) {
	defer close(mountDeviceChan)
	stream, err := sp.client.StreamMountCommands(ctx, &pb.EmptyRequest{})
	if err != nil {
		log.Printf("Failed to open mount command stream: %v", err)
		return
	}

	log.Println("gRPC Mount Command stream established. Forwarding to channel...")

	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream cleanly closed by Quarkus server.")
			break
		}
		if err != nil {
			log.Printf("Error reading from gRPC stream: %v", err)
			break
		}

		// Push the network event directly into the Go channel.
		// Note: If the channel is unbuffered and nothing is reading from it,
		// this line will safely block until a consumer reads it.
		select {
		case mountDeviceChan <- Device{
			DevName:   cmd.Device.DevName,
			Bus:       cmd.Device.Bus,
			FsUuid:    cmd.Device.FsUuid,
			FsLabel:   cmd.Device.FsLabel,
			FsSize:    cmd.Device.FsSize,
			FsType:    cmd.Device.FsType,
			FsVersion: cmd.Device.FsVersion,
			Model:     cmd.Device.Model,
		}:
		case <-ctx.Done():
			return
		}
	}
}
