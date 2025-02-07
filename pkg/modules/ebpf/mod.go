/*===========================================================================*\
 *           MIT License Copyright (c) 2022 Kris Nóva <kris@nivenly.com>     *
 *                                                                           *
 *                ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓                *
 *                ┃   ███╗   ██╗ ██████╗ ██╗   ██╗ █████╗   ┃                *
 *                ┃   ████╗  ██║██╔═████╗██║   ██║██╔══██╗  ┃                *
 *                ┃   ██╔██╗ ██║██║██╔██║██║   ██║███████║  ┃                *
 *                ┃   ██║╚██╗██║████╔╝██║╚██╗ ██╔╝██╔══██║  ┃                *
 *                ┃   ██║ ╚████║╚██████╔╝ ╚████╔╝ ██║  ██║  ┃                *
 *                ┃   ╚═╝  ╚═══╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝  ┃                *
 *                ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛                *
 *                                                                           *
 *                       This machine kills fascists.                        *
 *                                                                           *
\*===========================================================================*/

package modebpf

import (
	"fmt"
	"path/filepath"

	modproc "github.com/kris-nova/xpid/pkg/modules/proc"

	api "github.com/kris-nova/xpid/pkg/api/v1"
	module "github.com/kris-nova/xpid/pkg/modules"
	"github.com/kris-nova/xpid/pkg/procx"
)

var _ procx.ProcessExplorerModule = &EBPFModule{}

const (
	EBPFFullMount  string = "bpf /sys/fs/bpf bpf"
	EBPFSYSFSMount string = "/sys/fs/bpf"
)

type EBPFModule struct {
}

func NewEBPFModule() *EBPFModule {
	return &EBPFModule{}
}

type EBPFModuleResult struct {
	pid    *api.Process
	Mounts string
}

func (m *EBPFModule) Meta() *module.Meta {
	return &module.Meta{
		Name:        "eBPF module",
		Description: "Search proc(5) filesystems for eBPF programs. Will do an in depth scan and search for obfuscated directories.",
		Authors: []string{
			"Kris Nóva <kris@nivenly.com>",
		},
	}
}

func (m *EBPFModule) Execute(p *api.Process) (procx.ProcessExplorerResult, error) {
	// Module specific (correlated)
	result := &EBPFModuleResult{}

	procfs := modproc.NewProcFileSystem(modproc.Proc())
	mounts, _ := procfs.ContentsPID(p.PID, "mounts")
	result.Mounts = mounts

	e, err := NewEBPFFileSystemData()
	if err != nil {
		return nil, fmt.Errorf("unable to read /sys/fs/bpf: %v", err)
	}

	// Compare with file descriptors in /proc
	fds, err := procfs.DirPID(p.PID, "fdinfo")

	if err != nil {
		return nil, fmt.Errorf("unable to read /proc/%d/fdinfo: %v", p.PID, err)
	}
	for _, fd := range fds {
		fddata, err := procfs.ContentsPID(p.PID, filepath.Join("fdinfo", fd.Name()))
		if err != nil {
			continue
		}
		ebpfProgID := modproc.FileKeyValue(fddata, "prog_id")
		if ebpfProgID == "" {
			continue
		}
		for id, mp := range e.Progs {
			if id == "" {
				continue
			}
			if id == ebpfProgID {
				// We have mapped an eBPF program to a PID!
				p.EBPF = true
				p.EBPFMeta.Progs = append(p.EBPFMeta.Progs, mp.Name)
			}
		}
	}

	return result, nil
}
