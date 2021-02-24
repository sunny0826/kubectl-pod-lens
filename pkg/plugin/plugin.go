package plugin

import (
	"github.com/i582/cfmt"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	autov1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

type AllInfo struct {
	SvcList        *v1.ServiceList
	IngList        *v1beta1.IngressList
	PvcList        *v1.PersistentVolumeClaimList
	ConfigMapList  *v1.ConfigMapList
	SecretList     *v1.SecretList
	Hpa            *autov1.HorizontalPodAutoscaler
	DeploymentName string
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
		return errors.New("Failed to get pod: check your parameters, set a context or verify API server.")
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
	} else {
		pterm.Println(pterm.Cyan("Failed to get other, These labels do not exist: ") +
			pterm.Green("release") +
			pterm.Cyan(",") +
			pterm.Green("app") +
			pterm.Cyan(",") +
			pterm.Green("k8s-app"))
		os.Exit(1)
	}
	sf.LabelSelector = labelSelector
	return nil
}

func (sf *SnifferPlugin) findSvcByLabel(namespace string) error {
	svcFind, err := sf.Clientset.CoreV1().Services(namespace).List(metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SvcList = svcFind
	return nil
}

func (sf *SnifferPlugin) findIngressByLabel(namespace string) error {
	ingFind, err := sf.Clientset.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.IngList = ingFind
	return nil
}

func (sf *SnifferPlugin) findPVCByLabel(namespace string) error {
	pvcFind, err := sf.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.PvcList = pvcFind
	return nil
}

func (sf *SnifferPlugin) findConfigMapByLabel(namespace string) error {
	configMapFind, err := sf.Clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.ConfigMapList = configMapFind
	return nil
}

func (sf *SnifferPlugin) findSecretByLabel(namespace string) error {
	secretFind, err := sf.Clientset.CoreV1().Secrets(namespace).List(metav1.ListOptions{LabelSelector: sf.LabelSelector})
	if err != nil {
		return err
	}
	sf.AllInfo.SecretList = secretFind
	return nil
}

func (sf *SnifferPlugin) findDeployByRs() error {

	return nil
}

func (sf *SnifferPlugin) findHpaByName(namespace string) error {
	hpaFind, err := sf.Clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, hpa := range hpaFind.Items {
		if hpa.Spec.ScaleTargetRef.Name == sf.AllInfo.DeploymentName {
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

	leveledList = append(leveledList, pterm.LeveledListItem{Level: 0, Text: sf.PodObject.Spec.NodeName})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 1, Text: sf.PodObject.Namespace})
	for _, existingOwnerRef := range sf.PodObject.GetOwnerReferences() {
		ownerKind := existingOwnerRef.Kind
		if ownerKind == "ReplicaSet" {
			ownerKind = "Deployment"
			rsObject, err := sf.Clientset.AppsV1().ReplicaSets(
				sf.PodObject.GetNamespace()).Get(
				existingOwnerRef.Name,
				metav1.GetOptions{})
			if err != nil {
				return errors.New("Failed to retrieve replica set data, AppsV1 API was not available.")
			}
			for _, rsOwner := range rsObject.GetOwnerReferences() {
				sf.AllInfo.DeploymentName = rsOwner.Name
			}
		}
		workloadName := existingOwnerRef.Name
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 2, Text: ownerKind})
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 3, Text: workloadName})
	}
	if sf.PodObject.Status.Phase != "Running" {
		cfmt.RegisterStyle("pod", func(s string) string {
			return cfmt.Sprintf("{{%s}}::red|bold", s)
		})
	}
	podInfo := cfmt.Sprintf("{{%s}}::lightBlue|bold {{[%s]}}::pod", sf.PodObject.Name, sf.PodObject.Status.Phase)
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 4, Text: podInfo})
	for _, val := range sf.PodObject.Status.InitContainerStatuses {
		if val.State.Terminated.Reason != "Completed" {
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}
		initInfo := cfmt.Sprintf("%s {{init}}::gray|bold {{[%s]}}::pod restart: %d", val.Name, val.State.Terminated.Reason, val.RestartCount)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 5, Text: initInfo})
	}
	for _, val := range sf.PodObject.Status.ContainerStatuses {
		state := "Ready"
		if !val.Ready {
			state = "Not Ready"
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}
		containerInfo := cfmt.Sprintf("%s {{[%s]}}::pod restart: %d", val.Name, state, val.RestartCount)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 5, Text: containerInfo})
	}

	root := pterm.NewTreeFromLeveledList(leveledList)

	level, _ := pterm.DefaultTree.WithRoot(root).Srender()
	panels := pterm.Panels{
		{{Data: "[node]\n[namespace]\n[type]\n[workload]\n[pod]\n[containers]"}, {Data: level}},
	}

	err := pterm.DefaultPanel.WithPanels(panels).WithPadding(0).Render()
	if err != nil {
		return err
	}
	return nil
}

func (sf *SnifferPlugin) printResourceTalbe() error {
	var data [][]string

	cfmt.RegisterStyle("url", func(s string) string {
		return cfmt.Sprintf("{{%s}}::yellow|underline", s)
	})

	for _, svc := range sf.AllInfo.SvcList.Items {
		if svc.Spec.ClusterIP != "None" {
			detail := cfmt.Sprintf("ClusterIP:{{%s}}::yellow", svc.Spec.ClusterIP)
			data = append(data, []string{"Service", svc.Name, detail})
		}
		var ports string
		for _, v := range svc.Spec.Ports {
			ports = cfmt.Sprintf("Name:{{%s}}::yellow\nPort:{{%d}}::yellow\nTargetPort:{{%d}}::yellow", v.Name, v.Port, v.TargetPort.IntVal)
			data = append(data, []string{"Service", svc.Name, ports})
		}
	}
	for _, ing := range sf.AllInfo.IngList.Items {
		for _, r := range ing.Spec.Rules {
			data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Host:{{%s}}::url", r.Host)})
			for _, p := range r.IngressRuleValue.HTTP.Paths {
				data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Path:{{%s}}::url", p.Path)})
				data = append(data, []string{"Ingress", ing.Name, cfmt.Sprintf("Backend:{{%s}}::yellow", p.Backend.ServiceName)})
			}
		}
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
	//table.SetFooter([]string{"", "", "Total", "$146.93"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.SetCaption(true, "Movie ratings.")
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

	if err = sf.printPodLeveledList(); err != nil {
		return err
	}

	if err = sf.getLabelByPod(); err != nil {
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
