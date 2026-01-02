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

## Testing

```shell
make test
```

Expected output ends with text like:

```text
github.com/mabrarov/container-cover/cmd/app/main.go:8:  main            83.3%
total:                                                  (statements)    83.3
```
