package mermaid

import (
	"fmt"
	"strings"
)

type PieChart struct {
	Settings *Settings
	Title    string
	Values   []PieChartValue
	ShowData bool
}

type PieChartValue struct {
	Label string
	Value float64
}

func NewPiechart(settings *Settings) *PieChart {
	return &PieChart{Settings: settings}
}

func (piechart *PieChart) AddValue(label string, value float64) {
	if piechart.Values == nil {
		piechart.Values = make([]PieChartValue, 0)
	}
	node := PieChartValue{
		Label: label,
		Value: value,
	}
	piechart.Values = append(piechart.Values, node)
}

func (piechart *PieChart) RenderString() string {
	showdataText := ""
	if piechart.ShowData {
		showdataText = "showData"
	}
	titleText := ""
	if piechart.Title != "" {
		titleText = fmt.Sprintf("title %s\n", piechart.Title)
	}
	result := fmt.Sprintf("%spie %s\n%s%s\n%s",
		piechart.Settings.MarkdownHeader(),
		showdataText,
		titleText,
		piechart.getValuesString(),
		piechart.Settings.MarkdownFooter(),
	)
	return result
}

func (piechart *PieChart) getValuesString() string {
	result := make([]string, 0)
	for _, node := range piechart.Values {
		result = append(result, fmt.Sprintf("\t\"%s\" : %.2f", node.Label, node.Value))
	}
	return strings.Join(result, "\n")
}
