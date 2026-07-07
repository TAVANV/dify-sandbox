package integrationtests_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	python_runner "github.com/langgenius/dify-sandbox/internal/core/runner/python"
	"github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/service"
)

func TestPythonBase64(t *testing.T) {
	// Test case for base64
	runMultipleTestings(t, 50, func(t *testing.T) {
		resp := service.RunPython3Code(context.TODO(), `
import base64
print(base64.b64decode(base64.b64encode(b"hello world")).decode())
		`, "", &types.RunnerOptions{
			EnableNetwork: true,
		})
		if resp.Code != 0 {
			t.Fatal(resp)
		}

		if resp.Data.(*service.RunCodeResponse).Stderr != "" {
			t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
		}

		if !strings.Contains(resp.Data.(*service.RunCodeResponse).Stdout, "hello world") {
			t.Fatalf("unexpected output: %s\n", resp.Data.(*service.RunCodeResponse).Stdout)
		}
	})
}

func TestPythonJSON(t *testing.T) {
	runMultipleTestings(t, 50, func(t *testing.T) {
		// Test case for json
		resp := service.RunPython3Code(context.TODO(), `
import json
print(json.dumps({"hello": "world"}))
		`, "", &types.RunnerOptions{
			EnableNetwork: true,
		})
		if resp.Code != 0 {
			t.Fatal(resp)
		}

		if resp.Data.(*service.RunCodeResponse).Stderr != "" {
			t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
		}

		if !strings.Contains(resp.Data.(*service.RunCodeResponse).Stdout, `{"hello": "world"}`) {
			t.Fatalf("unexpected output: %s\n", resp.Data.(*service.RunCodeResponse).Stdout)
		}
	})
}

func TestPythonRequests(t *testing.T) {
	// Test case for http
	runMultipleTestings(t, 1, func(t *testing.T) {
		resp := service.RunPython3Code(context.TODO(), `
import requests
print(requests.get("https://www.bilibili.com").content)
	`, "", &types.RunnerOptions{
			EnableNetwork: true,
		})
		if resp.Code != 0 {
			t.Fatal(resp)
		}

		if resp.Data.(*service.RunCodeResponse).Stderr != "" {
			t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
		}

		if !strings.Contains(resp.Data.(*service.RunCodeResponse).Stdout, "bilibili") {
			t.Fatalf("unexpected output: %s\n", resp.Data.(*service.RunCodeResponse).Stdout)
		}
	})
}

func TestPythonHttpx(t *testing.T) {
	// Test case for http
	runMultipleTestings(t, 1, func(t *testing.T) {
		resp := service.RunPython3Code(context.TODO(), `
import httpx
print(httpx.get("https://www.bilibili.com").content)
	`, "", &types.RunnerOptions{
			EnableNetwork: true,
		})
		if resp.Code != 0 {
			t.Fatal(resp)
		}

		if resp.Data.(*service.RunCodeResponse).Stderr != "" {
			t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
		}

		if !strings.Contains(resp.Data.(*service.RunCodeResponse).Stdout, "bilibili") {
			t.Fatalf("unexpected output: %s\n", resp.Data.(*service.RunCodeResponse).Stdout)
		}
	})
}

func TestPythonTimezone(t *testing.T) {
	// Test case for time
	runMultipleTestings(t, 1, func(t *testing.T) {
		resp := service.RunPython3Code(context.TODO(), `
from datetime import datetime
from zoneinfo import ZoneInfo

print(datetime.now(ZoneInfo("Asia/Shanghai")).isoformat())
		`, "", &types.RunnerOptions{
			EnableNetwork: true,
		})
		if resp.Code != 0 {
			t.Fatal(resp)
		}

		if resp.Data.(*service.RunCodeResponse).Stderr != "" {
			t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
		}

		stdout := resp.Data.(*service.RunCodeResponse).Stdout
		// trim \n
		stdout = strings.TrimSpace(stdout)
		// check if stdout match time format
		_, err := time.Parse("2006-01-02T15:04:05.000000+08:00", stdout)
		if err != nil {
			t.Fatalf("unexpected output: %s, error: %v\n", stdout, err)
		}
	})
}

func TestPythonFd3Transport(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
marker = "fd3-transport"
print(marker)
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	if resp.Data.(*service.RunCodeResponse).Stderr != "" {
		t.Fatalf("unexpected error: %s\n", resp.Data.(*service.RunCodeResponse).Stderr)
	}

	if !strings.Contains(resp.Data.(*service.RunCodeResponse).Stdout, "fd3-transport") {
		t.Fatalf("unexpected output: %s\n", resp.Data.(*service.RunCodeResponse).Stdout)
	}
}

func TestPythonLoggingStaysInStderr(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
import logging

logging.basicConfig(level=logging.INFO)
logging.info("Starting task...")
print("ok")
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)

	if !strings.Contains(data.Stdout, "ok") {
		t.Fatalf("unexpected output: %s\n", data.Stdout)
	}

	if !strings.Contains(data.Stderr, "INFO:root:Starting task...") {
		t.Fatalf("expected logging output in stderr, got: %s\n", data.Stderr)
	}

	if data.Error != "" {
		t.Fatalf("expected empty execution error, got: %s\n", data.Error)
	}

	if data.ExitCode != 0 {
		t.Fatalf("expected zero exit code, got: %d\n", data.ExitCode)
	}
}

