package plugin

import (
	"os"
	"strings"

	mapset "github.com/deckarep/golang-set"

	"github.com/i582/cfmt"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	appsv1 "k8s.io/api/apps/v1"
	autov1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Workload struct {
	Type     string
	Name     string
	Replicas int32
}

type AllInfo struct {
	DeployList    *appsv1.DeploymentList
	StsList       *appsv1.StatefulSetList
	DsList        *appsv1.DaemonSetList
	SvcList       *v1.ServiceList
	IngList       *v1beta1.IngressList
	PvcList       *v1.PersistentVolumeClaimList
	ConfigMapList *v1.ConfigMapList
	SecretList    *v1.SecretList
	Hpa           *autov1.HorizontalPodAutoscaler
	Workload      Workload
}

type SnifferPlugin struct {
	config        *rest.Config
	Clientset     *kubernetes.Clientset
	PodObject     *v1.Pod
	LabelSelector string
	AllInfo       AllInfo
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
		return errors.New("Failed to get pod: [" + name + "], please check your parameters, set a context or verify API server.")
	}

	sf.PodObject = &podFind.Items[0]
	if sf.PodObject.Spec.NodeName == "" || sf.PodObject == nil {
		return errors.New("Pod is not assigned to a node yet, it's still pending scheduling probably.")
	}
	return nil
}

func (sf *SnifferPlugin) getLabelByPod() error {
	var labelSelector string
	labels := sf.PodObject.Labels
	if _, ok := labels["release"]; ok {
		labelSelector = "release=" + labels["release"]
	} else if _, ok = labels["app"]; ok {
		labelSelector = "app=" + labels["app"]
	} else if _, ok = labels["k8s-app"]; ok {
		labelSelector = "k8s-app=" + labels["k8s-app"]
	} else if _, ok = labels["app.kubernetes.io/name"]; ok {
		labelSelector = "app.kubernetes.io/name=" + labels["app.kubernetes.io/name"]
	} else {
		_, _ = cfmt.Println("Failed to get other, These labels do not exist: {{[release]}}::green {{[app]}}::green {{[k8s-app]}}::green {{[app.kubernetes.io/name]}}::green")
		os.Exit(1)
	}
	sf.LabelSelector = labelSelector
	return nil
}

