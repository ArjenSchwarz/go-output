package mermaid

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Flowchart represents a Mermaid flowchart diagram
type Flowchart struct {
	Direction string
	Nodes     map[string]*Node
	Edges     []*Edge
	Settings  *Settings
}

// Node represents a node in a flowchart
type Node struct {
	id   string
	name string
}

// Edge represents a connection between two nodes in a flowchart
type Edge struct {
	From *Node // Pointer to the Node where the Edge starts.
	To   *Node // Pointer to the Node where the Edge ends.
}

// NewFlowchart creates a new flowchart with the specified settings
func NewFlowchart(settings *Settings) *Flowchart {
	return &Flowchart{Direction: "TB", Settings: settings}
}

// AddBasicNode adds a basic node to the flowchart
func (flowchart *Flowchart) AddBasicNode(name string) {
	if flowchart.Nodes == nil {
		flowchart.Nodes = make(map[string]*Node)
	}
	id := fmt.Sprintf("n%s", strconv.Itoa(len(flowchart.Nodes)+1))
	node := Node{
		id:   id,
		name: name,
	}
	flowchart.Nodes[name] = &node
}

// AddEdgeByNames adds an edge between two nodes specified by their names
func (flowchart *Flowchart) AddEdgeByNames(from string, to string) {
	var edges []*Edge
	if flowchart.Edges != nil {
		edges = flowchart.Edges
	}
	fromNode := flowchart.Nodes[from]
	toNode := flowchart.Nodes[to]
	edge := Edge{
		From: fromNode,
		To:   toNode,
	}
	edges = append(edges, &edge)
	flowchart.Edges = edges
}

// RenderString returns the Mermaid syntax representation of the flowchart
func (flowchart *Flowchart) RenderString() string {
	result := fmt.Sprintf("flowchart %s\n%s\n%s",
		flowchart.Direction,
		flowchart.getNodeString(),
		flowchart.getEdgeString(),
	)
	return result
}

func (flowchart *Flowchart) getNodeString() string {
	result := make([]string, 0)
	for _, node := range flowchart.Nodes {
		result = append(result, fmt.Sprintf("\t%s(\"%s\")", node.id, node.name))
	}
	sort.Strings(result)
	return strings.Join(result, "\n")
}

func (flowchart *Flowchart) getEdgeString() string {
	result := make([]string, 0)
	for _, edge := range flowchart.Edges {
		result = append(result, fmt.Sprintf("\t%s --> %s", edge.From.id, edge.To.id))
	}
	sort.Strings(result)
	return strings.Join(result, "\n")
}
