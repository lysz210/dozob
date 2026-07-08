package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lysz210/dozob/internal/config"
	"github.com/lysz210/dozob/internal/storage"
)

func main() {
	cfg := config.Get()
	log.Println("Listening for ALL kernel events... Plug in your pendrive now.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	grpcConn, err := grpc.NewClient(cfg.GRPCServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer grpcConn.Close()

	storageProcessor := storage.NewProcessor(grpcConn)

	storageEventChan := make(chan storage.DeviceEvent)
	mountDeviceChan := make(chan storage.Device)

	go storage.ListenForFsPartitions(ctx, storageEventChan)

	go storageProcessor.StartCommandStream(ctx, mountDeviceChan)

	devices := make(map[string]storage.Device)

	for {
		select {
		case deviceEvent, ok := <-storageEventChan:
			if !ok {
				log.Println("Storage event channel closed.")
				return
			}
			log.Printf("Storage Device Detected: %+v\n", deviceEvent)
			err := storageProcessor.HandleDeviceEvent(ctx, deviceEvent)
			if err != nil {
				log.Printf("Failed to report udev event: %v", err)
			}
			switch deviceEvent.Action {
			case storage.StorageDeviceAdd:
				devices[deviceEvent.Device.DevName] = deviceEvent.Device
				log.Printf("Device added: %s", deviceEvent.Device.DevName)
			case storage.StorageDeviceRemove:
				delete(devices, deviceEvent.Device.DevName)
				log.Printf("Device removed: %s", deviceEvent.Device.DevName)
			default:
				log.Printf("Unhandled action: %s for device: %s", deviceEvent.Action, deviceEvent.Device.DevName)
			}
		case mountDevice, ok := <-mountDeviceChan:
			if !ok {
				log.Println("Mount device channel closed.")
				return
			}
			log.Printf("Received mount command for device: %+v\n", mountDevice)
			log.Printf("Mounted on: %s", cfg.GalleryBasePath)

			mountedDir, err := storage.MountDevice(mountDevice, cfg.GalleryBasePath)
			if err != nil {
				log.Printf("Failed to mount device %s: %v", mountDevice.DevName, err)
			} else {
				log.Printf("Device %s mounted successfully at %s", mountDevice.DevName, mountedDir)
			}
		case <-ctx.Done():
			log.Println("Context canceled, exiting main loop.")
			return
		}
	}
}
