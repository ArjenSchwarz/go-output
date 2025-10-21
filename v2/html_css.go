package output

func init() {
	// defaultResponsiveCSS contains the default responsive CSS styling for HTML templates.
	// It implements modern CSS features (flexbox, grid), CSS custom properties for theming,
	// and mobile-first responsive design with table stacking on mobile devices.
	defaultResponsiveCSS = `
:root {
  /* Color scheme - WCAG AA compliant */
  --color-primary: #2563eb;
  --color-background: #ffffff;
  --color-surface: #f9fafb;
  --color-border: #e5e7eb;
  --color-text: #111827;
  --color-text-muted: #6b7280;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-error: #ef4444;

  /* Spacing */
  --spacing-xs: 0.25rem;
  --spacing-sm: 0.5rem;
  --spacing-md: 1rem;
  --spacing-lg: 1.5rem;
  --spacing-xl: 2rem;

  /* Typography */
  --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-size-base: 16px;
  --font-size-small: 14px;
  --font-size-large: 18px;
  --line-height: 1.5;

  /* Layout */
  --border-radius: 0.375rem;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);

  /* Breakpoints (reference only, not used in CSS) */
  --breakpoint-mobile: 480px;
}

html {
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  line-height: var(--line-height);
}

body {
  background-color: var(--color-background);
  color: var(--color-text);
  margin: 0;
  padding: var(--spacing-lg);
}

/* Typography */
h1, h2, h3, h4, h5, h6 {
  margin-top: var(--spacing-lg);
  margin-bottom: var(--spacing-md);
  line-height: 1.2;
  color: var(--color-text);
}

h1 {
  font-size: 2rem;
}

h2 {
  font-size: 1.5rem;
}

h3 {
  font-size: 1.25rem;
}

h4, h5, h6 {
  font-size: 1rem;
}

p {
  margin-top: 0;
  margin-bottom: var(--spacing-md);
}

a {
  color: var(--color-primary);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

a:focus {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

/* Table base styles - mobile first (stacked layout) */
.data-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: var(--spacing-lg);
  font-size: var(--font-size-small);
}

.data-table thead {
  position: absolute;
  left: -9999px;
}

.data-table tbody {
  display: flex;
  flex-direction: column;
}

.data-table tr {
  display: block;
  margin-bottom: var(--spacing-md);
  border: 1px solid var(--color-border);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
}

.data-table td {
  display: block;
  text-align: right;
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border);
}

.data-table td:first-child {
  padding-top: var(--spacing-md);
}

.data-table td:last-child {
  border-bottom: none;
  padding-bottom: var(--spacing-md);
}

.data-table td::before {
  content: attr(data-label);
  float: left;
  font-weight: 600;
  color: var(--color-text-muted);
}

/* Mobile-specific adjustments */
@media (max-width: 480px) {
  body {
    padding: var(--spacing-md);
  }

  h1 {
    font-size: 1.5rem;
  }

  h2 {
    font-size: 1.25rem;
  }
}

/* Table desktop view - traditional layout */
@media (min-width: 481px) {
  .data-table thead {
    position: static;
    left: auto;
    background-color: var(--color-surface);
  }

  .data-table tbody {
    display: table-row-group;
  }

  .data-table tr {
    display: table-row;
    margin-bottom: 0;
    border: none;
    box-shadow: none;
  }

  .data-table td {
    display: table-cell;
    text-align: left;
    padding: var(--spacing-md);
    border-bottom: 1px solid var(--color-border);
  }

  .data-table td:last-child {
    border-bottom: 1px solid var(--color-border);
    padding-bottom: var(--spacing-md);
  }

  .data-table td::before {
    content: none;
  }

  .data-table th {
    padding: var(--spacing-md);
    text-align: left;
    font-weight: 600;
    color: var(--color-text);
    border-bottom: 2px solid var(--color-border);
  }

  .data-table tbody tr:hover {
    background-color: var(--color-surface);
  }
}

/* Sections */
.content-section {
  margin-bottom: var(--spacing-xl);
  padding: var(--spacing-lg);
  background-color: var(--color-surface);
  border-radius: var(--border-radius);
  border-left: 4px solid var(--color-primary);
}

.content-section h2 {
  margin-top: 0;
}

/* Details/Summary (collapsible sections) */
details {
  margin-bottom: var(--spacing-lg);
}

summary {
  cursor: pointer;
  padding: var(--spacing-md);
  background-color: var(--color-surface);
  border-radius: var(--border-radius);
  font-weight: 600;
  user-select: none;
}

summary:hover {
  background-color: var(--color-border);
}

summary:focus {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

details[open] summary {
  margin-bottom: var(--spacing-md);
}

details[open] > *:not(summary) {
  padding: 0 var(--spacing-md) var(--spacing-md);
}

/* Raw HTML containers */
.raw-html {
  margin: var(--spacing-md) 0;
}

/* Text content */
.text-content {
  margin-bottom: var(--spacing-lg);
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  line-height: var(--line-height);
}

/* Utility classes */
.text-muted {
  color: var(--color-text-muted);
}

.text-small {
  font-size: var(--font-size-small);
}

.mt-md {
  margin-top: var(--spacing-md);
}

.mb-md {
  margin-bottom: var(--spacing-md);
}

.p-md {
  padding: var(--spacing-md);
}
`

	// mermaidOptimizedCSS contains CSS optimizations for Mermaid diagram rendering.
	// It includes all features of defaultResponsiveCSS plus diagram-specific styling.
	mermaidOptimizedCSS = defaultResponsiveCSS + `

/* Mermaid diagram styles */
.mermaid {
  display: flex;
  justify-content: center;
  margin: var(--spacing-xl) 0;
}

.mermaid svg {
  max-width: 100%;
  height: auto;
}

/* Mobile responsiveness for mermaid diagrams */
@media (max-width: 480px) {
  .mermaid {
    margin: var(--spacing-lg) 0;
    overflow-x: auto;
    padding: var(--spacing-md) 0;
  }

  .mermaid svg {
    min-width: 100%;
  }
}
`
}
