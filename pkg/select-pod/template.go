package select_pod

import (
	"fmt"
	"github.com/manifoldco/promptui"
)

var podTemplate = &promptui.SelectTemplates{
	Label:    "{{ . }}",
	Active:   fmt.Sprintf("%s {{ .Name | cyan }}", promptui.IconSelect),
	Inactive: "{{ .Name | magenta }}",
	Selected: fmt.Sprintf("%s {{ .Name | cyan }}", promptui.IconGood),
	Details: `
--------- Info ----------
{{ "Namespace:" | faint }}	{{ .Namespace | yellow }}
{{ "Node:" | faint }}	{{ .Spec.NodeName | yellow }}
{{if ne  .Status.Phase "Running"}}{{ "Status:" | faint }}	{{ .Status.Phase | red }}{{else}}{{ "Status:" | faint }}	{{ .Status.Phase | green }}{{end}}
{{ "Pod IP:" | faint }}	{{ .Status.PodIP | yellow }}`,
}
