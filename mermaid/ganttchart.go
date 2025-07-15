package mermaid

import (
	"fmt"
	"strings"
)

// Ganttchart represents a mermaid gantt chart
type Ganttchart struct {
	Settings   *Settings
	Title      string
	DateFormat string
	AxisFormat string
	Sections   map[string]*GanttchartSection
}

// GanttchartSection represents a section in a gantt chart
type GanttchartSection struct {
	Title string
	Tasks []GanttchartTask
}

// GanttchartTask represents a task in a gantt chart section
type GanttchartTask struct {
	Title     string
	StartDate string
	Duration  string
	Status    string
}

// NewGanttchart creates a new gantt chart with default settings
func NewGanttchart(settings *Settings) *Ganttchart {
	defaultSection := make(map[string]*GanttchartSection)
	defaultSection["defaultmermaidsection"] = &GanttchartSection{}

	return &Ganttchart{
		DateFormat: "HH:mm:ss",
		AxisFormat: "%H:%M:%S",
		Settings:   settings,
		Sections:   defaultSection,
	}
}

// GetDefaultSection returns the default section of the gantt chart
func (chart *Ganttchart) GetDefaultSection() *GanttchartSection {
	return chart.Sections["defaultmermaidsection"]
}

// AddTask adds a task to the gantt chart section
func (section *GanttchartSection) AddTask(title string, startdate string, duration string, status string) {
	newtask := GanttchartTask{
		Title:     title,
		StartDate: startdate,
		Duration:  duration,
		Status:    status,
	}
	section.Tasks = append(section.Tasks, newtask)
}

// RenderString renders the gantt chart as a mermaid string
func (chart *Ganttchart) RenderString() string {
	titleText := ""
	if chart.Title != "" {
		titleText = fmt.Sprintf("\ttitle %s\n", chart.Title)
	}
	axisformat := ""
	if chart.AxisFormat != "" {
		axisformat = fmt.Sprintf("\taxisFormat %s\n", chart.AxisFormat)
	}
	sections := make([]string, 0)
	for _, section := range chart.Sections {
		sections = append(sections, section.toString())
	}

	result := fmt.Sprintf("%sgantt\n%s\tdateFormat %s\n%s%s%s\n",
		chart.Settings.Header(),
		titleText,
		chart.DateFormat,
		axisformat,
		strings.Join(sections, "\n"),
		chart.Settings.Footer(),
	)
	return result
}

func (section *GanttchartSection) toString() string {
	result := make([]string, 0)
	if section.Title != "" {
		result = append(result, fmt.Sprintf("\tsection %s", section.Title))
	}
	for _, task := range section.Tasks {
		result = append(result, task.toString())
	}
	return strings.Join(result, "\n")
}

func (task *GanttchartTask) toString() string {
	status := ""
	if task.Status != "" {
		status = fmt.Sprintf("%s, ", task.Status)
	}
	return fmt.Sprintf("\t%s\t:%s%s , %s",
		task.Title,
		status,
		task.StartDate,
		task.Duration,
	)
}
