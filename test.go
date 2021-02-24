package main

import (
	"github.com/olekukonko/tablewriter"
	"github.com/pterm/pterm"
	"os"
)

func main() {
	leveledList := pterm.LeveledList{
		pterm.LeveledListItem{Level: 0, Text: "ip-172-25-200-189.cn-northwest-1.compute.internal [ready]"},
		pterm.LeveledListItem{Level: 1, Text: "data-warehouse"},
		pterm.LeveledListItem{Level: 2, Text: "statefulset"},
		pterm.LeveledListItem{Level: 3, Text: "airflow-beta-redis-master [1 replica]"},
		pterm.LeveledListItem{Level: 4, Text: "airflow-beta-redis-master-0 [running]"},
		pterm.LeveledListItem{Level: 5, Text: "airflow-beta-redis [0 restarts]"},
		pterm.LeveledListItem{Level: 5, Text: "volume-permissions [init, 0 restarts]"},
	}

	// Generate tree from LeveledList.
	root := pterm.NewTreeFromLeveledList(leveledList)

	// Render TreePrinter
	a, _ := pterm.DefaultTree.WithRoot(root).Srender()
	// Declare panels in a two dimensional grid system.
	panels := pterm.Panels{
		{{Data: "[node]\n[namespace]\n[type]\n[workload]\n[pod]\n[containers]"}, {Data: a}},
	}

	_ = pterm.DefaultPanel.WithPanels(panels).WithPadding(0).Render()
	//t,_:=pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
	//	{Level: 0, Text: "configmap", TextStyle: pterm.NewStyle(pterm.FgGreen), BulletStyle: pterm.NewStyle(pterm.FgRed)},
	//	{Level: 1, Text: "configmap-beta-redis-master", TextStyle: pterm.NewStyle(pterm.FgCyan), Bullet: ">", BulletStyle: pterm.NewStyle(pterm.FgYellow)},
	//}).Srender()
	//pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(15).Print(
	//	"Related Resources")
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  " Ingress ",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  " Service ",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  "   HPA   ",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  "   PVC   ",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  "ConfigMap",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)
	//pterm.Info.Prefix = pterm.Prefix{
	//	Text:  " Secrets ",
	//	Style: pterm.NewStyle(pterm.BgLightBlue, pterm.FgLightWhite),
	//}
	//pterm.Info.Print(t)

	data := [][]string{
		[]string{"configmap", "Domain name", "1234", "$10.98"},
		[]string{"configmap", "January Hosting", "2345", "$54.95"},
		[]string{"Secrets", "February Hosting", "3456", "$51.00"},
		[]string{"Secrets", "February Extra Bandwidth", "4567", "$30.00"},
		[]string{"Service", "February Extra Bandwidth1", "4567312", "$30.00"},
		[]string{"Ingress", "February Extra Bandwidth2", "45673123123", "$30.00"},
		[]string{"HPA", "February Extra Bandwidth3", "456731231", "$30.00"},
		[]string{"PVC", "February Extra Bandwidth4", "456713231", "$30.00"},
		[]string{"PV", "February Extra Bandwidth213", "456713212", "$30.00"},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Name", "Description", "Details"})
	//table.SetFooter([]string{"", "", "Total", "$146.93"})
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.SetCaption(true, "Movie ratings.")
	table.Render()
}
