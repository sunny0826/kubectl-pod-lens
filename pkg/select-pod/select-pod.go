package select_pod

import (
	"fmt"
	"github.com/manifoldco/promptui"
	v1 "k8s.io/api/core/v1"
	"strings"
)

func MatchPods(pods *v1.PodList, poaName string) (v1.PodList, error) {
	var result v1.PodList

	for _, pod := range pods.Items {
		if strings.Contains(pod.GetName(), poaName) {
			result.Items = append(result.Items, pod)
		}
	}

	if len(result.Items) == 0 {
		err := fmt.Errorf("no pods found for filter: %s", poaName)
		return result, err
	}

	return result, nil
}

func SelectPod(pods []v1.Pod) (v1.Pod, error) {
	if len(pods) == 1 {
		return pods[0], nil
	}

	podsPrompt := promptui.Select{
		Label:     "Select Pod",
		Items:     pods,
		Templates: podTemplate,
		IsVimMode: false,
	}

	i, _, err := podsPrompt.Run()
	if err != nil {
		return pods[i], err
	}

	return pods[i], nil
}
