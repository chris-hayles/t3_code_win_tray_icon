//go:build windows

package supervisor

import (
	"fmt"
	"os"
	"sync"
	"time"

	"t3tray/internal/backoff"
	"t3tray/internal/cmdquote"
	"t3tray/internal/logroll"
	"t3tray/internal/ownership"
	"t3tray/internal/paths"
	"t3tray/internal/winutil"
)

type Status int

const (
	StatusStopped Status = iota
	StatusWaiting
	StatusRunning
	StatusCLIMissing
	StatusForeignOwner
)

type Supervisor struct {
	mu sync.Mutex

	job            *winutil.Job
	rootPID        int
	attempt        int
	startedAt      time.Time
	stopCh         chan struct{}
	stopped        bool
	logWriter      *os.File
	supLog         *os.File
	foreignAtStart bool
	loopRunning    bool
}

func New() (*Supervisor, error) {
	if err := logroll.EnsureDir(paths.TrayDataDir()); err != nil {
		return nil, err
	}
	sl, err := logroll.OpenAppend(paths.SupervisorLog())
	if err != nil {
		return nil, err
	}
	return &Supervisor{
		stopCh: make(chan struct{}),
		supLog: sl,
	}, nil
}

func (s *Supervisor) CloseLogs() {
	if s.supLog != nil {
		_ = s.supLog.Close()
	}
}

func (s *Supervisor) writeSup(msg string) {
	if s.supLog != nil {
		_, _ = fmt.Fprintf(s.supLog, "%s %s\n", time.Now().Format(time.RFC3339), msg)
	}
}

func (s *Supervisor) IsOwnedPID(pid int) bool {
	s.mu.Lock()
	job := s.job
	s.mu.Unlock()
	if job == nil {
		return false
	}
	return winutil.IsPIDInJob(pid, job)
}

func (s *Supervisor) CheckStartupOwnership() ownership.State {
	st, _, err := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, s.IsOwnedPID)
	if err != nil {
		return ownership.Absent
	}
	if st == ownership.Foreign {
		s.setForeign(true)
		s.writeSup("foreign server at startup")
	}
	return st
}

func (s *Supervisor) StartLoop() {
	s.mu.Lock()
	if s.loopRunning || s.foreignAtStart || s.stopped {
		s.mu.Unlock()
		return
	}
	s.loopRunning = true
	s.mu.Unlock()
	go s.loop()
}

func (s *Supervisor) loop() {
	for {
		if !s.sleepOrStop(0) {
			return
		}
		if _, err := os.Stat(paths.T3Cmd()); err != nil {
			s.writeSup("t3.cmd missing")
			if !s.sleepOrStop(2 * time.Second) {
				return
			}
			continue
		}
		s.mu.Lock()
		s.attempt++
		attempt := s.attempt
		s.mu.Unlock()

		if s.startOnce() {
			s.waitRun()
			s.mu.Lock()
			if backoff.ShouldResetUptime(time.Since(s.startedAt)) {
				s.attempt = backoff.NextAttemptAfterReset() - 1
			}
			s.mu.Unlock()
			st, _, _ := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, s.IsOwnedPID)
			if st == ownership.Foreign {
				s.setForeign(true)
				s.writeSup("detected foreign owner after exit")
				return
			}
		} else {
			delay := backoff.Delay(attempt)
			s.writeSup(fmt.Sprintf("start failed, backoff %v", delay))
			if !s.sleepOrStop(delay) {
				return
			}
		}
	}
}

func (s *Supervisor) sleepOrStop(d time.Duration) bool {
	if d <= 0 {
		select {
		case <-s.stopCh:
			return false
		default:
			return true
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-s.stopCh:
		return false
	case <-t.C:
		return true
	}
}

func (s *Supervisor) startOnce() bool {
	if s.isForeign() {
		return false
	}
	st, _, _ := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, s.IsOwnedPID)
	if st == ownership.Foreign {
		return false
	}
	job, err := winutil.NewKillOnCloseJob()
	if err != nil {
		s.writeSup(fmt.Sprintf("job create: %v", err))
		return false
	}
	serveLog, err := logroll.OpenAppend(paths.ServeLog())
	if err != nil {
		_ = job.Close()
		return false
	}
	cmdArgs := cmdquote.WindowsCmdQuoted(paths.T3Cmd(), paths.ServeArgs)
	pid, proc, err := winutil.StartHiddenCmdWithJob(cmdArgs, paths.UserProfile(), job, serveLog, serveLog)
	if err != nil {
		_ = serveLog.Close()
		_ = job.Close()
		s.writeSup(fmt.Sprintf("start serve: %v", err))
		return false
	}
	winutil.CloseProcessHandle(uintptr(proc))
	s.mu.Lock()
	if s.job != nil {
		_ = s.job.TerminateTree()
	}
	s.job = job
	s.rootPID = pid
	s.startedAt = time.Now()
	if s.logWriter != nil {
		_ = s.logWriter.Close()
	}
	s.logWriter = serveLog
	s.mu.Unlock()
	s.writeSup(fmt.Sprintf("started serve wrapper pid %d", pid))
	return true
}

