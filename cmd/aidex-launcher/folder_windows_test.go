package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNativeConcurrentAuthorizationLog(t *testing.T) {
	// Real separate processes, exact production append flags; only a temp log.
	if path := os.Getenv("AIDEX_SHARING_TEST_LOG"); path != "" {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		logger := authorizationLogger(file, "worker")
		for index := 0; index < 100; index++ {
			logger.Info("append test", "record", index)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "folder-authorization.log")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	worker := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeConcurrentAuthorizationLog$")
	var workerOutput bytes.Buffer
	worker.Stdout = &workerOutput
	worker.Stderr = &workerOutput
	worker.WaitDelay = 3 * time.Second
	worker.Env = append(os.Environ(), "AIDEX_SHARING_TEST_LOG="+path)
	worker.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err = worker.Start(); err != nil {
		file.Close()
		t.Fatal(err)
	}
	logger := authorizationLogger(file, "supervisor")
	for index := 0; index < 100; index++ {
		logger.Info("append test", "record", index)
	}
	if err = worker.Wait(); err != nil {
		file.Close()
		t.Fatal(err, workerOutput.String())
	}
	file.Close()
	input, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	scanner := bufio.NewScanner(input)
	roles := map[string]int{}
	for scanner.Scan() {
		var record struct {
			Role string `json:"role"`
			PID  int    `json:"pid"`
		}
		if err = json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatal("corrupt interprocess JSON", err)
		}
		if record.PID <= 0 {
			t.Fatal("missing PID")
		}
		roles[record.Role]++
	}
	if err = scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if roles["worker"] != 100 || roles["supervisor"] != 100 {
		t.Fatal("lost log records", roles)
	}
}
