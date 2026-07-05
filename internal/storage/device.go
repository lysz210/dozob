package storage

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"

	"github.com/pilebones/go-udev/netlink"
)

type Device struct {
	DevName   string `json:"dev_name"`
	Bus       string `json:"bus"`
	FsUuid    string `json:"fs_uuid"`
	FsLabel   string `json:"fs_label"`
	FsSize    int64  `json:"fs_size"`
	FsType    string `json:"fs_type"`
	FsVersion string `json:"fs_version"`
	Model     string `json:"model"`
}

type DeviceAction string

const (
	StorageDeviceAdd    DeviceAction = "add"
	StorageDeviceRemove DeviceAction = "remove"
)

type DeviceEvent struct {
	Action DeviceAction `json:"action"`
	Device Device       `json:"device"`
}

func ListenForFsPartitions(ctx context.Context, storageEventChan chan<- DeviceEvent) {
	defer close(storageEventChan)

	conn := netlink.UEventConn{}
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		log.Fatalf("FATAL: Cannot connect to netlink socket: %v (Did you forget sudo?)", err)
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent)
	errors := make(chan error)

	go conn.Monitor(queue, errors, nil)

	for {
		select {
		case event := <-queue:

			action := event.Action

			env := event.Env

			fsUsage := env["ID_FS_USAGE"]

			if fsUsage != "filesystem" {
				continue
			}

			if !(action == netlink.ADD || action == netlink.REMOVE) {
				continue
			}

			size, err := strconv.ParseInt(env["ID_FS_SIZE"], 10, 64)
			if err != nil {
				log.Printf("Failed to parse ID_FS_SIZE: %v", err)
				size = 0
			}
			storageDevice := Device{
				DevName:   env["DEVNAME"],
				Bus:       env["ID_BUS"],
				FsUuid:    env["ID_FS_UUID"],
				FsLabel:   env["ID_FS_LABEL"],
				FsSize:    size,
				FsType:    env["ID_FS_TYPE"],
				FsVersion: env["ID_FS_VERSION"],
				Model:     env["ID_MODEL"],
			}
			storageEventChan <- DeviceEvent{
				Action: DeviceAction(action),
				Device: storageDevice,
			}

			log.Printf("[Event Captured] Action: %s | fsUsage: %s\n", action, fsUsage)

		case err := <-errors:
			log.Printf("Error: %v", err)
		case <-ctx.Done():
			log.Println("Exiting.")
			return
		}
	}
}

func MountDevice(device Device, baseDir string) (string, error) {

	targetDir := filepath.Join(baseDir, device.FsUuid)
	// 1. Ensure the mount point directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return targetDir, err
	}

	// 2. Set up our mount flags
	// 0 means standard read-write mount options
	var flags uintptr = unix.MS_RDONLY

	// 3. Execute the Linux syscall
	// data = "" (used if you want to pass extra filesystem-specific strings, like uid/gid parameters)
	err := unix.Mount(device.DevName, targetDir, device.FsType, flags, "")
	if err != nil {
		return targetDir, err
	}

	return targetDir, nil
}
