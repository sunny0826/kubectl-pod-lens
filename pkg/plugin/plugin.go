package plugin

import (
	"fmt"
	"regexp"
	"strings"

	select_pod "github.com/sunny0826/kubectl-pod-lens/pkg/select-pod"
	"k8s.io/klog"

	mapset "github.com/deckarep/golang-set"
	"github.com/gosuri/uitable"

	"github.com/i582/cfmt"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	appsv1 "k8s.io/api/apps/v1"
	autov1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Workload struct {
	Type     string
	Name     string
	Replicas string
	Status   bool
}

type AllInfo struct {
	Node          *v1.Node
	DeployList    *appsv1.DeploymentList
	StsList       *appsv1.StatefulSetList
	DsList        *appsv1.DaemonSetList
	SvcList       *v1.ServiceList
	IngList       *v1beta1.IngressList
	PvcList       *v1.PersistentVolumeClaimList
	ConfigMapList *v1.ConfigMapList
	SecretList    *v1.SecretList
	Hpa           *autov1.HorizontalPodAutoscaler
	Pdbs          []*policyv1beta1.PodDisruptionBudget
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

func (sf *SnifferPlugin) findPodByName(name, namespace string) error {
	pods, err := sf.Clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil || len(pods.Items) == 0 {
		return errors.New("Failed to get pod: [" +
			name + "], please check your parameters, set a context or verify API server.")
	}

	podList, err := select_pod.MatchPods(pods, name)
	if err != nil {
		return err
	}

	podObj, err := select_pod.SelectPod(podList.Items)
	if err != nil {
		return err
	}

	sf.PodObject = &podObj
	if sf.PodObject.Spec.NodeName == "" || sf.PodObject == nil {
		return errors.New("Pod is not assigned to a node yet, it's still pending scheduling probably.")
	}
	return nil
}

func (sf *SnifferPlugin) findNodeByName() error {
	nodeObject, err := sf.Clientset.CoreV1().Nodes().Get(sf.PodObject.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return errors.New("Failed to get nodes info, verify the connection to their pool.")
	}

	sf.AllInfo.Node = nodeObject
	return nil
}

func (sf *SnifferPlugin) getLabelByPod(labelFlag string) error {
	if labelFlag != "" {
		match, err := regexp.MatchString("[a-z0-9/-]+=([a-z0-9/-]+)( |$)", labelFlag)
		if err != nil {
			return err
		}
		if match {
			sf.LabelSelector = labelFlag
			return nil
		} else {
			return errors.New(labelFlag + " is incorrectly formatted.")
		}
	}
	var labelSelector string
	labels := sf.PodObject.Labels
	if _, ok := labels["app"]; ok {
		labelSelector = "app=" + labels["app"]
	} else if _, ok = labels["release"]; ok {
		labelSelector = "release=" + labels["release"]
	} else if _, ok = labels["k8s-app"]; ok {
		labelSelector = "k8s-app=" + labels["k8s-app"]
	} else if _, ok = labels["app.kubernetes.io/name"]; ok {
		labelSelector = "app.kubernetes.io/name=" + labels["app.kubernetes.io/name"]
	} else {
		_, _ = cfmt.Println("Failed to get other, These labels do not exist:" +
			" {{[release]}}::green" +
			" {{[app]}}::green" +
			" {{[k8s-app]}}::green" +
			" {{[app.kubernetes.io/name]}}::green. So no related resources could be found.")
		//os.Exit(1)
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
		var status bool
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
				if rsObject.Status.ReadyReplicas != rsObject.Status.Replicas {
					status = true
				}
				sf.AllInfo.Workload = Workload{
					Name:     own.Name,
					Type:     own.Kind,
					Replicas: fmt.Sprintf("%d/%d", rsObject.Status.ReadyReplicas, rsObject.Status.Replicas),
					Status:   status,
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
			if ssObject.Status.ReadyReplicas != ssObject.Status.Replicas {
				status = true
			}
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: fmt.Sprintf("%d/%d", ssObject.Status.ReadyReplicas, ssObject.Status.Replicas),
				Status:   status,
			}
		case "daemonset":
			dsObject, err := sf.Clientset.AppsV1().DaemonSets(
				sf.PodObject.GetNamespace()).Get(
				existingOwnerRef.Name,
				metav1.GetOptions{})
			if err != nil {
				return errors.New("Failed to retrieve daemon set data, AppsV1 API was not available.")
			}
			if dsObject.Status.NumberReady != dsObject.Status.DesiredNumberScheduled {
				status = true
			}
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: fmt.Sprintf("%d/%d", dsObject.Status.NumberReady, dsObject.Status.DesiredNumberScheduled),
				Status:   status,
			}
		default:
			sf.AllInfo.Workload = Workload{
				Name:     existingOwnerRef.Name,
				Type:     ownerKind,
				Replicas: "0",
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

func (sf *SnifferPlugin) findPdbByName(namespace string) error {

	pdbFind, err := sf.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).List(
		metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pdb := range pdbFind.Items {
		selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if err != nil {
			return err
		}
		if selector.Empty() || selector.Matches(labels.Set(sf.PodObject.Labels)) {
			sf.AllInfo.Pdbs = append(sf.AllInfo.Pdbs, &pdb)
		}
	}
	return nil
}

func (sf *SnifferPlugin) printPodLeveledList() error {
	var leveledList pterm.LeveledList
	var stateList string
	cfmt.RegisterStyle("pod", func(s string) string {
		return cfmt.Sprintf("{{%s}}::green|bold", s)
	})
	cfmt.RegisterStyle("restart", func(s string) string {
		return cfmt.Sprintf("{{%s}}::green|bold", s)
	})
	cfmt.RegisterStyle("replica", func(s string) string {
		return cfmt.Sprintf("{{%s}}::green|bold", s)
	})
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 0,
		Text: cfmt.Sprintf("{{ [Namespace] }}::cyan|bold %s", sf.PodObject.Namespace)})
	stateList += "\n"
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 1,
		Text: cfmt.Sprintf("{{ [%s] }}::lightBlue|bold %s",
			sf.AllInfo.Workload.Type, sf.AllInfo.Workload.Name)})
	if sf.AllInfo.Workload.Status {
		cfmt.RegisterStyle("replica", func(s string) string {
			return cfmt.Sprintf("{{%s}}::red|bold", s)
		})
	}
	stateList += cfmt.Sprintf("Replica: {{%s}}::replica\n",
		sf.AllInfo.Workload.Replicas)
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 2,
		Text: cfmt.Sprintf("{{ [Node] }}::magenta|bold %s", sf.PodObject.Spec.NodeName)})
	var nodeIp string
	for _, ip := range sf.AllInfo.Node.Status.Addresses {
		if ip.Type == "InternalIP" {
			nodeIp = cfmt.Sprintf("Node IP: {{%s}}::magenta", ip.Address)
		}
	}
	var nodeStatus v1.NodeConditionType = "Ready"
	for _, s := range sf.AllInfo.Node.Status.Conditions {
		if s.Status == "True" && s.Type != "Ready" {
			nodeStatus = s.Type
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}
	}
	stateList += cfmt.Sprintf("{{[%s]}}::pod %s\n", nodeStatus, nodeIp)
	if sf.PodObject.Status.Phase != "Running" && sf.PodObject.Status.Phase != "Succeeded" {
		cfmt.RegisterStyle("pod", func(s string) string {
			return cfmt.Sprintf("{{%s}}::red|bold", s)
		})
	}
	podInfo := cfmt.Sprintf("{{ [Pod] }}::blue|bold %s", sf.PodObject.Name)
	leveledList = append(leveledList, pterm.LeveledListItem{Level: 3, Text: podInfo})
	stateList += cfmt.Sprintf("{{[%s]}}::pod Pod IP: {{%s}}::magenta\n",
		sf.PodObject.Status.Phase, sf.PodObject.Status.PodIP)
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
		initInfo := cfmt.Sprintf("{{ [initContainer] }}::gray|bold %s", val.Name)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 4, Text: initInfo})
		stateList += cfmt.Sprintf("{{[%s]}}::pod Restart: {{%d}}::restart\n",
			val.State.Terminated.Reason, val.RestartCount)
	}
	for _, val := range sf.PodObject.Status.ContainerStatuses {
		state := "Running"
		if val.State.Terminated != nil {
			state = val.State.Terminated.Reason
			if state != "Completed" {
				cfmt.RegisterStyle("pod", func(s string) string {
					return cfmt.Sprintf("{{%s}}::red|bold", s)
				})
			}
		} else if val.State.Waiting != nil {
			state = val.State.Waiting.Reason
			cfmt.RegisterStyle("pod", func(s string) string {
				return cfmt.Sprintf("{{%s}}::red|bold", s)
			})
		}

		if val.RestartCount != 0 {
			cfmt.RegisterStyle("restart", func(s string) string {
				return cfmt.Sprintf("{{%s}}::yellow|bold", s)
			})
		}
		containerInfo := cfmt.Sprintf("{{ [Container] }}::lightGreen|bold %s", val.Name)
		leveledList = append(leveledList, pterm.LeveledListItem{Level: 4, Text: containerInfo})
		stateList += cfmt.Sprintf("{{[%s]}}::pod Restart: {{%d}}::restart\n",
			state, val.RestartCount)
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
	tree, _ := pterm.DefaultTree.WithRoot(root).Srender()

	panels := pterm.Panels{
		{{Data: tree}, {Data: stateList}},
	}
	_ = pterm.DefaultPanel.WithPanels(panels).WithPadding(5).Render()
	return nil
}

