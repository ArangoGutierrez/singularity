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

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs/mount"
	"github.com/singularityware/singularity/src/pkg/util/loop"
	"github.com/singularityware/singularity/src/runtime/engines/singularity/rpc/client"
	"github.com/sylabs/sif/pkg/sif"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// CreateContainer creates a container
func (engine *EngineOperations) CreateContainer(pid int, rpcConn net.Conn) error {
	//WIP--->
	//
	if engine.CommonConfig.EngineName != Name {
		return fmt.Errorf("engineName configuration doesn't match runtime name")
	}
	// rpc init
	rpcOps := &client.RPC{
		Client: rpc.NewClient(rpcConn),
		Name:   engine.CommonConfig.EngineName,
	}
	if rpcOps.Client == nil {
		return fmt.Errorf("failed to initialiaze RPC client")
	}

	// Mount
	sylog.Debugf("initialize mount points\n")
	p := &mount.Points{}
	if err := p.ImportFromSpec(engine.CommonConfig.OciConfig.Spec.Mounts); err != nil {
		return err
	}
	if err := engine.addRootfs(p); err != nil {
		return err
	}

	if err := mountAll(rpcOps, p); err != nil {
		return err
	}

	// create config and state files for container monitoring
	uid := syscall.Geteuid()
	syscall.Setresuid(uid, 0, uid)

	sPath := fmt.Sprintf("%s/%s", StatePath, engine.CommonConfig.ContainerID)
	sylog.Debugf("writing state files into %s", sPath)
	if _, err := os.Stat(sPath); os.IsNotExist(err) {
		sylog.Debugf("%s doesn't exist...creating", sPath)
		if err := os.Mkdir(sPath, 0644); err != nil {
			return err
		}
	}
	configPath := fmt.Sprintf("%s/config.json", sPath)
	statePath := fmt.Sprintf("%s/state.json", sPath)

	specJSON, err := json.Marshal(engine.CommonConfig.OciConfig.Spec)
	if err != nil {
		return err
	}
	ioutil.WriteFile(configPath, specJSON, 0644)

	state := &specs.State{
		Version: engine.CommonConfig.OciConfig.Spec.Version,
		ID:      engine.CommonConfig.ContainerID,
		Status:  "created",
		Pid:     pid,
		Bundle:  engine.EngineConfig.Image,
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	ioutil.WriteFile(statePath, stateJSON, 0644)

	syscall.Setresuid(uid, uid, 0)

	sylog.Debugf("Chdir into %s\n", buildcfg.SESSIONDIR)
	err = syscall.Chdir(buildcfg.SESSIONDIR)
	if err != nil {
		return fmt.Errorf("change directory failed: %s", err)
	}

	sylog.Debugf("Chroot into %s\n", buildcfg.SESSIONDIR)
	_, err = rpcOps.Chroot(buildcfg.SESSIONDIR)
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

func (engine *EngineOperations) addRootfs(p *mount.Points) error {
	var flags uintptr = syscall.MS_NOSUID | syscall.MS_RDONLY | syscall.MS_NODEV
	rootfs := engine.EngineConfig.Image

	imageObject, err := image.Init(rootfs, false)
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
			mountType = "squashfs"
		} else if fstype == sif.FsExt3 {
			mountType = "ext3"
		} else {
			return fmt.Errorf("unknown file system type: %v", fstype)
		}

		imageObject.Offset = uint64(part.Fileoff)
		imageObject.Size = uint64(part.Filelen)
	case image.SQUASHFS:
		mountType = "squashfs"
	case image.EXT3:
		mountType = "ext3"
	case image.SANDBOX:
		sylog.Debugf("Mounting directory rootfs: %v\n", rootfs)
		return p.AddBind(mount.RootfsTag, rootfs, buildcfg.CONTAINER_FINALDIR, syscall.MS_BIND|flags)
	}

	sylog.Debugf("Mounting block [%v] image: %v\n", mountType, rootfs)
	return p.AddImage(mount.RootfsTag, rootfs, buildcfg.CONTAINER_FINALDIR, mountType, flags, imageObject.Offset, imageObject.Size)
}

func mountAll(rpcOps *client.RPC, p *mount.Points) error {
	for _, tag := range mount.GetTagList() {
		for _, point := range p.GetByTag(tag) {
			if _, err := mount.GetOffset(point.InternalOptions); err == nil {
				if err := mountImage(rpcOps, &point); err != nil {
					return err
				}
			} else {
				if err := mountGeneric(rpcOps, &point); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// mount any generic mount (not loop dev)
func mountGeneric(rpcOps *client.RPC, mnt *mount.Point) error {
	flags, opts := mount.ConvertOptions(mnt.Options)
	optsString := strings.Join(opts, ",")

	sylog.Debugf("Mounting %s to %s\n", mnt.Source, mnt.Destination)
	_, err := rpcOps.Mount(mnt.Source, mnt.Destination, mnt.Type, flags, optsString)
	return err
}

// mount image via loop
func mountImage(rpcOps *client.RPC, mnt *mount.Point) error {
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

	info := &loop.Info64{
		Offset:    offset,
		SizeLimit: sizelimit,
		Flags:     loop.FlagsAutoClear,
	}

	sylog.Debugf("Mounting %v to loop device from %v - %v\n", mnt.Source, offset, sizelimit)
	number, err := rpcOps.LoopDevice(mnt.Source, os.O_RDONLY, *info)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/dev/loop%d", number)
	sylog.Debugf("Mounting loop device %s to %s\n", path, mnt.Destination)
	_, err = rpcOps.Mount(path, mnt.Destination, mnt.Type, flags, optsString)
	if err != nil {
		return fmt.Errorf("failed to mount %s filesystem: %s", mnt.Type, err)
	}

	return nil
}
