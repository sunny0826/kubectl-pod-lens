---
title: More Info
---

[![asciicast](https://asciinema.org/a/400180.svg)](https://asciinema.org/a/400180)

## Interactive operation

```console
kubectl pod-lens
```

## Show pod-related resources

```console
kubectl pod-lens <pod-name>
```

### Support input pod name fuzzy matching

```console
kubectl pod-lens prometheus-prometheus-operator-prometheus-
kubectl pod-lens prometheus-prometheus-operator-prometheus-* # fuzzy matching
```


### Assign LabelSelector

```console
kubectl pod-lens <pod-name> -l app=demo
```