func (sf *SnifferPlugin) printResource() error {
	table := uitable.New()
	//table.MaxColWidth = 80
	table.Wrap = true
	table.AddRow("")
	//fmt.Println(strings.Repeat("-", 80))

	cfmt.RegisterStyle("url", func(s string) string {
		return cfmt.Sprintf("{{%s}}::yellow|underline", s)
	})
	for _, deploy := range sf.AllInfo.DeployList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{Deployment}}::lightBlue"))
		table.AddRow("Name:", deploy.Name)
		table.AddRow("Replicas:", cfmt.Sprintf("{{%d}}::yellow", deploy.Status.Replicas))
		table.AddRow("---", "---")
	}

	for _, sts := range sf.AllInfo.StsList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{StatefulSet}}::lightBlue"))
		table.AddRow("Name:", sts.Name)
		table.AddRow("Replicas:", cfmt.Sprintf("{{%d}}::yellow", sts.Status.Replicas))
		table.AddRow("---", "---")
	}

	for _, ds := range sf.AllInfo.DsList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{DaemonSet}}::lightBlue"))
		table.AddRow("Name:", ds.Name)
		table.AddRow("Replicas:", cfmt.Sprintf("{{%d}}::yellow", ds.Status.DesiredNumberScheduled))
		table.AddRow("---", "---")
	}

	for _, svc := range sf.AllInfo.SvcList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{Service}}::lightYellow"))
		table.AddRow("Name:", svc.Name)
		if svc.Spec.ClusterIP != "None" {
			table.AddRow("Cluster IP:", cfmt.Sprintf("{{%s}}::yellow", svc.Spec.ClusterIP))
		}
		var ports string
		table.AddRow("Ports", "")
		for _, v := range svc.Spec.Ports {
			if v.TargetPort.IntVal == 0 {
				ports = cfmt.Sprintf("---\nName: {{%s}}::yellow\nPort: {{%d}}::yellow\nTargetPort: {{%s}}::yellow",
					v.Name, v.Port, v.TargetPort.StrVal)
			} else {
				ports = cfmt.Sprintf("---\nName: {{%s}}::yellow\nPort: {{%d}}::yellow\nTargetPort: {{%d}}::yellow",
					v.Name, v.Port, v.TargetPort.IntVal)
			}
			table.AddRow("", ports)
		}
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				table.AddRow("IP:", cfmt.Sprintf("{{%s}}::url", ing.IP))
			}
			if ing.Hostname != "" {
				table.AddRow("Host:", cfmt.Sprintf("{{%s}}::url", ing.Hostname))
			}

		}
		table.AddRow("---", "---")
	}

	for _, ing := range sf.AllInfo.IngList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{Ingress}}::green"))
		table.AddRow("Name:", ing.Name)
		for _, r := range ing.Spec.Rules {
			for _, p := range r.IngressRuleValue.HTTP.Paths {
				table.AddRow("Url:", cfmt.Sprintf("{{https://%s%s}}::url",
					r.Host, p.Path))
				table.AddRow("Backend:", p.Backend.ServiceName)
			}
		}
		var loadBalancesList string
		for _, i := range ing.Status.LoadBalancer.Ingress {
			if i.IP != "" {
				loadBalancesList += cfmt.Sprintf("\n{{%s}}::lightGreen", i.IP)
			}
			if i.Hostname != "" {
				loadBalancesList += cfmt.Sprintf("\n{{%s}}::lightGreen", i.Hostname)
			}
		}
		table.AddRow("LoadBalance IP:", loadBalancesList)
		table.AddRow("---")
	}

	for _, pvc := range sf.AllInfo.PvcList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{PVC}}::gray"))
		table.AddRow("Name:", pvc.Name)
		table.AddRow("Storage Class:", cfmt.Sprintf("{{%s}}::lightGreen",
			*pvc.Spec.StorageClassName))
		table.AddRow("Access Modes:", cfmt.Sprintf("{{%s}}::lightGreen",
			string(pvc.Spec.AccessModes[0])))
		pvcSize := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		table.AddRow("Size:", cfmt.Sprintf("{{%s}}::lightGreen",
			pvcSize.String()))
		table.AddRow("PV Name:", pvc.Spec.VolumeName)
		table.AddRow("---", "---")
	}

	for _, conf := range sf.AllInfo.ConfigMapList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{ConfigMap}}::magenta"))
		table.AddRow("Name:", conf.Name)
		table.AddRow("---", "---")
	}

	for _, sec := range sf.AllInfo.SecretList.Items {
		table.AddRow("Kind:", cfmt.Sprintf("{{Secrets}}::red"))
		table.AddRow("Name:", sec.Name)
		table.AddRow("---", "---")
	}

	if sf.AllInfo.Hpa != nil {
		table.AddRow("Kind:", cfmt.Sprintf("{{HPA}}::cyan"))
		table.AddRow("Name:", sf.AllInfo.Hpa.Name)
		table.AddRow("MIN:", cfmt.Sprintf("{{%d}}::lightGreen",
			*sf.AllInfo.Hpa.Spec.MinReplicas))
		table.AddRow("MAX:", cfmt.Sprintf("{{%d}}::lightGreen",
			sf.AllInfo.Hpa.Spec.MaxReplicas))
		table.AddRow("---", "---")
	}

	for _, pdb := range sf.AllInfo.Pdbs {
		table.AddRow("Kind:", cfmt.Sprintf("{{PDB}}::cyan"))
		table.AddRow("Name:", pdb.Name)
		if pdb.Spec.MinAvailable != nil {
			table.AddRow("MinAvailable:", cfmt.Sprintf("{{%s}}::lightGreen",
				pdb.Spec.MinAvailable))
		}
		if pdb.Spec.MaxUnavailable != nil {
			table.AddRow("MaxAvailable:", cfmt.Sprintf("{{%s}}::lightGreen",
				pdb.Spec.MaxUnavailable))
		}
		table.AddRow("Disruptions:", cfmt.Sprintf("{{%d}}::lightGreen",
			pdb.Status.PodDisruptionsAllowed))
		table.AddRow("---", "---")
	}

	if len(table.Rows) > 1 {
		_, _ = cfmt.Println("{{ Related Resources }}::bgCyan|#ffffff")
		fmt.Println(table)
	}
	return nil
}

