package bridge

import (
	"bufio"
	"fmt"

	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CoreService struct {
	cmd       *exec.Cmd
	lock      sync.Mutex
	stopChan  chan struct{}
	logFile   *os.File
	pidPath   string
	isRunning bool
}

var coreService = &CoreService{}

func (s *CoreService) setRunning(running bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.isRunning = running
}

func (s *CoreService) getRunning() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.isRunning
}

func (a *App) StartCore(path string, args []string, options ExecOptions) FlagResult {
	log.Printf("StartCore: %s %s %v", path, args, options)
	return coreService.Start(a, path, args, options)
}

func (a *App) StopCore() FlagResult {
	log.Printf("StopCore")
	return coreService.Stop(a)
}

func (s *CoreService) Start(app *App, path string, args []string, options ExecOptions) FlagResult {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isRunning {
		return FlagResult{false, "Core is already running"}
	}

	exePath := GetPath(path)
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		exePath = path
	}

	// 1. Check & Kill Existing Process (Orphaned from previous session)
	if options.PidFile != "" {
		pidPath := GetPath(options.PidFile)
		if pidData, err := os.ReadFile(pidPath); err == nil {
			pidStr := strings.TrimSpace(string(pidData))
			if pid, err := strconv.Atoi(pidStr); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					// Check if process is actually running (signal 0 on unix / generic check)
					// On Windows FindProcess always succeeds. We need to try to kill it or check exit code.
					// A simple way is just try to kill it. If it fails, maybe it doesn't exist.
					log.Printf("Found existing orphaned core PID: %d. Killing it...", pid)
					proc.Kill()
					// Wait a bit?
					proc.Wait()
				}
			}
			os.Remove(pidPath)
		}
	}

	// 2. Kill by Name (Force clean other instances)
	// This ensures that even if PID file is missing, we don't start duplicate cores.
	binaryName := filepath.Base(exePath)
	killProcessByName(binaryName)

	cmd := exec.Command(exePath, args...)
	SetCmdWindowHidden(cmd)

	cmd.Env = os.Environ()
	for key, value := range options.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	// Prepare Log File
	if options.LogFile != "" {
		fullPath := GetPath(options.LogFile)
		if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err == nil {
			f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err == nil {
				s.logFile = f
			}
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return FlagResult{false, "Failed to get stdout pipe: " + err.Error()}
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return FlagResult{false, "Failed to start core: " + err.Error()}
	}

	s.cmd = cmd
	s.isRunning = true
	s.stopChan = make(chan struct{})

	// Handle PID File
	pid := strconv.Itoa(cmd.Process.Pid)
	if options.PidFile != "" {
		s.pidPath = GetPath(options.PidFile)
		if err := os.WriteFile(s.pidPath, []byte(pid), os.ModePerm); err != nil {
			// If we fail to write PID file, we should probably warn but proceed, or fail?
			// Proceeding is safer for the UI.
			log.Printf("Failed to write PID file: %v", err)
		}
	}

	// Handle Logs (Async)
	go s.handleLogs(app, stdout, options.StopOutputKeyword)

	// Monitor Process (Async)
	go s.monitorProcess(app, cmd)

	return FlagResult{true, pid}
}

// Cleanup forcibly kills the core process. Used for app shutdown.
func Cleanup() {
	if Config.ExitCoreOnShutdown {
		coreService.Shutdown()
	} else {
		log.Println("CoreService: Keeping core running as requested.")
	}
}

func (s *CoreService) Shutdown() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isRunning || s.cmd == nil || s.cmd.Process == nil {
		return
	}

	log.Println("CoreService: Cleanup on exit...")
	// Force kill on shutdown is safer to ensure no zombies
	// But valid signal is better if possible.
	// Since we are shutting down, time is limited.
	// Try Terminate first?
	// Just Kill to be sure.
	s.cmd.Process.Kill()
	s.isRunning = false
}

