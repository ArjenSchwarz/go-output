# v2 Code Quality Review

## Summary
This review focuses on the v2 implementation, calling out duplicated logic and naming issues that make the surface area harder to maintain.

## 1. JSON/YAML renderer duplication
The JSON and YAML renderers each reimplement the same control flow for rendering documents and content: `Render`, `RenderTo`, `renderDocument*`, `renderContent`, and the streaming helpers follow the same branching logic with only format-specific marshalling differences.【F:v2/json_yaml_renderer.go†L20-L135】【F:v2/json_yaml_renderer.go†L544-L633】 The table, text, raw, section, and collapsible helpers also repeat their structure with just the marshaling call changing.【F:v2/json_yaml_renderer.go†L137-L220】【F:v2/json_yaml_renderer.go†L662-L805】 Extracting shared helpers (for example, a format-neutral traversal that accepts serializer callbacks) would collapse hundreds of near-identical lines and make new format-specific tweaks less error-prone.

## 2. Pipeline execution branches are clones
`ExecuteWithFormatContext` and `ExecuteContext` differ only in how they pass the optional `format` string, yet both copy the full validation, timeout, loop, and stats bookkeeping logic.【F:v2/pipeline.go†L200-L403】 Similarly, `applyOperations` and `applyOperationsWithFormat` duplicate the entire operation loop, error handling, and metrics collection with minor format-aware branching added in the latter.【F:v2/pipeline.go†L405-L538】 Merging these into a shared implementation that accepts optional format hooks would eliminate the duplication and reduce the chance of the two paths drifting in behavior.

## 3. Collapsible option naming inconsistency
The collapsible value options expose a generic `WithExpanded` helper, while the section equivalent is `WithSectionExpanded` (and other section options share the `WithSection*` prefix).【F:v2/collapsible.go†L54-L120】【F:v2/collapsible_section.go†L55-L114】 The API surface would read more consistently if the value option followed the same prefix convention (e.g., `WithCollapsibleExpanded`) so that callers can distinguish value vs. section helpers at a glance.
