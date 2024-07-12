# `pipelines/`

This directory contains all the pipelines to automate continuous integration and continuous deployment of a `draft` application.

## Getting Started

### Requirements

* [Kubernetes](https://kubernetes.io/docs/tasks/tools/) (if on Mac or Windows you can use the Kubernetes engine built into Docker Desktop)

### Install `dctl`

First you need to install the `dctl` CLI:

```shell
cd tools/dctl
go install
```

### Install k8s manifests

Now, initialize the pipelines in your k8s cluster by running the below command and following the prompts:

```shell
dctl pipelines init
```

> [!TIP]
> Depending on your SSH setup, you may need to add a flag like `-f "$HOME/.ssh/id_ed25519"` to specify a non-default SSH key file.

### Done!

You should now be good to run `dctl pipelines run` commands! Try one out:

```shell
dctl pipelines run --pipeline go-binary-run --directory tools/dctl --revision main
```

And remember that you can keep track of all runs in the dashboard using:

```shell
dctl pipelines dashboard
```

Happy pipelineing!
