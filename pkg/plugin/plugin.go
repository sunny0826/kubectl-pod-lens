package plugin

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type AllInfo struct {
	SvcList       *v1.ServiceList
	IngList       *v1beta1.IngressList
	PvcList       *v1.PersistentVolumeClaimList
	ConfigMapList *v1.ConfigMapList
	SecretList    *v1.SecretList
}

type SnifferPlugin struct {
	config    *rest.Config
	Clientset *kubernetes.Clientset
	PodObject *v1.Pod
	Labels    map[string]string
	AllInfo   AllInfo
}

func NewSnifferPlugin(configFlags *genericclioptions.ConfigFlags) (*SnifferPlugin, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, errors.New("Failed to read kubeconfig, exiting.")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("Failed to create API clientset")
	}

	return &SnifferPlugin{
		config:    config,
		Clientset: clientset,
	}, nil
}

func (sf *SnifferPlugin) findPodByName(name string, namespace string) error {
	podFieldSelector := "metadata.name=" + name

	// we will seek the whole cluster if namespace is not passed as a flag (it will be a "" string)
	podFind, err := sf.Clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{FieldSelector: podFieldSelector})
	if err != nil || len(podFind.Items) == 0 {
		return errors.New("Failed to get pod: check your parameters, set a context or verify API server.")
	}

	sf.PodObject = &podFind.Items[0]
	if sf.PodObject.Spec.NodeName == "" || sf.PodObject == nil {
		return errors.New("Pod is not assigned to a node yet, it's still pending scheduling probably.")
	}
	sf.Labels = sf.PodObject.Labels
	return nil
}

func (sf *SnifferPlugin) findSvcByLabel(namespace string) error {
	var svcLabelSelector string
	if _, ok := sf.Labels["release"]; ok {
		svcLabelSelector = "release=" + sf.Labels["release"]
	}

	svcFind, err := sf.Clientset.CoreV1().Services(namespace).List(metav1.ListOptions{LabelSelector: svcLabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SvcList = svcFind
	for _, v := range svcFind.Items {
		fmt.Printf("svcName: %s\n", v.Name)
	}
	return nil
}

func (sf *SnifferPlugin) findIngressByLabel(namespace string) error {
	var ingLabelSelector string
	if _, ok := sf.Labels["release"]; ok {
		ingLabelSelector = "release=" + sf.Labels["release"]
	}
	ingFind, err := sf.Clientset.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{LabelSelector: ingLabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.IngList = ingFind
	for _, v := range ingFind.Items {
		fmt.Printf("ingName: %s\n", v.Name)
	}
	return nil
}

func (sf *SnifferPlugin) findPVCByLabel(namespace string) error {
	var ingLabelSelector string
	if _, ok := sf.Labels["release"]; ok {
		ingLabelSelector = "release=" + sf.Labels["release"]
	}
	pvcFind, err := sf.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{LabelSelector: ingLabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.PvcList = pvcFind
	for _, v := range pvcFind.Items {
		fmt.Printf("pvcName: %s\n", v.Name)
		fmt.Printf("pvName: %s\n", v.Spec.VolumeName)
	}
	return nil
}

func (sf *SnifferPlugin) findConfigMapByLabel(namespace string) error {
	var ingLabelSelector string
	if _, ok := sf.Labels["release"]; ok {
		ingLabelSelector = "release=" + sf.Labels["release"]
	}
	configMapFind, err := sf.Clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{LabelSelector: ingLabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.ConfigMapList = configMapFind
	for _, v := range configMapFind.Items {
		fmt.Printf("configMapName: %s\n", v.Name)
	}
	return nil
}

func (sf *SnifferPlugin) findSecretByLabel(namespace string) error {
	var ingLabelSelector string
	if _, ok := sf.Labels["release"]; ok {
		ingLabelSelector = "release=" + sf.Labels["release"]
	}
	secretFind, err := sf.Clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{LabelSelector: ingLabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SecretList = secretFind
	for _, v := range secretFind.Items {
		fmt.Printf("secretName: %s\n", v.Name)
	}
	return nil
}

func RunPlugin(configFlags *genericclioptions.ConfigFlags, outputCh chan string) error {
	sf, err := NewSnifferPlugin(configFlags)
	if err != nil {
		return err
	}

	podName := <-outputCh

	if err := sf.findPodByName(podName, *configFlags.Namespace); err != nil {
		return err
	}

	fmt.Printf("namespace: %s\n", sf.PodObject.Namespace)
	fmt.Printf("podName: %s\n", podName)
	fmt.Printf("nodeName: %s\n", sf.PodObject.Spec.NodeName)
	for _, existingOwnerRef := range sf.PodObject.GetOwnerReferences() {
		ownerKind := strings.ToLower(existingOwnerRef.Kind)
		workloadName := existingOwnerRef.Name
		fmt.Printf("type: %s\n", ownerKind)
		fmt.Printf("workloadName: %s\n", workloadName)
	}
	for _, val := range sf.PodObject.Status.ContainerStatuses {
		fmt.Printf("containers: %s\n", val.Name)
	}
	for _, val := range sf.PodObject.Status.InitContainerStatuses {
		fmt.Printf("init: %s\n", val.Name)
	}
	if err = sf.findSvcByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}
	if err = sf.findIngressByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}
	if err = sf.findPVCByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}
	if err = sf.findConfigMapByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}
	if err = sf.findSecretByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}
	return nil
}
