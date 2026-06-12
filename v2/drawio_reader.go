package output

// drawioConnectionJSON pins the draw.io CSV wire format for `# connect:`
// directive values (Decision 14): keys from/to/invert/label/style in that
// order, all always present. The public DrawIOConnection stays untagged so
// JSON/YAML document renderers keep their exported-field key casing; this
// private mirror confines the wire format to the drawio CSV writer and
// parser.
type drawioConnectionJSON struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Invert bool   `json:"invert"` // no omitempty: "invert":false is load-bearing for byte identity (requirement 3.5)
	Label  string `json:"label"`
	Style  string `json:"style"`
}