func (sf *SnifferPlugin) findDeployByLabel(namespace string) error {
	deployFind, err := sf.Clientset.AppsV1().Deployments(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.DeployList = deployFind
	return nil
}

func (sf *SnifferPlugin) findStsByLabel(namespace string) error {
	stsFind, err := sf.Clientset.AppsV1().StatefulSets(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.StsList = stsFind
	return nil
}

func (sf *SnifferPlugin) findDsByLabel(namespace string) error {
	dsFind, err := sf.Clientset.AppsV1().DaemonSets(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.DsList = dsFind
	return nil
}

func (sf *SnifferPlugin) findSvcByLabel(namespace string) error {
	svcFind, err := sf.Clientset.CoreV1().Services(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SvcList = svcFind
	return nil
}

func (sf *SnifferPlugin) findIngressByLabel(namespace string) error {
	ingFind, err := sf.Clientset.ExtensionsV1beta1().Ingresses(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.IngList = ingFind
	return nil
}

func (sf *SnifferPlugin) findPVCByLabel(namespace string) error {
	pvcFind, err := sf.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.PvcList = pvcFind
	return nil
}

func (sf *SnifferPlugin) findConfigMapByLabel(namespace string) error {
	configMapFind, err := sf.Clientset.CoreV1().ConfigMaps(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.ConfigMapList = configMapFind
	return nil
}

func (sf *SnifferPlugin) findSecretByLabel(namespace string) error {
	secretFind, err := sf.Clientset.CoreV1().Secrets(namespace).List(
		metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SecretList = secretFind
	return nil
}

func (sf *SnifferPlugin) getOwnerByPod() error {
	for _, existingOwnerRef := range sf.PodObject.GetOwnerReferences() {
		ownerKind := strings.ToLower(existingOwnerRef.Kind)
		switch ownerKind {
		case "replicaset":
			rsObject, err := sf.Clientset.AppsV1().ReplicaSets(
				sf.PodObject.GetNamespace()).Get(
				existingOwnerRef.Name,
				metav1.GetOptions{})
			if err != nil {
				return errors.New("Failed to retrieve replica set data, AppsV1 API was not available.")
			}

			for _, own := range rsObject.GetOwnerReferences() {
				sf.AllInfo.Workload = Workload{
					Name:     own.Name,
					Type:     own.Kind,
					Replicas: rsObject.Status.Replicas,
				}
			}
		case "statefulset":
			ssObject, err := sf.Clientset.AppsV1().StatefulSets(
				sf.PodObject.GetNamespace()).Get(
				existingOwnerRef.Name,
				metav1.GetOptions{})
			if err != nil {
				return errors.New("Failed to retrieve stateful set data, AppsV1 API was not available.")
			}
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: ssObject.Status.Replicas,
			}
		case "daemonset":
			dsObject, err := sf.Clientset.AppsV1().DaemonSets(
				sf.PodObject.GetNamespace()).Get(
				existingOwnerRef.Name,
				metav1.GetOptions{})
			if err != nil {
				return errors.New("Failed to retrieve daemon set data, AppsV1 API was not available.")
			}
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: dsObject.Status.DesiredNumberScheduled,
			}
		default:
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: 0,
			}
		}
	}
	return nil
}

func (sf *SnifferPlugin) findHpaByName(namespace string) error {
	hpaFind, err := sf.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(
		metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, hpa := range hpaFind.Items {
		if hpa.Spec.ScaleTargetRef.Name == sf.AllInfo.Workload.Name {
			sf.AllInfo.Hpa = &hpa
		}
	}
	return nil
}

func (sf *SnifferPlugin) printPodLeveledList() error {
	var leveledList pterm.LeveledList
	cfmt.RegisterStyle("pod", func(s string) string {
		return cfmt.Sprintf("{{%s}}::green|bold", s)
	})
	cfmt.RegisterStyle("restart", func(s string) string {
		return cfmt.Sprintf("{{%s}}::green|bold", s)
	})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 0,
		Text: cfmt.Sprintf("{{ [Namespace] }}::cyan|bold %s", sf.PodObject.Namespace)})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 1,
		Text: cfmt.Sprintf("{{ [%s] }}::lightBlue|bold %s replica: {{%d}}::green",
			sf.AllInfo.Workload.Type, sf.AllInfo.Workload.Name, sf.AllInfo.Workload.Replicas)})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 2,
		Text: cfmt.Sprintf("{{ [Node] }}::magenta|bold %s", sf.PodObject.Spec.NodeName)})
	if sf.PodObject.Status.Phase != "Running" {
		cfmt.RegisterStyle("pod", func(s string) string {
			return cfmt.Sprintf("{{%s}}::red|bold", s)
		})
	}
	podInfo := cfmt.Sprintf("{{ [Pod] }}::blue|bold %s {{[%s]}}::pod",
		sf.PodObject.Name, sf.PodObject.Status.Phase)
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 3, Text: podInfo})
	for _, val := range sf.PodObject.Status.InitContainerStatuses {
		if val.State.Terminated.Reason != "Completed" {
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}
		if val.RestartCount != 0 {
			cfmt.RegisterStyle("restart", func(s string) string {
				return cfmt.Sprintf("{{%s}}::yellow|bold", s)
			})
		}
		initInfo := cfmt.Sprintf("{{ [initContainer] }}::gray|bold %s {{[%s]}}::pod restart: {{%d}}::restart", val.Name, val.State.Terminated.Reason, val.RestartCount)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 4, Text: initInfo})
	}
	for _, val := range sf.PodObject.Status.ContainerStatuses {
		state := "Ready"
		if !val.Ready {
			state = "Not Ready"
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}
		if val.RestartCount != 0 {
			cfmt.RegisterStyle("restart", func(s string) string {
				return cfmt.Sprintf("{{%s}}::yellow|bold", s)
			})
		}
		containerInfo := cfmt.Sprintf("{{ [Container] }}::lightGreen|bold %s {{[%s]}}::pod restart: {{%d}}::restart", val.Name, state, val.RestartCount)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 4, Text: containerInfo})
	}
	for _, val := range sf.PodObject.Spec.Volumes {
		secretList := mapset.NewSet()
		configmapList := mapset.NewSet()
		pvcList := mapset.NewSet()
		if val.Projected != nil {
			for _, i := range val.Projected.Sources {
				if i.Secret != nil {
					secretList.Add(i.Secret.Name)
				}
				if i.ConfigMap != nil {
					configmapList.Add(i.ConfigMap.Name)
				}
			}
		}
		if val.ConfigMap != nil {
			configmapList.Add(val.ConfigMap.Name)
		}
		if val.Secret != nil {
			secretList.Add(val.Secret.SecretName)
		}
		if val.PersistentVolumeClaim != nil {
			pvcList.Add(val.PersistentVolumeClaim.ClaimName)
		}
		for _, p := range pvcList.ToSlice() {
			leveledList = append(leveledList, pterm.LeveledListItem{Level: 2,
				Text: cfmt.Sprintf("{{ [PVC] }}::yellow|bold %s", p)})
		}
		for _, c := range configmapList.ToSlice() {
			leveledList = append(leveledList, pterm.LeveledListItem{Level: 2,
				Text: cfmt.Sprintf("{{ [ConfigMap] }}::lightMagenta|bold %s", c)})
		}
		for _, s := range secretList.ToSlice() {
			leveledList = append(leveledList, pterm.LeveledListItem{Level: 2,
				Text: cfmt.Sprintf("{{ [Secret] }}::red|bold %s", s)})
		}
	}
	root := pterm.NewTreeFromLeveledList(leveledList)
	_ = pterm.DefaultTree.WithRoot(root).Render()
	return nil
}

