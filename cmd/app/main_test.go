package main_test

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	containerCoverDir = "/app/cover"
	hostCoverDirEnv   = "COVER_DIR"
)

func Test_NoArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	ctx := context.Background()
	rootDir := getRootDir(t)

	container, stdout := startContainer(t, ctx, rootDir, nil)
	waitForCompletionOfTest(t, ctx, container)
	require.Equal(t, []byte("Hello, World!\n"), stdout.bytes)

	// Stop container to ensure coverage data are written.
	stopContainer(t, ctx, container)
	copyCoverage(t, ctx, rootDir, container)
}

func Test_SingleArg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	ctx := context.Background()
	rootDir := getRootDir(t)

	container, stdout := startContainer(t, ctx, rootDir, []string{"Test"})
	waitForCompletionOfTest(t, ctx, container)
	require.Equal(t, []byte("Hello, Test!\n"), stdout.bytes)

	// Stop container to ensure coverage data are written.
	stopContainer(t, ctx, container)
	copyCoverage(t, ctx, rootDir, container)
}

type containerStdoutLogConsumer struct {
	mu    sync.Mutex
	bytes []byte
}

func (c *containerStdoutLogConsumer) Accept(l testcontainers.Log) {
	if l.LogType != testcontainers.StdoutLog {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.bytes = append(c.bytes, l.Content...)
}

func getRootDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		require.True(t, ok, "failed to get caller filename")
	}

	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func startContainer(t *testing.T, ctx context.Context, rootDir string, args []string) (testcontainers.Container, *containerStdoutLogConsumer) {
	t.Helper()

	logConsumer := &containerStdoutLogConsumer{}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: rootDir,
				BuildArgs: map[string]*string{
					"COVER_INSTRUMENT": lo.ToPtr("1"),
				},
			},
			Env: map[string]string{
				"GOCOVERDIR": containerCoverDir,
			},
			Cmd: args,
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{logConsumer},
			},
		},
		Started: true,
		Logger:  testcontainers.TestLogger(t),
	})
	testcontainers.CleanupContainer(t, container, testcontainers.StopTimeout(0))
	require.NoError(t, err, "failed to create and start container")

	return container, logConsumer
}

func waitForCompletionOfTest(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	// Perform test activities - just wait for container to stop.
	err := wait.ForExit().WaitUntilReady(ctx, container)
	require.NoError(t, err, "failed to wait until container stops")
}

func stopContainer(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	err := container.Stop(ctx, lo.ToPtr(5*time.Minute))
	require.NoError(t, err, "failed to stop container")
}

func copyCoverage(t *testing.T, ctx context.Context, rootDir string, container testcontainers.Container) {
	t.Helper()

	hostCoverDir := os.Getenv(hostCoverDirEnv)
	if len(hostCoverDir) == 0 {
		return
	}

	if !filepath.IsAbs(hostCoverDir) {
		joinedPath := filepath.Join(rootDir, hostCoverDir)
		var err error
		hostCoverDir, err = filepath.Abs(joinedPath)
		require.NoErrorf(t, err, "failed to get absolute path for directory %q", joinedPath)
	}

	containerID := container.GetContainerID()
	err := copyFromContainer(ctx, containerID, containerCoverDir, hostCoverDir)
	require.NoError(t, err, "failed to copy coverage data from directory %q of container %q to host directory %q",
		containerCoverDir, containerID, hostCoverDir)
}

func copyFromContainer(ctx context.Context, containerID, containerPath, hostDir string) error {
	provider, err := createDockerProvider()
	if err != nil {
		return fmt.Errorf("get docker provider: %w", err)
	}
	defer func() { _ = provider.Close() }()

	reader, _, err := provider.Client().CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		return fmt.Errorf("copy from container: %w", err)
	}
	defer func() { _ = reader.Close() }()

	err = os.MkdirAll(hostDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create directory %q: %w", hostDir, err)
	}

	tarReader := tar.NewReader(reader)
	defer func() { _ = reader.Close() }()

	for {
		entry, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reader next entry: %w", err)
		}
		entry.Name = filepath.Clean(entry.Name)
		entryPath := filepath.Join(hostDir, entry.Name)
		switch entry.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(entryPath, os.FileMode(entry.Mode))
			if err != nil {
				return fmt.Errorf("create directory %q for entry %q: %w", entryPath, entry.Name, err)
			}
		case tar.TypeReg:
			fileDir := filepath.Dir(entryPath)
			err := os.MkdirAll(fileDir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("create directory %q for entry %q: %w", fileDir, entry.Name, err)
			}
			err = func() error {
				file, err := os.OpenFile(entryPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(entry.Mode))
				if err != nil {
					return fmt.Errorf("create file %q for entry %q: %w", entryPath, entry.Name, err)
				}
				defer func() { _ = file.Close() }()
				_, err = io.Copy(file, tarReader)
				if err != nil {
					return fmt.Errorf("copy entry %q into file %q: %w", entry.Name, entryPath, err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected type of entry %q: %v", entry.Name, entry.Typeflag)
		}
	}

	return nil
}

func createDockerProvider(opts ...testcontainers.ContainerCustomizer) (*testcontainers.DockerProvider, error) {
	// Use a dummy request to get the provider from options.
	var req testcontainers.GenericContainerRequest
	for _, opt := range opts {
		err := opt.Customize(&req)
		if err != nil {
			return nil, fmt.Errorf("customize option: %w", err)
		}
	}

	logging := req.Logger
	if logging == nil {
		logging = &noopTestcontainersLogger{}
	}

	provider, err := req.ProviderType.GetProvider(testcontainers.WithLogger(logging))
	if err != nil {
		return nil, err
	}

	closeProvider := true
	defer func() {
		if closeProvider {
			_ = provider.Close()
		}
	}()

	dockerProvider, ok := provider.(*testcontainers.DockerProvider)
	if !ok {
		return nil, fmt.Errorf("unknown type of container provider: %T", provider)
	}

	closeProvider = false
	return dockerProvider, nil
}

type noopTestcontainersLogger struct{}

func (n noopTestcontainersLogger) Printf(_ string, _ ...any) {}
