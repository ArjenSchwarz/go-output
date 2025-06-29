# mermaid package

Utilities for generating [Mermaid](https://mermaid.js.org) diagrams.
The package exposes a small set of types that turn your data into
Mermaid syntax which can then be embedded in Markdown or HTML.

Supported chart types:

- **Flowchart** - basic node and edge graphs
- **PieChart**  - pie charts with optional value labels
- **Ganttchart** - simple project timelines

All charts implement the `MermaidChart` interface which provides a
`RenderString()` method returning the Mermaid code. The [`Settings`](settings.go)
type controls whether Markdown or HTML wrappers are added around the chart.

See the [examples](../examples/basic_usage.go) and the full
[documentation](../DOCUMENTATION.md#mermaid-format) for detailed usage.
