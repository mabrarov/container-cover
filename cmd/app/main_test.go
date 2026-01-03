package main_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/moby/go-archive"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	containerCoverDirEnv = "COVER_CONTAINER_DIR"
	hostCoverDirEnv      = "COVER_HOST_DIR"
)

func Test_NoArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	ctx := t.Context()
	rootDir := getRootDir(t)

	container := runContainer(t, ctx, rootDir, nil)
	assertContainerLog(t, ctx, container, "Hello, World!\n")

	// Stop container to ensure coverage data are written.
	stopContainer(t, ctx, container)
	copyCoverage(t, ctx, rootDir, container)
}

func Test_SingleArg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	ctx := t.Context()
	rootDir := getRootDir(t)

	container := runContainer(t, ctx, rootDir, []string{"Test"})
	assertContainerLog(t, ctx, container, "Hello, Test!\n")

	// Stop container to ensure coverage data are written.
	stopContainer(t, ctx, container)
	copyCoverage(t, ctx, rootDir, container)
}

func getRootDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		require.True(t, ok, "failed to get caller filename")
	}

	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func runContainer(t *testing.T, ctx context.Context, rootDir string, args []string) testcontainers.Container {
	t.Helper()

	containerCoverDir := os.Getenv(containerCoverDirEnv)
	var (
		buildArgs map[string]*string
		env       map[string]string
		mounts    []testcontainers.ContainerMount
	)
	if len(containerCoverDir) != 0 {
		buildArgs = map[string]*string{
			"COVER_INSTRUMENT": toPtr("1"),
		}
		env = map[string]string{
			"GOCOVERDIR": containerCoverDir,
		}
		mounts = []testcontainers.ContainerMount{
			testcontainers.VolumeMount("", testcontainers.ContainerMountTarget(containerCoverDir)),
		}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:   rootDir,
				BuildArgs: buildArgs,
			},
			Cmd:    args,
			Env:    env,
			Mounts: mounts,
		},
		Started: true,
		Logger:  log.TestLogger(t),
	})
	testcontainers.CleanupContainer(t, container, testcontainers.StopTimeout(0))
	require.NoError(t, err, "failed to create and start container")

	err = wait.ForExit().WaitUntilReady(ctx, container)
	require.NoError(t, err, "failed to wait until container stops")

	return container
}

func assertContainerLog(t *testing.T, ctx context.Context, container testcontainers.Container, expected string) {
	t.Helper()

	logReader, err := container.Logs(ctx)
	require.NoError(t, err, "failed to get container logs")
	defer func() { _ = logReader.Close() }()

	actual, err := io.ReadAll(logReader)
	require.NoError(t, err, "failed to read container logs")

	require.Equal(t, []byte(expected), actual)
}

func stopContainer(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	err := container.Stop(ctx, toPtr(5*time.Minute))
	require.NoError(t, err, "failed to stop container")
}

func copyCoverage(t *testing.T, ctx context.Context, rootDir string, container testcontainers.Container) {
	t.Helper()

	containerCoverDir := os.Getenv(containerCoverDirEnv)
	if len(containerCoverDir) == 0 {
		return
	}

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
	t.Logf("Copying coverage data from directory %q of container %q to host directory %q",
		containerCoverDir, containerID, hostCoverDir)

	err := copyFromContainer(ctx, containerID, containerCoverDir, hostCoverDir)
	require.NoError(t, err, "failed to copy coverage data from directory %q of container %q to host directory %q",
		containerCoverDir, containerID, hostCoverDir)
}

func toPtr[T any](v T) *T {
	return &v
}

func copyFromContainer(ctx context.Context, containerID, containerPath, hostDir string) error {
	provider, err := createDockerProvider()
	if err != nil {
		return fmt.Errorf("create docker provider: %w", err)
	}
	defer func() { _ = provider.Close() }()

	reader, stat, err := provider.Client().CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		return fmt.Errorf("copy %q from container %q: %w", containerPath, containerID, err)
	}
	defer func() { _ = reader.Close() }()

	err = os.MkdirAll(hostDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("create directory %q: %w", hostDir, err)
	}

	srcInfo := archive.CopyInfo{
		Path:       containerPath,
		Exists:     true,
		IsDir:      stat.Mode.IsDir(),
		RebaseName: "",
	}

	err = archive.CopyTo(reader, srcInfo, hostDir)
	if err != nil {
		return fmt.Errorf("untar to %q: %w", hostDir, err)
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
		logging = log.Default()
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
