# Kubectl Sniffer

[![Go Report Card](https://goreportcard.com/badge/github.com/sunny0826/kubectl-sniffer)](https://goreportcard.com/report/github.com/sunny0826/kubectl-sniffer)
![GitHub](https://img.shields.io/github/license/sunny0826/kubectl-sniffer.svg)
[![GitHub release](https://img.shields.io/github/release/sunny0826/kubectl-sniffer)](https://github.com/sunny0826/kubectl-sniffer/releases)

<img src="https://github.com/sunny0826/kubectl-sniffer/raw/master/doc/logo.png" width="200">

`kubectl-sniffer` is a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) that show pod-related resource information.

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
$ kubectl krew install sniffer
```

## Example

```bash
$ kubectl sniffer thanos-query-9fbb8c4bc-5x2zf
└─┬ [Namespace]  kube-system                                                                            
  └─┬ [Deployment]  thanos-query                                    Replica: 2/2                        
    ├─┬ [Node]  ip-172-25-204-130.cn-northwest-1.compute.internal   [Ready] Node IP: 172.25.204.130     
    │ └─┬ [Pod]  thanos-query-9fbb8c4bc-5x2zf                       [Running] Pod IP: 100.123.170.122   
    │   └── [Container]  thanos-query                               [Running] Restart: 0                
    └── [Secret]  default-token-7rrxp                                                                   

 Related Resources 
               
Kind:            Deployment                                  
Name:           thanos-bucket                                
Replicas:       1                                            
---             ---                                          
Kind:            Deployment                                  
Name:           thanos-compact                               
Replicas:       1                                            
---             ---                                          
Kind:            Deployment                                  
Name:           thanos-query                                 
Replicas:       2                                            
---             ---                                          
Kind:            Deployment                                  
Name:           thanos-store-0                               
Replicas:       1                                            
---             ---                                          
Kind:            Service                                     
Name:           thanos-bucket                                
Cluster IP:     100.71.173.222                               
Ports                                                        
                ---                                          
                Name: http                                   
                Port: 8080                                   
                TargetPort: http                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-compact                               
Cluster IP:     100.65.165.163                               
Ports                                                        
                ---                                          
                Name: http                                   
                Port: 10902                                  
                TargetPort: http                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-query-grpc                            
Ports                                                        
                ---                                          
                Name: grpc                                   
                Port: 10901                                  
                TargetPort: grpc                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-query-http                            
Cluster IP:     100.71.18.91                                 
Ports                                                        
                ---                                          
                Name: http                                   
                Port: 10902                                  
                TargetPort: http                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-sidecar-grpc                          
Ports                                                        
                ---                                          
                Name: grpc                                   
                Port: 10901                                  
                TargetPort: grpc                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-sidecar-http                          
Cluster IP:     100.64.94.33                                 
Ports                                                        
                ---                                          
                Name: http                                   
                Port: 10902                                  
                TargetPort: http                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-store-grpc                            
Ports                                                        
                ---                                          
                Name: grpc                                   
                Port: 10901                                  
                TargetPort: grpc                             
---             ---                                          
Kind:            Service                                     
Name:           thanos-store-http                            
Cluster IP:     100.68.88.86                                 
Ports                                                        
                ---                                          
                Name: http                                   
                Port: 10902                                  
                TargetPort: http                             
---             ---                                          
Kind:            Ingress                                     
Name:           thanos-query-http                            
Url:            https://xxx.dev.pod-lens.io/
Backend:        thanos-query-http                            
Url:            https://xxx.pod-lens.io/               
Backend:        thanos-query-http                            
LoadBalance IP:                                              
                172.25.200.106                               
                172.25.202.93                                
                172.25.202.98                                
                172.25.204.214                               
---            
Kind:            PVC                                         
Name:           compact-data-volume                          
Storage Class:  gp2                                          
Access Modes:   ReadWriteOnce                                
Size:           200Gi                                        
PV Name:        pvc-287a9257-9fca-40ed-808d-277ac796597c     
---             ---                    
```

## Reference

- [Kubectl Plugins](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)
- [Krew](https://krew.sigs.k8s.io/)

## Design

![](doc/architecture.jpg)