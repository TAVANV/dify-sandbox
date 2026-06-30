package python

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/langgenius/dify-sandbox/internal/core/runner"
	"github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/static"
)

type PythonRunner struct {
	runner.TempDirRunner
}

//go:embed prescript.py
var sandbox_fs []byte

func (p *PythonRunner) Run(
	ctx context.Context,
	code string,
	timeout time.Duration,
	stdin []byte,
	preload string,
	options *types.RunnerOptions,
) (*runner.OutputCaptureResult, error) {
	configuration := static.GetDifySandboxGlobalConfigurations()

	uid, err := AcquireUID(ctx)
	if err != nil {
		return nil, fmt.Errorf("no available sandbox UID: %w", err)
	}

	bootstrapPath, err := p.InitializeEnvironment(preload, options, uid)
	if err != nil {
		ReleaseUID(uid)
		return nil, err
	}

	codeReader, codeWriter, err := os.Pipe()
	if err != nil {
		os.Remove(bootstrapPath)
		ReleaseUID(uid)
		return nil, err
	}

	outputHandler := runner.NewOutputCaptureRunner()
	outputHandler.SetTimeout(timeout)
	outputHandler.SetAfterExitHook(func() {
		codeReader.Close()
		codeWriter.Close()
		os.Remove(bootstrapPath)
		cleanupTempFiles(uid)
		ReleaseUID(uid)
	})

	// create a new process
	cmd := exec.Command(
		configuration.PythonPath,
		bootstrapPath,
		LIB_PATH,
	)
	cmd.Env = []string{
		// The sandbox child loads a Go c-shared library to install seccomp.
		// Disable Go runtime features that may issue housekeeping syscalls after
		// the seccomp filter is active; the prescript removes this before running
		// user code.
		"GODEBUG=decoratemappings=0,containermaxprocs=0,updatemaxprocs=0",
	}
	cmd.Dir = LIB_PATH
	cmd.ExtraFiles = []*os.File{codeReader}

	if configuration.Proxy.Socks5 != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Socks5))
		cmd.Env = append(cmd.Env, fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Socks5))
	} else if configuration.Proxy.Https != "" || configuration.Proxy.Http != "" {
		if configuration.Proxy.Https != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Https))
		}
		if configuration.Proxy.Http != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Http))
		}
	}

	if configuration.Proxy.NoProxy != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NO_PROXY=%s", configuration.Proxy.NoProxy))
	}

	for _, envVar := range configuration.AllowedEnvVars {
		if val := os.Getenv(envVar); val != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envVar, val))
		}
	}

	if len(configuration.AllowedSyscalls) > 0 {
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("ALLOWED_SYSCALLS=%s",
				strings.Trim(strings.Join(strings.Fields(fmt.Sprint(configuration.AllowedSyscalls)), ","), "[]"),
			),
		)
	}

	go func() {
		_, _ = io.WriteString(codeWriter, code)
		codeWriter.Close()
	}()

	err = outputHandler.CaptureOutput(ctx, cmd)
	if err != nil {
		codeReader.Close()
		codeWriter.Close()
		os.Remove(bootstrapPath)
		ReleaseUID(uid)
		return nil, err
	}

	return outputHandler.Result(), nil
}

func cleanupTempFiles(uid int) {
	for _, dir := range []string{
		path.Join(LIB_PATH, "tmp"),
		path.Join(LIB_PATH, "var", "tmp"),
	} {
		cleanupTempDir(dir, uid)
	}
}

func cleanupTempDir(dir string, uid int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		entryPath := path.Join(dir, entry.Name())
		info, err := os.Lstat(entryPath)
		if err != nil {
			continue
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || int(stat.Uid) != uid {
			continue
		}

		_ = os.RemoveAll(entryPath)
	}
}

func buildBootstrap(preload string, options *types.RunnerOptions, uid int) string {
	script := strings.Replace(
		string(sandbox_fs),
		"{{uid}}", strconv.Itoa(uid), 1,
	)

	script = strings.Replace(
		script,
		"{{gid}}", strconv.Itoa(static.SANDBOX_GROUP_ID), 1,
	)

	if options.EnableNetwork {
		script = strings.Replace(
			script,
			"{{enable_network}}", "1", 1,
		)
	} else {
		script = strings.Replace(
			script,
			"{{enable_network}}", "0", 1,
		)
	}

	return strings.Replace(
		script,
		"{{preload}}",
		fmt.Sprintf("%s\n", preload),
		1,
	)
}

func (p *PythonRunner) InitializeEnvironment(preload string, options *types.RunnerOptions, uid int) (string, error) {
	if !checkLibAvaliable() {
		releaseLibBinary(false)
	}

	tempCodeName := strings.ReplaceAll(uuid.New().String(), "-", "_")
	tempCodeName = strings.ReplaceAll(tempCodeName, "/", ".")

	script := buildBootstrap(preload, options, uid)

	bootstrapPath := fmt.Sprintf("%s/tmp/%s.py", LIB_PATH, tempCodeName)
	if err := runner.EnsureSandboxTempDirs(LIB_PATH); err != nil {
		return "", err
	}
	err := os.MkdirAll(path.Dir(bootstrapPath), 0755)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(bootstrapPath, []byte(script), 0600)
	if err != nil {
		return "", err
	}
	if err = syscall.Chown(bootstrapPath, uid, static.SANDBOX_GROUP_ID); err != nil {
		os.Remove(bootstrapPath)
		return "", fmt.Errorf("chown script to uid %d: %w", uid, err)
	}

	return bootstrapPath, nil
}