func (s *CoreService) Stop(app *App) FlagResult {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isRunning || s.cmd == nil || s.cmd.Process == nil {
		return FlagResult{true, "Core not running"}
	}

	// Kill logic
	proc := s.cmd.Process
	if err := SendExitSignal(proc); err != nil {
		log.Printf("Failed to send exit signal: %v", err)
		proc.Kill()
	}

	// We don't block heavily here, we expect monitorProcess to handle the cleanup and events.
	// But we should ensure we wait for it to actually die if we want synchronous "Stop" feel.
	err := waitForProcessExitWithTimeout(proc, 5)
	if err != nil {
		return FlagResult{false, err.Error()}
	}

	return FlagResult{true, "Stopped"}
}

func (s *CoreService) monitorProcess(app *App, cmd *exec.Cmd) {
	err := cmd.Wait()

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.cmd == cmd {
		s.isRunning = false
		s.cmd = nil
		if s.logFile != nil {
			s.logFile.Close()
			s.logFile = nil
		}
		if s.pidPath != "" {
			os.Remove(s.pidPath)
			s.pidPath = ""
		}

		exitMsg := "Core Stopped"
		if err != nil {
			exitMsg = fmt.Sprintf("Core Exited with error: %v", err)
		}
		log.Println(exitMsg)
		// Emit event to frontend
		runtime.EventsEmit(app.Ctx, "core-start-failed", exitMsg) // Reusing an event or use a new one?
		// Actually, frontend expects "endEvent".
		// "kernelApi.ts" currently passes an endEvent to ExecBackground.
		// We should emit a standard event "core-stopped".
		runtime.EventsEmit(app.Ctx, "core-stopped", exitMsg)
	}
}

// handleLogs buffers logs and emits them in batches to avoid flooding the bridge.
func (s *CoreService) handleLogs(app *App, reader io.Reader, stopKeyword string) {
	scanner := bufio.NewScanner(reader)
	linesChan := make(chan string, 1000)

	// Reader routine
	go func() {
		for scanner.Scan() {
			linesChan <- scanner.Text()
		}
		close(linesChan)
	}()

	buffer := make([]string, 0, 100)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Emitter loop
	for {
		select {
		case line, ok := <-linesChan:
			if !ok {
				// Flush remaining
				if len(buffer) > 0 {
					s.emitLogBatch(app, buffer)
				}
				return
			}

			// Process Line
			if s.logFile != nil {
				s.logFile.WriteString(line + "\n")
			}

			// Append to buffer
			buffer = append(buffer, line)

			// If buffer is full, emit immediately
			if len(buffer) >= 50 {
				s.emitLogBatch(app, buffer)
				buffer = buffer[:0]
			}

			if stopKeyword != "" && strings.Contains(line, stopKeyword) {
				runtime.EventsEmit(app.Ctx, "core-started")
			}

		case <-ticker.C:
			if len(buffer) > 0 {
				s.emitLogBatch(app, buffer)
				buffer = buffer[:0] // Clear buffer
			}
		}
	}
}

func (s *CoreService) emitLogBatch(app *App, logs []string) {
	// Join logs with newline for a single string event
	// Or emit as array if frontend supports it. String is safer for current "string" expectation logic?
	// kernelApi.ts: ExecBackground callback receives string 'out'.
	// If we send a big chunk joined by \n, frontend logic `out.includes(...)` still works.
	// But frontend `logsStore.recordKernelLog(out)` might expect line by line or handle chunk.
	// Let's check frontend.
	data := strings.Join(logs, "\n")
	runtime.EventsEmit(app.Ctx, "core-log", data)
}

func killProcessByName(name string) {
	log.Printf("Ensuring no other instances of %s are running...", name)
	switch Env.OS {
	case "windows":
		// /F = Force, /IM = Image Name
		exec.Command("taskkill", "/F", "/IM", name).Run()
	case "darwin", "linux":
		// Unix-like systems
		// Use SIGKILL (-9) for force kill
		// Try pkill first (Process Kill by name pattern, -x for exact)
		if err := exec.Command("pkill", "-9", "-x", name).Run(); err != nil {
			// If pkill fails (e.g. command not found or no process), try killall as fallback
			exec.Command("killall", "-9", name).Run()
		}
	default:
		// Fallback for others (e.g. FreeBSD if supported later), try pkill
		exec.Command("pkill", "-9", "-x", name).Run()
	}
}
