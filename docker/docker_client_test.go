package docker_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"knative.dev/kn-plugin-func/docker"
)

// Test that we are creating client in accordance
// with the DOCKER_HOST environment variable
func TestNewDockerClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	tmpDir := t.TempDir()
	sock := filepath.Join(tmpDir, "docker.sock")

	defer startMockDaemon(t, sock)()

	defer withEnvVar(t, "DOCKER_HOST", fmt.Sprintf("unix://%s", sock))()

	dockerClient, _, err := docker.NewClient(client.DefaultDockerHost)
	if err != nil {
		t.Error(err)
	}
	defer dockerClient.Close()

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		t.Error(err)
	}
}

func withEnvVar(t *testing.T, name, value string) func() {
	oldDh, hadDh := os.LookupEnv(name)
	err := os.Setenv(name, value)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		if hadDh {
			os.Setenv(name, oldDh)
		} else {
			os.Unsetenv(name)
		}
	}
}

func startMockDaemon(t *testing.T, sock string) func() {

	server := http.Server{}
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	// mimics /_ping endpoint
	server.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("OK"))
	})

	serErrChan := make(chan error)
	go func() {
		serErrChan <- server.Serve(listener)
	}()
	return func() {
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		err = <-serErrChan
		if err != nil && !strings.Contains(err.Error(), "Server closed") {
			t.Fatal(err)
		}
	}
}