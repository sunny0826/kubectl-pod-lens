# Kubectl Pod Lens

[![Go Report Card](https://goreportcard.com/badge/github.com/sunny0826/kubectl-sniffer)](https://goreportcard.com/report/github.com/sunny0826/kubectl-sniffer)
![GitHub](https://img.shields.io/github/license/sunny0826/kubectl-sniffer.svg)
[![GitHub release](https://img.shields.io/github/release/sunny0826/kubectl-sniffer)](https://github.com/sunny0826/kubectl-sniffer/releases)

<p align="center">
    <img src="https://github.com/sunny0826/kubectl-sniffer/raw/master/doc/logo.png" width="200">
</p>
`kubectl-pod-lens` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that show pod-related resource information.

The plugin can display pod-related:
* Workloads(Deployment,StatefulSet,DaemonSet)
* Namespace
* Node
* Service
* Ingress
* ConfigMap
* Secret
* HPA

## Requirements

- Kubernetes 1.10.0+
- Kubectl 1.13.0+
- Krew 0.4.0+

## Installation

```shell
$ kubectl krew install pod-lens
```

## Example

![](doc/example.png)

## Reference

- [Kubectl Plugins](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [Krew](https://krew.sigs.k8s.io/)

## Design

![](doc/architecture.png)