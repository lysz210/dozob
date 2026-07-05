package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	storageEventChan := make(chan storage.DeviceEvent)

	go storage.ListenForFsPartitions(ctx, storageEventChan)

	devices := make(map[string]storage.Device)

	for {
		select {
		case storageDevice, ok := <-storageEventChan:
			if !ok {
				log.Println("Storage event channel closed.")
				return
			}
			log.Printf("Storage Device Detected: %+v\n", storageDevice)
			switch storageDevice.Action {
			case storage.StorageDeviceAdd:
				devices[storageDevice.Device.DevName] = storageDevice.Device
				log.Printf("Device added: %s", storageDevice.Device.DevName)
				mountedDir, err := storage.MountDevice(storageDevice.Device, cfg.GalleryBasePath)
				if err != nil {
					log.Printf("Failed to mount device %s: %v", storageDevice.Device.DevName, err)
				} else {
					log.Printf("Device %s mounted successfully at %s", storageDevice.Device.DevName, mountedDir)
				}
			case storage.StorageDeviceRemove:
				delete(devices, storageDevice.Device.DevName)
				log.Printf("Device removed: %s", storageDevice.Device.DevName)
			default:
				log.Printf("Unhandled action: %s for device: %s", storageDevice.Action, storageDevice.Device.DevName)
			}
		case <-ctx.Done():
			log.Println("Context canceled, exiting main loop.")
			return
		}
	}
}
