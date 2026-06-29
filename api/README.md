# api

This directory contains the API specification of the system. The spec is written in protocol buffers and compiled using Buf.

## Generating files

### Using `dctl`

The easiest way to generate protos is using [`dctl`](TODO):

```shell
dctl api init # you only need to run this the first time
dctl api build
```

Any time you want to regenerate your protos just run `dctl api build` again.
