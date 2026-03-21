# Go test coverage report from container

Reporting Go application test coverage when application runs in docker container.

## Building

```shell
make build
```

## Building image

```shell
docker build -t container-cover-app .
```

## Running in container

```shell
docker run --rm container-cover-app User
```

Expected output is:

```text
Hello, User!
```

## Testing with coverage

```shell
make test-cover
```

Expected output ends with text like:

```text
github.com/mabrarov/container-cover/cmd/app/main.go:8:  main            83.3%
total:                                                  (statements)    83.3
```

# Reproducing issue with BuildKit and Testcontainers for Go

Run:

```bash
docker rmi --force golang:1.24.9 && \
docker system prune --force && \
make test
```

Expected output ends with:

```text
...
PASS
ok      github.com/mabrarov/container-cover/cmd/app     ...
```

Actual output (the output when issue happens) looks like:

```text
=== RUN   TestNoArgs
2026/03/24 16:40:50 github.com/testcontainers/testcontainers-go - Connected to docker:
  Server Version: 28.2.2
  API Version: 1.50
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15953 MB
  Testcontainers for Go Version: v0.39.0
  Resolved Docker Host: unix:///var/run/docker.sock
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 68c6136500b50b097597ca6710ffc5c8d914044ea582049efce2569c3045076f
  Test ProcessID: d646e67d-a253-41ac-9759-4ea1dcc095d6
    lifecycle.go:66: 🐳 Building image 8308bd97-a887-41d2-9bc8-45147697b97f:302790fa-c5ad-4415-a775-13c18cb3462c
    main_test.go:34:
                Error Trace:    /home/user/container-cover/cmd/app/main_test.go:107
                                                        /home/user/container-cover/cmd/app/main_test.go:34
                Error:          Received unexpected error:
                                create container: build image: golang:1.24.9: failed to resolve source metadata for docker.io/library/golang:1.24.9: no active sessions
                Test:           TestNoArgs
                Messages:       failed to create and start container
--- FAIL: TestNoArgs (0.62s)
=== RUN   TestSingleArg
    lifecycle.go:66: 🐳 Building image 8293a67d-b701-4b18-a877-5733825748eb:ac606439-d64e-4dc7-935d-df0abc9f95c9
    main_test.go:50:
                Error Trace:    /home/user/container-cover/cmd/app/main_test.go:107
                                                        /home/user/container-cover/cmd/app/main_test.go:50
                Error:          Received unexpected error:
                                create container: build image: golang:1.24.9: failed to resolve source metadata for docker.io/library/golang:1.24.9: no active sessions
                Test:           TestSingleArg
                Messages:       failed to create and start container
--- FAIL: TestSingleArg (0.16s)
FAIL
FAIL    github.com/mabrarov/container-cover/cmd/app     0.796s
FAIL
make: *** [Makefile:75: test] Error 1
```

The workaround is to pull required docker image before running test:

```bash
docker pull golang:1.24.9 && \
docker system prune --force && \
make test
```
