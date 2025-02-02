package jsvserver

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type JSVTestServer struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Reader
	stderr       *bufio.Reader
	mu           sync.Mutex
	timeout      time.Duration
	envRequested bool
}

// NewJSVTestServer creates a new JSVTestServer instance.
//
// The first argument is the path to the JSV script.
func NewJSVTestServer(jsvPath string) (*JSVTestServer, error) {
	cmd := exec.Command(jsvPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	return &JSVTestServer{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		stderr:  bufio.NewReader(stderr),
		timeout: 5 * time.Second,
	}, nil
}

func (s *JSVTestServer) Start() error {
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start JSV process: %w", err)
	}

	go s.monitorStderr()

	//ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	//defer cancel()

	// Send initial START command
	if err := s.sendCommand("START"); err != nil {
		return err
	}

	// Handle JSV initialization sequence
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return fmt.Errorf("protocol error: %w", err)
		}
		line = strings.TrimSpace(line)

		switch {
		case line == "STARTED":
			return nil
		case strings.HasPrefix(line, "SEND"):
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 && parts[1] == "ENV" {
				s.envRequested = true
			}
		default:
			return fmt.Errorf("unexpected response during startup: %s", line)
		}
	}
}

func (s *JSVTestServer) SendJob(job *JobSpec) (*JSVResult, error) {
	//ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	//defer cancel()

	// Send pseudo-parameters first
	if err := s.sendCommand("PARAM VERSION 1.0"); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM CONTEXT %s", job.Context)); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM CLIENT %s", job.Client)); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM USER %s", job.User)); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM GROUP %s", job.Group)); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM CMDNAME %s", job.CmdName)); err != nil {
		return nil, err
	}
	if err := s.sendCommand(fmt.Sprintf("PARAM CMDARGS %d", job.CmdArgs)); err != nil {
		return nil, err
	}

	// Send other parameters
	for param, value := range job.Params {
		if err := s.sendCommand(fmt.Sprintf("PARAM %s %s", param, value)); err != nil {
			return nil, err
		}
	}

	// Send environment if requested
	if s.envRequested {
		for env, value := range job.Environment {
			if err := s.sendCommand(fmt.Sprintf("ENV ADD %s %s", env, value)); err != nil {
				return nil, err
			}
		}
	}

	// Begin verification
	if err := s.sendCommand("BEGIN"); err != nil {
		return nil, err
	}

	// Process JSV response
	result := &JSVResult{}
	for {
		line, err := s.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read error: %w", err)
		}
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "RESULT"):
			parts := strings.SplitN(line, " ", 4)
			if len(parts) < 3 {
				return nil, fmt.Errorf("invalid RESULT format: %s", line)
			}
			result.State = parts[2]
			if len(parts) > 3 {
				result.Message = strings.Join(parts[3:], " ")
			}
			log.Printf("JSV Result: %+v", result)
			return result, nil

		case strings.HasPrefix(line, "PARAM"):
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 2 {
				return nil, fmt.Errorf("invalid PARAM format: %s", line)
			}
			value := ""
			if len(parts) > 2 {
				value = parts[2]
			}
			if result.ModifiedParams == nil {
				result.ModifiedParams = make(map[string]string)
			}
			result.ModifiedParams[parts[1]] = value

		case strings.HasPrefix(line, "ENV"):
			parts := strings.SplitN(line, " ", 4)
			if len(parts) < 4 {
				return nil, fmt.Errorf("invalid ENV format: %s", line)
			}
			switch parts[1] {
			case "ADD", "MOD":
				if result.ModifiedEnv == nil {
					result.ModifiedEnv = make(map[string]string)
				}
				result.ModifiedEnv[parts[2]] = parts[3]
			case "DEL":
				delete(result.ModifiedEnv, parts[2])
			}

		case strings.HasPrefix(line, "LOG"):
			log.Printf("JSV LOG: %s", line)

		default:
			log.Printf("Unexpected JSV response: %s", line)
		}
	}
}

func (s *JSVTestServer) sendCommand(cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := fmt.Fprintf(s.stdin, "%s\n", cmd)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	return nil
}

func (s *JSVTestServer) monitorStderr() {
	for {
		line, err := s.stderr.ReadString('\n')
		if err != nil {
			return
		}
		log.Printf("JSV STDERR: %s", strings.TrimSpace(line))
	}
}

func (s *JSVTestServer) Stop() error {
	if err := s.sendCommand("QUIT"); err != nil {
		return err
	}

	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("JSV process exit error: %w", err)
	}
	return nil
}

// JobSpec is a specifiction of a batch job which is processed by
// the JSV script. See the qsub(1) man page for more information
// about the parameters.
type JobSpec struct {
	Context     string            `json:"context"`
	Client      string            `json:"client"`
	User        string            `json:"user"`
	Group       string            `json:"group"`
	CmdName     string            `json:"cmd_name"`
	CmdArgs     int               `json:"cmd_args"`
	Params      map[string]string `json:"params"`
	Environment map[string]string `json:"environment"`
}

type JSVResult struct {
	State          string
	Message        string
	ModifiedParams map[string]string
	ModifiedEnv    map[string]string
}
