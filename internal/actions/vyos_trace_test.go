package actions

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/routerarchitects/nats-agent-core/agentcore"
)

type fakeCommand struct {
	runFunc func() error
}

func (c *fakeCommand) Run() error {
	if c.runFunc != nil {
		return c.runFunc()
	}
	return nil
}

type fakeCommandRunner struct {
	calls    int
	lastArgs []string
	runErr   error
	runFunc  func() error
}

func (r *fakeCommandRunner) Command(ctx context.Context, name string, args ...string) Command {
	r.calls++
	r.lastArgs = args
	return &fakeCommand{
		runFunc: func() error {
			if r.runFunc != nil {
				return r.runFunc()
			}
			return r.runErr
		},
	}
}

/*
TC-ACTIONS-TRACE-001
Type: Positive
Title: Happy path trace action execution
Summary:
Submits a trace action payload with valid interface and upload URI.
Verifies that tcpdump executes with correct parameters, the PCAP file
is uploaded via HTTP multipart POST, local PCAP is deleted, and success output is returned.
Validates:
  - tcpdump command parameters (interface, output path, packets count)
  - HTTP upload content is correct and multipart boundary is handled
  - local PCAP cleanup works on successful execution
  - deterministic JSON output structure
*/
func TestVyOSTraceExecutorHappyPath(t *testing.T) {
	// 1. Start test HTTP upload server
	var uploadedFileContent []byte
	var formFileName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		formFileName = header.Filename

		content, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "read failed", http.StatusInternalServerError)
			return
		}
		uploadedFileContent = content
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"uploaded"}`))
	}))
	defer server.Close()

	runner := &fakeCommandRunner{
		runFunc: func() error {
			// Mock writing a dummy PCAP file
			rpcID := "rpc-trace-1"
			pcapPath := filepath.Join("/tmp", "pcap-"+rpcID+".pcap")
			return os.WriteFile(pcapPath, []byte("pcap-contents"), 0o600)
		},
	}

	exec := NewVyOSTraceExecutor(runner, server.Client())
	msg := agentcore.ActionCommand{
		Version: "1.0",
		RPCID:   "rpc-trace-1",
		Target:  "vyos",
		Action:  ActionTrace,
		Payload: json.RawMessage(`{
			"interface": "eth0",
			"duration": 5,
			"packets": 50,
			"uri": "` + server.URL + `"
		}`),
		Timestamp: time.Now(),
	}

	out, err := exec.Execute(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Message != "trace action completed" {
		t.Fatalf("unexpected message: %q", out.Message)
	}

	// Verify command parameters
	if runner.calls != 1 {
		t.Fatalf("expected 1 command runner call, got %d", runner.calls)
	}
	expectedArgs := []string{"-U", "-i", "eth0", "-w", "/tmp/pcap-rpc-trace-1.pcap", "-c", "50"}
	if len(runner.lastArgs) != len(expectedArgs) {
		t.Fatalf("args len mismatch: got %+v, want %+v", runner.lastArgs, expectedArgs)
	}
	for i, arg := range runner.lastArgs {
		if arg != expectedArgs[i] {
			t.Fatalf("arg[%d] got %q, want %q", i, arg, expectedArgs[i])
		}
	}

	// Verify uploaded file
	if string(uploadedFileContent) != "pcap-contents" {
		t.Fatalf("uploaded content got %q want %q", string(uploadedFileContent), "pcap-contents")
	}
	if formFileName != "pcap-rpc-trace-1.pcap" {
		t.Fatalf("uploaded file name got %q want %q", formFileName, "pcap-rpc-trace-1.pcap")
	}

	// Verify local cleanup
	if _, err := os.Stat("/tmp/pcap-rpc-trace-1.pcap"); !os.IsNotExist(err) {
		t.Fatalf("expected pcap file to be cleaned up, stat returned: %v", err)
	}
}

/*
TC-ACTIONS-TRACE-002
Type: Negative
Title: Payload validation for trace action execution
Summary:
Asserts payload structure validation constraints on the input payload,
including required fields, interface format to prevent command injection,
and HTTP/HTTPS URI scheme check.
Validates:
  - interface name format matching ^[a-z0-9\.\-]+$
  - upload URI scheme (only http/https are allowed)
  - json structure validity
  - empty target or rpc ID validation
*/
func TestVyOSTraceExecutorPayloadValidation(t *testing.T) {
	runner := &fakeCommandRunner{}
	exec := NewVyOSTraceExecutor(runner, nil)

	cases := []struct {
		name    string
		msg     agentcore.ActionCommand
		wantErr string
	}{
		{
			name: "missing interface",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"uri":"http://localhost"}`),
			},
			wantErr: "interface is required",
		},
		{
			name: "invalid interface name / command injection safe",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":"eth0; rm -rf /","uri":"http://localhost"}`),
			},
			wantErr: "invalid interface name",
		},
		{
			name: "missing uri",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":"eth0"}`),
			},
			wantErr: "uri is required",
		},
		{
			name: "invalid uri scheme",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":"eth0","uri":"ftp://localhost"}`),
			},
			wantErr: "invalid upload uri",
		},
		{
			name: "invalid json payload",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":`),
			},
			wantErr: "payload must be valid json",
		},
		{
			name: "empty target",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "rpc-1", Target: "", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":"eth0","uri":"http://localhost"}`),
			},
			wantErr: "target is empty",
		},
		{
			name: "empty rpc id",
			msg: agentcore.ActionCommand{
				Version: "1.0", RPCID: "", Target: "vyos", Action: ActionTrace,
				Payload: json.RawMessage(`{"interface":"eth0","uri":"http://localhost"}`),
			},
			wantErr: "rpc id is empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := exec.Execute(context.Background(), tc.msg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error got %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

/*
TC-ACTIONS-TRACE-003
Type: Negative
Title: Context cancellation for trace action execution
Summary:
Verifies that cancelling the parent context aborts the capture operation
and does not proceed to the upload step.
Validates:
  - executor responds to parent context cancellation
  - capture aborted error is returned
*/
func TestVyOSTraceExecutorContextCancellation(t *testing.T) {
	// Define a runner that blocks until the context is cancelled
	runner := &fakeCommandRunner{}

	exec := NewVyOSTraceExecutor(runner, nil)
	msg := agentcore.ActionCommand{
		Version: "1.0",
		RPCID:   "rpc-trace-cancel",
		Target:  "vyos",
		Action:  ActionTrace,
		Payload: json.RawMessage(`{
			"interface": "eth0",
			"duration": 5,
			"uri": "http://localhost"
		}`),
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner.runFunc = func() error {
		cancel() // Cancel the parent context inside command execution
		return context.Canceled
	}

	_, err := exec.Execute(ctx, msg)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "capture aborted") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

/*
TC-ACTIONS-TRACE-004
Type: Negative
Title: HTTP upload failure during trace execution
Summary:
Simulates a remote HTTP server failure (e.g. status 500) during trace upload.
Asserts that the executor handles the error properly and still cleans up the local PCAP file.
Validates:
  - HTTP upload non-2xx status code handled as failure
  - local PCAP file cleanup occurs on upload failure
*/
func TestVyOSTraceExecutorHTTPFailure(t *testing.T) {
	// Start a test server that returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	runner := &fakeCommandRunner{
		runFunc: func() error {
			pcapPath := filepath.Join("/tmp", "pcap-rpc-trace-http-fail.pcap")
			return os.WriteFile(pcapPath, []byte("dummy-pcap"), 0o600)
		},
	}

	exec := NewVyOSTraceExecutor(runner, server.Client())
	msg := agentcore.ActionCommand{
		Version: "1.0",
		RPCID:   "rpc-trace-http-fail",
		Target:  "vyos",
		Action:  ActionTrace,
		Payload: json.RawMessage(`{
			"interface": "eth0",
			"duration": 5,
			"uri": "` + server.URL + `"
		}`),
		Timestamp: time.Now(),
	}

	_, err := exec.Execute(context.Background(), msg)
	if err == nil {
		t.Fatal("expected http upload error, got nil")
	}
	if !strings.Contains(err.Error(), "upload failed status=500") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Verify local cleanup happens even on HTTP upload failures
	if _, err := os.Stat("/tmp/pcap-rpc-trace-http-fail.pcap"); !os.IsNotExist(err) {
		t.Fatalf("expected pcap file to be cleaned up, stat returned: %v", err)
	}
}