func RunPlugin(configFlags *genericclioptions.ConfigFlags, outputCh chan string, allNamespacesFlag bool, labelFlag string) error {
	klog.V(1).Info("start run plugins")
	sf, err := NewSnifferPlugin(configFlags)
	if err != nil {
		return err
	}

	podName := <-outputCh
	var namespace string
	if !allNamespacesFlag {
		namespace = getNamespace(configFlags)
	}

	if err = sf.findPodByName(podName, namespace); err != nil {
		return err
	}

	if err = sf.findNodeByName(); err != nil {
		return err
	}

	if err = sf.getOwnerByPod(); err != nil {
		return err
	}

	if err = sf.printPodLeveledList(); err != nil {
		return err
	}

	if err = sf.getLabelByPod(labelFlag); err != nil {
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

	if err = sf.findPdbByName(sf.PodObject.Namespace); err != nil {
		return err
	}

	if err = sf.printResource(); err != nil {
		return err
	}

	return nil
}

func getNamespace(configFlags *genericclioptions.ConfigFlags) string {
	if v := *configFlags.Namespace; v != "" {
		return v
	}
	clientConfig := configFlags.ToRawKubeConfigLoader()
	defaultNamespace, _, err := clientConfig.Namespace()
	if err != nil {
		defaultNamespace = "default"
	}
	return defaultNamespace
}
