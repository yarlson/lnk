package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type testEnv struct {
	t        *testing.T
	tempDir  string
	original string
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "lnk-v2-test-*")
	if err != nil {
		t.Fatalf("mktemp: %v", err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}

	t.Setenv("HOME", tempDir)
	t.Setenv("LNK_HOME", "")
	t.Setenv("LNK_REPO", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	env := &testEnv{
		t:        t,
		tempDir:  tempDir,
		original: original,
		stdout:   &bytes.Buffer{},
		stderr:   &bytes.Buffer{},
	}

	t.Cleanup(func() {
		_ = os.Chdir(original)
		_ = os.RemoveAll(tempDir)
	})

	return env
}

func (e *testEnv) run(args ...string) error {
	e.t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()

	rootCmd := NewRootCommand()
	rootCmd.SetOut(e.stdout)
	rootCmd.SetErr(e.stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func (e *testEnv) repoPath() string {
	return filepath.Join(e.tempDir, ".config", "lnk")
}

func (e *testEnv) createFile(path, content string) {
	e.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		e.t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatalf("write %s: %v", path, err)
	}
}

func (e *testEnv) setupRemote(files map[string]string) string {
	e.t.Helper()

	remoteDir := filepath.Join(e.tempDir, "remote.git")
	if err := os.MkdirAll(remoteDir, 0o755); err != nil {
		e.t.Fatalf("mkdir remote: %v", err)
	}
	if err := runCmd(remoteDir, "git", "init", "--bare", "--initial-branch=main"); err != nil {
		e.t.Fatalf("init bare remote: %v", err)
	}

	workingDir := filepath.Join(e.tempDir, "remote-work")
	if err := runCmd(e.tempDir, "git", "clone", remoteDir, workingDir); err != nil {
		e.t.Fatalf("clone working repo: %v", err)
	}
	for path, content := range files {
		fullPath := filepath.Join(workingDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			e.t.Fatalf("mkdir file parent: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			e.t.Fatalf("write remote file: %v", err)
		}
	}
	if err := runCmd(workingDir, "git", "add", "-A"); err != nil {
		e.t.Fatalf("git add remote files: %v", err)
	}
	if err := runCmd(workingDir, "git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "seed remote"); err != nil {
		e.t.Fatalf("git commit remote files: %v", err)
	}
	if err := runCmd(workingDir, "git", "push", "origin", "main"); err != nil {
		e.t.Fatalf("git push remote files: %v", err)
	}
	return remoteDir
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmtError(name, args, output, err)
	}
	return nil
}

func fmtError(name string, args []string, output []byte, err error) error {
	if len(output) == 0 {
		return err
	}
	return &execError{text: strings.TrimSpace(string(output))}
}

type execError struct {
	text string
}

func (e *execError) Error() string {
	return e.text
}

func TestV2CLI(t *testing.T) {
	t.Run("push does not auto commit dirty repo", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		env.createFile(filepath.Join(env.repoPath(), "notes.txt"), "dirty")

		err := env.run("push")
		if err == nil {
			t.Fatalf("expected push to fail on dirty repo")
		}
		if !strings.Contains(err.Error(), "working tree is dirty") {
			t.Fatalf("unexpected push error: %v", err)
		}
	})

	t.Run("clone only runs bootstrap when requested", func(t *testing.T) {
		env := newTestEnv(t)
		remote := env.setupRemote(map[string]string{
			"bootstrap.sh": "#!/bin/bash\ntouch bootstrap-ran.txt\n",
		})

		if err := env.run("clone", remote); err != nil {
			t.Fatalf("clone without bootstrap: %v", err)
		}
		if _, err := os.Stat(filepath.Join(env.repoPath(), "bootstrap-ran.txt")); !os.IsNotExist(err) {
			t.Fatalf("bootstrap marker should not exist without --bootstrap")
		}

		env = newTestEnv(t)
		remote = env.setupRemote(map[string]string{
			"bootstrap.sh": "#!/bin/bash\ntouch bootstrap-ran.txt\n",
		})
		if err := env.run("clone", remote, "--bootstrap"); err != nil {
			t.Fatalf("clone with bootstrap: %v", err)
		}
		if _, err := os.Stat(filepath.Join(env.repoPath(), "bootstrap-ran.txt")); err != nil {
			t.Fatalf("bootstrap marker should exist with --bootstrap: %v", err)
		}
	})

	t.Run("add rejects outside home paths", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		outside := filepath.Join(os.TempDir(), "lnk-v2-outside-home.txt")
		if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
			t.Fatalf("write outside file: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Remove(outside)
		})

		err := env.run("add", outside)
		if err == nil {
			t.Fatalf("expected add outside home to fail")
		}
		if !strings.Contains(err.Error(), "inside $HOME") {
			t.Fatalf("unexpected outside-home error: %v", err)
		}
	})

	t.Run("host ownership is exclusive", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		vimrc := filepath.Join(env.tempDir, ".vimrc")
		env.createFile(vimrc, "set number")
		if err := env.run("add", vimrc); err != nil {
			t.Fatalf("add common: %v", err)
		}

		err := env.run("add", "--host", "work", vimrc)
		if err == nil {
			t.Fatalf("expected duplicate ownership add to fail")
		}
		if !strings.Contains(err.Error(), "already managed in scope common") {
			t.Fatalf("unexpected duplicate ownership error: %v", err)
		}
	})

	t.Run("restore with host applies common and host profile", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		bashrc := filepath.Join(env.tempDir, ".bashrc")
		vimrc := filepath.Join(env.tempDir, ".vimrc")
		env.createFile(bashrc, "export PATH=/bin")
		env.createFile(vimrc, "set number")
		if err := env.run("add", bashrc); err != nil {
			t.Fatalf("add common: %v", err)
		}
		if err := env.run("add", "--host", "work", vimrc); err != nil {
			t.Fatalf("add host: %v", err)
		}
		if err := os.Remove(bashrc); err != nil {
			t.Fatalf("remove bashrc symlink: %v", err)
		}
		if err := os.Remove(vimrc); err != nil {
			t.Fatalf("remove vimrc symlink: %v", err)
		}

		if err := env.run("restore", "--host", "work"); err != nil {
			t.Fatalf("restore: %v", err)
		}
		if info, err := os.Lstat(bashrc); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("bashrc should be restored as symlink: %v", err)
		}
		if info, err := os.Lstat(vimrc); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("vimrc should be restored as symlink: %v", err)
		}
	})

	t.Run("doctor is read only by default and fixes with --fix", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		bashrc := filepath.Join(env.tempDir, ".bashrc")
		env.createFile(bashrc, "export PATH=/bin")
		if err := env.run("add", bashrc); err != nil {
			t.Fatalf("add: %v", err)
		}
		if err := os.Remove(bashrc); err != nil {
			t.Fatalf("remove symlink: %v", err)
		}

		if err := env.run("doctor"); err != nil {
			t.Fatalf("doctor scan: %v", err)
		}
		if _, err := os.Lstat(bashrc); !os.IsNotExist(err) {
			t.Fatalf("doctor without --fix should not restore symlink")
		}

		if err := env.run("doctor", "--fix"); err != nil {
			t.Fatalf("doctor fix: %v", err)
		}
		if info, err := os.Lstat(bashrc); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("doctor --fix should restore symlink: %v", err)
		}
	})

	t.Run("move re-scopes path and repoints symlink", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.run("init"); err != nil {
			t.Fatalf("init: %v", err)
		}
		zshrc := filepath.Join(env.tempDir, ".zshrc")
		env.createFile(zshrc, "export ZDOTDIR")
		if err := env.run("add", zshrc); err != nil {
			t.Fatalf("add: %v", err)
		}

		if err := env.run("move", zshrc, "--to-host", "work"); err != nil {
			t.Fatalf("move: %v", err)
		}
		output := env.stdout.String()
		if !strings.Contains(output, "Moved") {
			t.Fatalf("unexpected move output: %s", output)
		}

		if err := env.run("list"); err != nil {
			t.Fatalf("list common: %v", err)
		}
		if strings.Contains(env.stdout.String(), ".zshrc") {
			t.Fatalf("common scope should no longer contain .zshrc: %s", env.stdout.String())
		}

		if err := env.run("list", "--host", "work"); err != nil {
			t.Fatalf("list host: %v", err)
		}
		if !strings.Contains(env.stdout.String(), ".zshrc") {
			t.Fatalf("host scope should contain .zshrc: %s", env.stdout.String())
		}
	})
}
