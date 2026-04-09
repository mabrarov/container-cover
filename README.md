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
docker rmi --force golang:1.25.4 && \
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
2026/04/10 22:06:41 github.com/testcontainers/testcontainers-go - Connected to docker:
  Server Version: 29.4.0
  API Version: 1.54
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15953 MB
  Testcontainers for Go Version: v0.42.0
  Resolved Docker Host: unix:///var/run/docker.sock
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 412789f6fab5582ad1f80ab60e1bdf6f9c59867a0099f3844638b94e46e93b14
  Test ProcessID: 553d5c68-5bc1-4b94-8616-252264f373e1
    lifecycle.go:65: 🐳 Building image e232937a-0b63-435c-98d7-afe3e7969579:8a57d34d-814b-49a2-8ba8-147be9f02bec
    main_test.go:35:
                Error Trace:    /home/user/container-cover/cmd/app/main_test.go:108
                                                        /home/user/container-cover/cmd/app/main_test.go:35
                Error:          Received unexpected error:
                                create container: build image: golang:1.25.4: failed to resolve source metadata for docker.io/library/golang:1.25.4: no active sessions
                Test:           TestNoArgs
                Messages:       failed to create and start container
--- FAIL: TestNoArgs (0.78s)
=== RUN   TestSingleArg
    lifecycle.go:65: 🐳 Building image 9efd7bf9-9c4b-4d57-9844-4a1103f8eafa:45d30dd8-2f89-4c24-abff-9bd389d46129
    main_test.go:51:
                Error Trace:    /home/user/container-cover/cmd/app/main_test.go:108
                                                        /home/user/container-cover/cmd/app/main_test.go:51
                Error:          Received unexpected error:
                                create container: build image: golang:1.25.4: failed to resolve source metadata for docker.io/library/golang:1.25.4: no active sessions
                Test:           TestSingleArg
                Messages:       failed to create and start container
--- FAIL: TestSingleArg (0.16s)
FAIL
FAIL    github.com/mabrarov/container-cover/cmd/app     0.956s
FAIL
make: *** [Makefile:75: test] Error 1
```

The workaround is to pull required docker image before running test:

```bash
docker pull golang:1.25.4 && \
docker system prune --force && \
make test
```
