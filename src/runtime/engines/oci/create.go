// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/layout"
	"github.com/singularityware/singularity/src/pkg/util/fs/layout/layer/overlay"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/sif/pkg/sif"
)

var session *layout.Session
var rpcOps *client.RPC

func (engine *EngineOperations) localMount(point *mount.Point) error {
	if _, err := mount.GetOffset(point.InternalOptions); err == nil {
		if err := engine.mountImage(point); err != nil {
			return fmt.Errorf("can't mount image %s: %s", point.Source, err)
		}
	} else {
		if err := engine.mountGeneric(point, true); err != nil {
			flags, _ := mount.ConvertOptions(point.Options)
			if flags&syscall.MS_REMOUNT != 0 {
				return fmt.Errorf("can't remount %s: %s", point.Destination, err)
			}
			sylog.Verbosef("can't mount %s: %s", point.Source, err)
			return nil
		}
	}
	return nil
}

func (engine *EngineOperations) rpcMount(point *mount.Point) error {
	if err := engine.mountGeneric(point, false); err != nil {
		flags, _ := mount.ConvertOptions(point.Options)
		if flags&syscall.MS_REMOUNT != 0 {
			return fmt.Errorf("can't remount %s: %s", point.Destination, err)
		}
		sylog.Verbosef("can't mount %s: %s", point.Source, err)
		return nil
	}
	return nil
}

func (engine *EngineOperations) switchMount(system *mount.System) error {
	system.Mount = engine.rpcMount
	return nil
}

// mount any generic mount (not loop dev)
func (engine *EngineOperations) mountGeneric(mnt *mount.Point, local bool) (err error) {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")
	sessionPath := session.Path()
	remount := false

	if flags&syscall.MS_REMOUNT != 0 {
		remount = true
	}

	if flags&syscall.MS_BIND != 0 && !remount {
		if _, err := os.Stat(mnt.Source); os.IsNotExist(err) {
			sylog.Debugf("Skipping mount, host source %s doesn't exist", mnt.Source)
			return nil
		}
	}

	dest := ""
	if !strings.HasPrefix(mnt.Destination, sessionPath) {
		dest = session.FinalPath() + mnt.Destination
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			sylog.Debugf("Skipping mount, %s doesn't exist in container", dest)
			return nil
		}
	} else {
		dest = mnt.Destination
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			return fmt.Errorf("destination %s doesn't exist", dest)
		}
	}

	if remount {
		sylog.Debugf("Remounting %s\n", dest)
	} else {
		sylog.Debugf("Mounting %s to %s\n", mnt.Source, dest)
	}
	if !local {
		_, err = rpcOps.Mount(mnt.Source, dest, mnt.Type, flags, optsString)
	} else {
		err = syscall.Mount(mnt.Source, dest, mnt.Type, flags, optsString)
	}
	return err
}

// mount image via loop
func (engine *EngineOperations) mountImage(mnt *mount.Point) error {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")

	offset, err := mount.GetOffset(mnt.InternalOptions)
	if err != nil {
		return err
	}

	sizelimit, err := mount.GetSizeLimit(mnt.InternalOptions)
	if err != nil {
		return err
	}

	attachFlag := os.O_RDWR
	loopFlags := uint32(loop.FlagsAutoClear)

	if flags&syscall.MS_RDONLY == 1 {
		loopFlags |= loop.FlagsReadOnly
		attachFlag = os.O_RDONLY
	}

	info := &loop.Info64{
		Offset:    offset,
		SizeLimit: sizelimit,
		Flags:     loopFlags,
	}

	loopdev := new(loop.Device)

	number := 0

	if err := loopdev.Attach(mnt.Source, attachFlag, &number); err != nil {
		return err
	}
	if err := loopdev.SetStatus(info); err != nil {
		return err
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, mnt.Destination)
	err = syscall.Mount(path, mnt.Destination, mnt.Type, flags, optsString)
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mnt.Type, err)
	}

	return nil
}

