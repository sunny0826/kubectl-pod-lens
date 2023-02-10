---
title: 更多信息
---

[![asciicast](https://asciinema.org/a/400180.svg)](https://asciinema.org/a/400180)

## 交互式操作

```console
kubectl pod-lens
```

## 展示 Pod 相关 K8S 资源

```console
kubectl pod-lens <pod-name>
```

### 输入 pod name 模糊匹配

```console
kubectl pod-lens prometheus-prometheus-operator-prometheus-
kubectl pod-lens prometheus-prometheus-operator-prometheus-* # fuzzy matching
```

### 指定 LabelSelector

```console
kubectl pod-lens <pod-name> -l app=demo
```
