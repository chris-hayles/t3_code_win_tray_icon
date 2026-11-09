package ownership

import (
	"encoding/json"
	"errors"
	"os"
)

type State int

const (
	Absent State = iota
	Owned
	Foreign
)

type Runtime struct {
	PID    *int   `json:"pid"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Origin string `json:"origin"`
}

type PIDLive func(pid int) bool
type PIDOwned func(pid int) bool

func ReadRuntime(path string) (*Runtime, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rt Runtime
	if err := json.Unmarshal(data, &rt); err != nil {
		return nil, err
	}
	return &rt, nil
}

func Classify(rt *Runtime, isLive PIDLive, isOwned PIDOwned) State {
	if rt == nil || rt.PID == nil {
		return Absent
	}
	pid := *rt.PID
	if !isLive(pid) {
		return Absent
	}
	if isOwned(pid) {
		return Owned
	}
	return Foreign
}

func ClassifyFile(path string, isLive PIDLive, isOwned PIDOwned) (State, *Runtime, error) {
	rt, err := ReadRuntime(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Absent, nil, nil
		}
		return Absent, nil, err
	}
	return Classify(rt, isLive, isOwned), rt, nil
}
