# Building `mqwatch`

## Dependencies
`mqwatch` uses Go `dep` for dependency management. Mostly to facilitate a clean compile from the sources in an openshift environment.

Install Go `dep` using

```
go get github.com/golang/dep/cmd/dep
```

in your favorite shell and download all dependencies by calling

```
dep ensure -v
```

## Compiling

After installing the dependencies, compile by calling 

```
go build -a -ldflags '-extldflags "-static"' .
```

Simple enough.

