//go:build windows

package updatebusy

import (
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// DefaultList returns running processes with name, exe path, and command line.
func DefaultList() ([]Proc, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	out := make([]Proc, 0, len(procs))
	for _, p := range procs {
		var item Proc
		if name, err := p.Name(); err == nil {
			item.Name = name
		}
		if exe, err := p.Exe(); err == nil {
			item.ExePath = exe
		}
		if cmd, err := p.Cmdline(); err == nil {
			item.CmdLine = cmd
		}
		if item.Name == "" && item.ExePath == "" && item.CmdLine == "" {
			continue
		}
		// Skip our own capture noise reduction: empty names already dropped.
		if strings.TrimSpace(item.Name) == "" && strings.TrimSpace(item.ExePath) == "" {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

// NewDetector returns a detector wired to the live process list.
func NewDetector() *Detector {
	return &Detector{List: DefaultList}
}