func (engine *EngineOperations) addRootfsMount(system *mount.System) error {
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NODEV)
	rootfs := engine.EngineConfig.JSON.Image
	writable := false

	imageObject, err := image.Init(rootfs, writable)
	if err != nil {
		return err
	}

	mountType := ""

	switch imageObject.Type {
	case image.SIF:
		// Load the SIF file
		fimg, err := sif.LoadContainerFp(imageObject.File, !imageObject.Writable)
		if err != nil {
			return err
		}

		// Get the default system partition image
		part, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
		if err != nil {
			return err
		}

		// Check that this is a system partition
		parttype, err := part.GetPartType()
		if err != nil {
			return err
		}
		if parttype != sif.PartSystem {
			return fmt.Errorf("found partition is not system")
		}

		// record the fs type
		fstype, err := part.GetFsType()
		if err != nil {
			return err
		}
		if fstype == sif.FsSquash {
			if writable {
				return fmt.Errorf("can't set writable flag with squashfs image")
			}
			mountType = "squashfs"
		} else if fstype == sif.FsExt3 {
			mountType = "ext3"
		} else {
			return fmt.Errorf("unknown file system type: %v", fstype)
		}

		imageObject.Offset = uint64(part.Fileoff)
		imageObject.Size = uint64(part.Filelen)
	}

	src := fmt.Sprintf("/proc/self/fd/%d", imageObject.File.Fd())
	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	return system.Points.AddImage(mount.RootfsTag, src, session.RootFsPath(), mountType, flags, imageObject.Offset, imageObject.Size)
}

func (engine *EngineOperations) createEtc(system *mount.System) error {
	ov := session.Layer.(*overlay.Overlay)
	return ov.AddDir("/etc")
}

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	//WIP--->
	//
	var err error

	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}
	// rpc init
	rpcOps = &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	// Mount
	sylog.Debugf("initialize mount points\n")
	p := &mount.Points{}
	system := &mount.System{Points: p, Mount: engine.localMount}

	session, err = layout.NewSession(buildcfg.SESSIONDIR, "tmpfs", 4, system, overlay.New())
	if err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.LayerTag, engine.createEtc); err != nil {
		return err
	}

	if err := system.RunAfterTag(mount.LayerTag, engine.switchMount); err != nil {
		return err
	}

	if err := p.ImportFromSpec(engine.CommonConfig.OciConfig.Spec.Mounts); err != nil {
		return err
	}

	if err := engine.addRootfsMount(system); err != nil {
		return err
	}

	if err := system.MountAll(); err != nil {
		return err
	}

	//create config and state files for container monitoring
	uid := syscall.Geteuid()
	syscall.Setresuid(uid, 0, uid)

	sPath := fmt.Sprintf("%s/%s", StatePath, engine.CommonConfig.ContainerID)
	sylog.Debugf("writing state files into %s", sPath)
	if _, err := os.Stat(sPath); os.IsNotExist(err) {
		sylog.Debugf("%s doesn't exist...creating", sPath)
		if err := os.MkdirAll(sPath, 0755); err != nil {
			return err
		}
	}
	configPath := fmt.Sprintf("%s/config.json", sPath)
	statePath := fmt.Sprintf("%s/state.json", sPath)

	exportOptions := generate.ExportOptions{Seccomp: false}
	engine.CommonConfig.OciConfig.Generator.SaveToFile(configPath, exportOptions)

	state := &specs.State{
		Version: engine.CommonConfig.OciConfig.Spec.Version,
		ID:      engine.CommonConfig.ContainerID,
		Status:  "created",
		Pid:     pid,
		Bundle:  engine.EngineConfig.JSON.Image,
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ioutil.WriteFile(statePath, stateJSON, 0644)

	syscall.Setresuid(uid, uid, 0)

	sylog.Debugf("Chdir into %s\n", session.FinalPath())
	err = syscall.Chdir(session.FinalPath())
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Chroot into %s\n", session.FinalPath())
	_, err = rpcOps.Chroot(session.FinalPath())
	if err != nil {
		return fmt.Errorf("chroot failed: %s", err)
	}

	sylog.Debugf("Chdir into / to avoid errors\n")
	err = syscall.Chdir("/")
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	return nil
	//
	//<---
}
