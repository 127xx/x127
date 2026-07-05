// Package ports enumerates listening TCP ports on the host.
// On macOS, PID/process info for other users' processes may be
// unavailable without root; such entries keep an empty Process.
package ports

import (
	"fmt"
	"sort"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Entry struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Address string `json:"address"`
	PID     int32  `json:"pid"`
	Process string `json:"process"`
}

func Scan() ([]Entry, error) {
	conns, err := gnet.Connections("tcp")
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var out []Entry
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		key := fmt.Sprintf("%s|%d|%d", c.Laddr.IP, c.Laddr.Port, c.Pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		e := Entry{
			Port:    int(c.Laddr.Port),
			Proto:   "tcp",
			Address: c.Laddr.IP,
			PID:     c.Pid,
		}
		if c.Pid > 0 {
			if p, err := process.NewProcess(c.Pid); err == nil {
				if name, err := p.Name(); err == nil {
					e.Process = name
				}
			}
		}
		out = append(out, e)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].Address < out[j].Address
	})
	return out, nil
}
