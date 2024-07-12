# api

This directory contains the API specification of the system. The spec is written in protocol buffers and compiled using Buf.

## Generating files for local testing

### Using `dctl`

The easiest way to generate protos is using [`dctl`](TODO):

```shell
# you only need to run this the first time
dctl api init
dctl api build
```

Any time you want to regenerate your protos just run `dctl api build` again.

## Using locally generated files

To use the generated files locally, you will need to add a `replace` to the `go.mod` of any service using the protobufs:

```go.mod
...

replace <YOUR_REPO>/api => ../../../api/gen/go

...
```