func TestPythonCanUseTemporaryDirectories(t *testing.T) {
	explicitName := "explicit-temp-check.txt"
	hostExplicitPath := path.Join(python_runner.LIB_PATH, "tmp", explicitName)
	_ = os.Remove(hostExplicitPath)

	resp := service.RunPython3Code(context.TODO(), `
import os
import stat
import tempfile

for directory in ("/tmp", "/var/tmp"):
    mode = os.stat(directory).st_mode
    assert stat.S_ISDIR(mode), f"{directory} is not a directory"
    assert mode & stat.S_IWOTH, f"{directory} is not world-writable"
    assert mode & stat.S_ISVTX, f"{directory} does not have sticky bit"

with tempfile.NamedTemporaryFile(mode="w+", delete=False) as tmp:
    tmp.write("sandbox-temp-ok")
    tmp_path = tmp.name

with open(tmp_path) as tmp:
    assert tmp.read() == "sandbox-temp-ok"

explicit_path = "/tmp/`+explicitName+`"
with open(explicit_path, "w") as tmp:
    tmp.write("explicit-ok")

with open(explicit_path) as tmp:
    assert tmp.read() == "explicit-ok"

mkdir_path = "/tmp/os-mkdir-check"
os.mkdir(mkdir_path)
assert os.path.isdir(mkdir_path), "os.mkdir did not create a directory"
with open(os.path.join(mkdir_path, "nested.txt"), "w") as tmp:
    tmp.write("mkdir-ok")

mkdtemp_path = tempfile.mkdtemp()
assert os.path.isdir(mkdtemp_path), "tempfile.mkdtemp did not create a directory"
with open(os.path.join(mkdtemp_path, "nested.txt"), "w") as tmp:
    tmp.write("mkdtemp-ok")

with tempfile.TemporaryDirectory() as tmp_dir:
    assert os.path.isdir(tmp_dir), "TemporaryDirectory did not create a directory"
    with open(os.path.join(tmp_dir, "nested.txt"), "w") as tmp:
        tmp.write("temporary-directory-ok")

print(tmp_path)
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if !strings.Contains(data.Stdout, "/tmp/") {
		t.Fatalf("expected tempfile path under /tmp, got: %q\n", data.Stdout)
	}
	if _, err := os.Stat(hostExplicitPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp cleanup to remove %s, stat err=%v", hostExplicitPath, err)
	}
}

func TestPythonCanImportPyMySQL(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
import pymysql

print(pymysql.__version__)
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if strings.TrimSpace(data.Stdout) == "" {
		t.Fatal("expected pymysql version in stdout")
	}
}

func TestPythonCanImportGmSSL(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
import gmssl

print(gmssl.__path__[0])
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if strings.TrimSpace(data.Stdout) == "" {
		t.Fatal("expected gmssl package path in stdout")
	}
}

func TestPythonExceptionPopulatesErrorAndStderr(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
raise ValueError("bad input")
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)

	if !strings.Contains(data.Stderr, "ValueError: bad input") {
		t.Fatalf("expected traceback in stderr, got: %s\n", data.Stderr)
	}

	if !strings.Contains(data.Error, "process exited with code") {
		t.Fatalf("expected error to include exit code, got: %s\n", data.Error)
	}

	if !strings.Contains(data.Error, data.Stderr) {
		t.Fatalf("expected error to include full stderr, got: %s\n", data.Error)
	}

	if data.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got: %d\n", data.ExitCode)
	}
}
func TestPythonNoProxyEnvPropagation(t *testing.T) {
	resp := service.RunPython3Code(context.TODO(), `
import os
print(os.environ.get('NO_PROXY', ''))
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if !strings.Contains(data.Stdout, "test.no-proxy.internal") {
		t.Fatalf("expected NO_PROXY to be propagated to subprocess, got: %q\n", data.Stdout)
	}
}

func TestPythonAllowedEnvVarsPropagation(t *testing.T) {
	t.Setenv("TEST_SANDBOX_ENV_VAR", "hello_from_allowed_env")

	resp := service.RunPython3Code(context.TODO(), `
import os
print(os.environ.get('TEST_SANDBOX_ENV_VAR', ''))
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if !strings.Contains(data.Stdout, "hello_from_allowed_env") {
		t.Fatalf("expected TEST_SANDBOX_ENV_VAR to be propagated to subprocess, got: %q\n", data.Stdout)
	}
}

func TestPythonUnlistedEnvVarNotPropagated(t *testing.T) {
	t.Setenv("UNLISTED_ENV_VAR", "should_not_appear")

	resp := service.RunPython3Code(context.TODO(), `
import os
print(os.environ.get('UNLISTED_ENV_VAR', 'not_found'))
	`, "", &types.RunnerOptions{
		EnableNetwork: true,
	})
	if resp.Code != 0 {
		t.Fatal(resp)
	}

	data := resp.Data.(*service.RunCodeResponse)
	if data.Stderr != "" {
		t.Fatalf("unexpected stderr: %s\n", data.Stderr)
	}
	if strings.Contains(data.Stdout, "should_not_appear") {
		t.Fatalf("expected UNLISTED_ENV_VAR NOT to be propagated, but it was: %q\n", data.Stdout)
	}
}
