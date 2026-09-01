# CRDs

This folder contains the Home Cloud Custom Resource Definitions (CRDs). This is a minimal [kubebuilder](https://book.kubebuilder.io) project focused exclusively on building CRD manifests and generated code from Go type files.

The actual implementation of the controllers for these CRDs is elsewhere in the repository.

## Controllers

- [apps](../../cmd/operator/controller/apps/)
- [disks](../../cmd/operator/controller/disks/)
- [install](../../cmd/operator/controller/installs/)
- [wireguard](../../cmd/tunnel/wireguard/)

## How to use

Install the `controller-gen` CLI:

```shell
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0
```

Generate manifests:

```shell
controller-gen crd paths="./..." output:crd:artifacts:config=./v1/manifests
```

Generate code (`DeepCopy`, etc.):

```shell
controller-gen object paths="./..."
```
