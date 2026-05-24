package output

// Content type name constants used by ContentType.String and renderers.
const (
	contentTypeNameRaw     = "raw"
	contentTypeNameSection = "section"
)

// File extension constants that do not map directly to a format name.
const (
	extYML = "yml"
)

// Transformer name constants.
const (
	transformerNameColor = "color"
)

// Truncation indicator shared by collapsible rendering.
const (
	truncateIndicatorText = "[...truncated]"
)

// HTML document metadata defaults.
const (
	htmlCharsetUTF8 = "UTF-8"
	htmlViewport    = "width=device-width, initial-scale=1.0"
)

// Field key constants used when building generic JSON/YAML representations
// and CSV/collapsible output. These values are part of the serialization
// contract, so they are defined once and reused.
const (
	keyType     = "type"
	keyContent  = "content"
	keyFormat   = "format"
	keyData     = "data"
	keyKeys     = "keys"
	keyFields   = "fields"
	keySummary  = "summary"
	keyDetails  = "details"
	keyExpanded = "expanded"
	keyTitle    = "title"
	keyLevel    = "level"
	keyName     = "name"
	keyHidden   = "hidden"
	keyBold     = "bold"
	keyItalic   = "italic"
	keyColor    = "color"
	keySize     = "size"
	keyHeader   = "header"
)