func (s *Supervisor) waitRun() {
	for {
		s.mu.Lock()
		root := s.rootPID
		started := s.startedAt
		s.mu.Unlock()
		if root == 0 || !winutil.ProcessAlive(root) {
			s.mu.Lock()
			s.cleanupJobLocked()
			s.mu.Unlock()
			return
		}
		if time.Since(started) >= 60*time.Second {
			s.mu.Lock()
			s.attempt = 0
			s.mu.Unlock()
		}
		if !s.sleepOrStop(time.Second) {
			return
		}
	}
}

func (s *Supervisor) cleanupJobLocked() {
	if s.logWriter != nil {
		_ = s.logWriter.Close()
		s.logWriter = nil
	}
	if s.job != nil {
		_ = s.job.TerminateTree()
		s.job = nil
	}
	s.rootPID = 0
}

func (s *Supervisor) StopOwned() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()
	close(s.stopCh)
	s.mu.Lock()
	s.cleanupJobLocked()
	s.mu.Unlock()
}

func (s *Supervisor) RestartOwned() bool {
	st, _, _ := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, s.IsOwnedPID)
	s.mu.Lock()
	owned := st == ownership.Owned || (s.rootPID != 0 && winutil.ProcessAlive(s.rootPID))
	s.mu.Unlock()
	if !owned {
		return false
	}
	s.mu.Lock()
	s.cleanupJobLocked()
	s.attempt = 0
	s.mu.Unlock()
	return true
}

func (s *Supervisor) CurrentStatus() (Status, string) {
	if s.isForeign() {
		return StatusForeignOwner, "Another server owns this home"
	}
	if _, err := os.Stat(paths.T3Cmd()); err != nil {
		return StatusCLIMissing, "Open T3 Code once to install the t3 command"
	}
	rt, err := ownership.ReadRuntime(paths.ServerRuntimeJSON())
	if err == nil && rt.PID != nil && winutil.ProcessAlive(*rt.PID) {
		if s.IsOwnedPID(*rt.PID) {
			host := rt.Host
			if host == "" {
				host = "127.0.0.1"
			}
			return StatusRunning, fmt.Sprintf("Running on %s:%d", host, rt.Port)
		}
		return StatusForeignOwner, "Another server owns this home"
	}
	s.mu.Lock()
	root := s.rootPID
	s.mu.Unlock()
	if root != 0 && winutil.ProcessAlive(root) {
		return StatusWaiting, "Waiting"
	}
	if s.loopRunning && !s.stopped {
		return StatusWaiting, "Waiting"
	}
	return StatusStopped, "Stopped"
}

func (s *Supervisor) EnsureOwnedOrStart() {
	st, _, _ := ownership.ClassifyFile(paths.ServerRuntimeJSON(), winutil.ProcessAlive, s.IsOwnedPID)
	if st != ownership.Foreign {
		// The desktop server that blocked startup can be gone after the attach flow closes it.
		s.setForeign(false)
	}
	if st == ownership.Owned || st == ownership.Foreign {
		return
	}
	s.StartLoop()
}

func (s *Supervisor) setForeign(v bool) {
	s.mu.Lock()
	s.foreignAtStart = v
	s.mu.Unlock()
}

func (s *Supervisor) isForeign() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.foreignAtStart
}

func (s *Supervisor) WaitForLoopback(timeout time.Duration) (*ownership.Runtime, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		rt, err := ownership.ReadRuntime(paths.ServerRuntimeJSON())
		if err == nil && rt.PID != nil && winutil.ProcessAlive(*rt.PID) && rt.Host == "127.0.0.1" && s.IsOwnedPID(*rt.PID) {
			return rt, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for server on 127.0.0.1")
}

func (s *Supervisor) ForeignBlocked() bool {
	return s.foreignAtStart
}