func (sf *SnifferPlugin) printResourceTalbe() error {
	var data [][]string

	cfmt.RegisterStyle("url", func(s string) string {
		return cfmt.Sprintf("{{%s}}::yellow|underline", s)
	})
	for _, deploy := range sf.AllInfo.DeployList.Items {
		data = append(data, []string{"Deployment", deploy.Name, cfmt.Sprintf("Replicas:{{%d}}::yellow", deploy.Status.Replicas)})
	}

	for _, sts := range sf.AllInfo.StsList.Items {
		data = append(data, []string{"StatefulSet", sts.Name, cfmt.Sprintf("Replicas:{{%d}}::yellow", sts.Status.Replicas)})
	}

	for _, ds := range sf.AllInfo.DsList.Items {
		data = append(data, []string{"DaemonSet", ds.Name, cfmt.Sprintf("Replicas:{{%d}}::yellow", ds.Status.DesiredNumberScheduled)})
	}

	for _, svc := range sf.AllInfo.SvcList.Items {
		if svc.Spec.ClusterIP != "None" {
			detail := cfmt.Sprintf("ClusterIP:{{%s}}::yellow", svc.Spec.ClusterIP)
			data = append(data, []string{"Service", svc.Name, detail})
		}
		var ports string
		for _, v := range svc.Spec.Ports {
			if v.TargetPort.IntVal == 0 {
				ports = cfmt.Sprintf("Name:{{%s}}::yellow\nPort:{{%d}}::yellow\nTargetPort:{{%s}}::yellow", v.Name, v.Port, v.TargetPort.StrVal)
			} else {
				ports = cfmt.Sprintf("Name:{{%s}}::yellow\nPort:{{%d}}::yellow\nTargetPort:{{%d}}::yellow", v.Name, v.Port, v.TargetPort.IntVal)
			}
			data = append(data, []string{"Service", svc.Name, ports})
		}
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				data = append(data, []string{"Service", svc.Name, cfmt.Sprintf("IP:{{%s}}::url", ing.IP)})
			}
			if ing.Hostname != "" {
				data = append(data, []string{"Service", svc.Name, cfmt.Sprintf("Host:{{https://%s}}::url", ing.Hostname)})
			}
		}

	}
	for _, ing := range sf.AllInfo.IngList.Items {
		for _, r := range ing.Spec.Rules {
			for _, p := range r.IngressRuleValue.HTTP.Paths {
				//data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Host:https://{{%s}}::url", r.Host)})
				data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Url:{{https://%s%s}}::url", r.Host, p.Path)})
				data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Backend:{{%s}}::yellow", p.Backend.ServiceName)})
			}
		}
		loadBalancesList := "LoadBalanceIP: "
		for _, i := range ing.Status.LoadBalancer.Ingress {
			if i.IP != "" {
				loadBalancesList += cfmt.Sprintf("{{%s}}::yellow ", i.IP)
			}
			if i.Hostname != "" {
				loadBalancesList += cfmt.Sprintf("{{%s}}::yellow ", i.Hostname)
			}
		}
		data = append(data, []string{"Ingress", ing.Name, loadBalancesList})
	}
	for _, pvc := range sf.AllInfo.PvcList.Items {
		data = append(data, []string{"PVC", pvc.Name, cfmt.Sprintf("StorageClass:{{%s}}::yellow", *pvc.Spec.StorageClassName)})
		data = append(data, []string{"PVC", pvc.Name, cfmt.Sprintf("AccessModes:{{%s}}::yellow", string(pvc.Spec.AccessModes[0]))})
		pvcSize := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		data = append(data, []string{"PVC", pvc.Name, cfmt.Sprintf("Size:{{%s}}::yellow", pvcSize.String())})
		data = append(data, []string{"PV", pvc.Spec.VolumeName, ""})
	}
	for _, conf := range sf.AllInfo.ConfigMapList.Items {
		data = append(data, []string{"ConfigMap", conf.Name, ""})
	}
	for _, sec := range sf.AllInfo.SecretList.Items {
		data = append(data, []string{"Secrets", sec.Name, ""})
	}
	if sf.AllInfo.Hpa != nil {
		data = append(data, []string{"HPA", sf.AllInfo.Hpa.Name, cfmt.Sprintf("MIN:{{%d}}::yellow", *sf.AllInfo.Hpa.Spec.MinReplicas)})
		data = append(data, []string{"HPA", sf.AllInfo.Hpa.Name, cfmt.Sprintf("MAX:{{%d}}::yellow", sf.AllInfo.Hpa.Spec.MaxReplicas)})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Name", "Details"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.AppendBulk(data)
	//table.SetCaption(true, "Movie ratings.")
	table.Render()
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

	if err := sf.getOwnerByPod(); err != nil {
		return err
	}

	if err = sf.printPodLeveledList(); err != nil {
		return err
	}

	if err = sf.getLabelByPod(); err != nil {
		return err
	}

	if err = sf.findDeployByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}

	if err = sf.findStsByLabel(sf.PodObject.Namespace); err != nil {
		return err
	}

	if err = sf.findDsByLabel(sf.PodObject.Namespace); err != nil {
		return err
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

	if err = sf.findHpaByName(sf.PodObject.Namespace); err != nil {
		return err
	}

	if err = sf.printResourceTalbe(); err != nil {
		return err
	}

	return nil
}
