# Error Handling and Validation - Design Document

## 1. Architecture Overview

### 1.1 Error System Architecture

```
┌─────────────────────────────────────────────┐
│           Error Management System            │
├─────────────────┬─────────────┬────────────┤
│   Validation    │    Error    │  Recovery  │
│   Framework     │   Registry  │   Engine   │
├─────────────────┼─────────────┼────────────┤
│  Validators     │  Error Types │  Handlers  │
│  Rules Engine   │  Formatters  │  Strategies│
│  Constraints    │  Context     │  Fallbacks │
└─────────────────┴─────────────┴────────────┘
```

### 1.2 Core Components

#### 1.2.1 Error Interface Hierarchy
```go
// OutputError is the base error interface
type OutputError interface {
    error
    Code() ErrorCode
    Severity() ErrorSeverity
    Context() ErrorContext
    Suggestions() []string
    Wrap(error) OutputError
}

// ValidationError for validation failures
type ValidationError interface {
    OutputError
    Violations() []Violation
    IsComposite() bool
}

// ProcessingError for runtime failures
type ProcessingError interface {
    OutputError
    Retryable() bool
    PartialResult() interface{}
}
```

#### 1.2.2 Error Types
```go
type ErrorCode string

const (
    // Configuration errors (1xxx)
    ErrInvalidFormat      ErrorCode = "OUT-1001"
    ErrMissingRequired    ErrorCode = "OUT-1002"
    ErrIncompatibleConfig ErrorCode = "OUT-1003"
    ErrInvalidFilePath    ErrorCode = "OUT-1004"
    
    // Validation errors (2xxx)
    ErrMissingColumn      ErrorCode = "OUT-2001"
    ErrInvalidDataType    ErrorCode = "OUT-2002"
    ErrConstraintViolation ErrorCode = "OUT-2003"
    ErrEmptyDataset       ErrorCode = "OUT-2004"
    
    // Processing errors (3xxx)
    ErrFileWrite          ErrorCode = "OUT-3001"
    ErrS3Upload           ErrorCode = "OUT-3002"
    ErrTemplateRender     ErrorCode = "OUT-3003"
    ErrMemoryExhausted    ErrorCode = "OUT-3004"
)

type ErrorSeverity int

const (
    SeverityFatal ErrorSeverity = iota
    SeverityError
    SeverityWarning
    SeverityInfo
)
```

## 2. Detailed Design

### 2.1 Error Implementation

#### 2.1.1 Base Error Structure
```go
type baseError struct {
    code        ErrorCode
    severity    ErrorSeverity
    message     string
    context     ErrorContext
    suggestions []string
    cause       error
}

type ErrorContext struct {
    Operation   string
    Field       string
    Value       interface{}
    Index       int
    Metadata    map[string]interface{}
}

func (e *baseError) Error() string {
    var b strings.Builder
    fmt.Fprintf(&b, "[%s] %s", e.code, e.message)
    
    if e.context.Field != "" {
        fmt.Fprintf(&b, " (field: %s)", e.context.Field)
    }
    
    if len(e.suggestions) > 0 {
        fmt.Fprintf(&b, "\nSuggestions:\n")
        for _, s := range e.suggestions {
            fmt.Fprintf(&b, "  - %s\n", s)
        }
    }
    
    if e.cause != nil {
        fmt.Fprintf(&b, "\nCaused by: %v", e.cause)
    }
    
    return b.String()
}
```

#### 2.1.2 Validation Error Implementation
```go
type validationError struct {
    baseError
    violations []Violation
}

type Violation struct {
    Field       string
    Value       interface{}
    Constraint  string
    Message     string
}

func (v *validationError) Error() string {
    var b strings.Builder
    b.WriteString(v.baseError.Error())
    
    if len(v.violations) > 0 {
        b.WriteString("\nValidation violations:\n")
        for _, violation := range v.violations {
            fmt.Fprintf(&b, "  - %s: %s (value: %v)\n", 
                violation.Field, violation.Message, violation.Value)
        }
    }
    
    return b.String()
}
```

### 2.2 Validation Framework

#### 2.2.1 Validator Interface
```go
type Validator interface {
    Validate(subject interface{}) error
    Name() string
}

type ValidatorFunc func(interface{}) error

func (f ValidatorFunc) Validate(subject interface{}) error {
    return f(subject)
}
```

#### 2.2.2 Built-in Validators
```go
// RequiredColumnsValidator validates required columns exist
type RequiredColumnsValidator struct {
    columns []string
}

func (v *RequiredColumnsValidator) Validate(subject interface{}) error {
    output, ok := subject.(*OutputArray)
    if !ok {
        return NewValidationError(ErrInvalidDataType, "expected OutputArray")
    }
    
    missing := []string{}
    for _, required := range v.columns {
        found := false
        for _, key := range output.Keys {
            if key == required {
                found = true
                break
            }
        }
        if !found {
            missing = append(missing, required)
        }
    }
    
    if len(missing) > 0 {
        return NewValidationError(ErrMissingColumn, 
            "missing required columns").
            WithViolations(missing...)
    }
    
    return nil
}

// DataTypeValidator validates column data types
type DataTypeValidator struct {
    columnTypes map[string]reflect.Type
}

// ConstraintValidator for custom constraints
type ConstraintValidator struct {
    constraints []Constraint
}

type Constraint interface {
    Check(row map[string]interface{}) error
    Description() string
}
```

### 2.3 Error Handling Integration

#### 2.3.1 OutputArray Changes
```go
type OutputArray struct {
    Settings      *OutputSettings
    Contents      []OutputHolder
    Keys          []string
    validators    []Validator
    errorHandler  ErrorHandler
}

// Validate runs all validators
func (o *OutputArray) Validate() error {
    // Settings validation
    if err := o.Settings.Validate(); err != nil {
        return err
    }
    
    // Format-specific validation
    if err := o.validateForFormat(); err != nil {
        return err
    }
    
    // Data validation
    for _, validator := range o.validators {
        if err := validator.Validate(o); err != nil {
            return o.handleError(err)
        }
    }
    
    return nil
}

// Write with error handling
func (o *OutputArray) Write() error {
    // Validate first
    if err := o.Validate(); err != nil {
        return err
    }
    
    // Stop any active progress
    stopActiveProgress()
    
    // Generate output
    result, err := o.generate()
    if err != nil {
        return o.handleError(err)
    }
    
    // Write output
    if err := o.writeOutput(result); err != nil {
        return o.handleError(err)
    }
    
    return nil
}

// AddValidator adds a custom validator
func (o *OutputArray) AddValidator(v Validator) {
    o.validators = append(o.validators, v)
}

// WithErrorHandler sets a custom error handler
func (o *OutputArray) WithErrorHandler(h ErrorHandler) *OutputArray {
    o.errorHandler = h
    return o
}
```

#### 2.3.2 OutputSettings Validation
```go
func (s *OutputSettings) Validate() error {
    errors := NewCompositeError()
    
    // Validate output format
    if !s.isValidFormat() {
        errors.Add(NewValidationError(
            ErrInvalidFormat,
            fmt.Sprintf("invalid output format: %s", s.OutputFormat),
        ).WithSuggestions(
            fmt.Sprintf("Valid formats: %s", strings.Join(validFormats, ", ")),
        ))
    }
    
    // Validate format-specific requirements
    switch s.OutputFormat {
    case "mermaid":
        if s.FromToColumns == nil && s.MermaidSettings == nil {
            errors.Add(NewConfigError(
                ErrMissingRequired,
                "mermaid format requires FromToColumns or MermaidSettings",
            ).WithSuggestions(
                "Use AddFromToColumns() to set source and target columns",
                "Or configure MermaidSettings for chart generation",
            ))
        }
    case "dot":
        if s.FromToColumns == nil {
            errors.Add(NewConfigError(
                ErrMissingRequired,
                "dot format requires FromToColumns configuration",
            ))
        }
    }
    
    // Validate file output
    if s.OutputFile != "" {
        if err := s.validateOutputFile(); err != nil {
            errors.Add(err)
        }
    }
    
    // Validate S3 configuration
    if s.S3Bucket.Bucket != "" {
        if err := s.validateS3Config(); err != nil {
            errors.Add(err)
        }
    }
    
    return errors.ErrorOrNil()
}
```

### 2.4 Error Handlers

#### 2.4.1 Error Handler Interface
```go
type ErrorHandler interface {
    HandleError(err error) error
    SetMode(mode ErrorMode)
}

type ErrorMode int

const (
    ErrorModeStrict ErrorMode = iota
    ErrorModeLenient
    ErrorModeInteractive
)
```

#### 2.4.2 Default Error Handler
```go
type DefaultErrorHandler struct {
    mode            ErrorMode
    errors          []error
    warningHandler  func(error)
    recoveryHandler RecoveryHandler
}

func (h *DefaultErrorHandler) HandleError(err error) error {
    outputErr, ok := err.(OutputError)
    if !ok {
        outputErr = WrapError(err)
    }
    
    switch h.mode {
    case ErrorModeStrict:
        return h.handleStrict(outputErr)
    case ErrorModeLenient:
        return h.handleLenient(outputErr)
    case ErrorModeInteractive:
        return h.handleInteractive(outputErr)
    }
    
    return err
}

func (h *DefaultErrorHandler) handleStrict(err OutputError) error {
    if err.Severity() >= SeverityError {
        return err
    }
    
    if h.warningHandler != nil {
        h.warningHandler(err)
    }
    
    return nil
}

func (h *DefaultErrorHandler) handleLenient(err OutputError) error {
    h.errors = append(h.errors, err)
    
    if err.Severity() == SeverityFatal {
        return err
    }
    
    // Try recovery for non-fatal errors
    if h.recoveryHandler != nil {
        if recovered := h.recoveryHandler.Recover(err); recovered != nil {
            return nil
        }
    }
    
    return nil
}
```

### 2.5 Recovery Strategies

#### 2.5.1 Recovery Handler
```go
type RecoveryHandler interface {
    Recover(err OutputError) error
    CanRecover(err OutputError) bool
}

type RecoveryStrategy interface {
    Apply(err OutputError, context interface{}) (interface{}, error)
    ApplicableFor(err OutputError) bool
}
```

#### 2.5.2 Built-in Recovery Strategies
```go
// FormatFallbackStrategy falls back to simpler formats
type FormatFallbackStrategy struct {
    fallbackChain []string // e.g., ["table", "csv", "json"]
}

func (s *FormatFallbackStrategy) Apply(err OutputError, context interface{}) (interface{}, error) {
    settings, ok := context.(*OutputSettings)
    if !ok {
        return nil, fmt.Errorf("invalid context for format fallback")
    }
    
    currentIdx := -1
    for i, format := range s.fallbackChain {
        if format == settings.OutputFormat {
            currentIdx = i
            break
        }
    }
    
    if currentIdx >= 0 && currentIdx < len(s.fallbackChain)-1 {
        settings.OutputFormat = s.fallbackChain[currentIdx+1]
        return settings, nil
    }
    
    return nil, fmt.Errorf("no fallback format available")
}

// DefaultValueStrategy provides defaults for missing values
type DefaultValueStrategy struct {
    defaults map[string]interface{}
}

// RetryStrategy for transient errors
type RetryStrategy struct {
    maxAttempts int
    backoff     BackoffStrategy
    retryable   func(error) bool
}
```

### 2.6 Backward Compatibility

#### 2.6.1 Legacy Mode
```go
// LegacyErrorHandler mimics old log.Fatal behavior
type LegacyErrorHandler struct{}

func (h *LegacyErrorHandler) HandleError(err error) error {
    if err != nil {
        log.Fatal(err)
    }
    return nil
}

// EnableLegacyMode for backward compatibility
func (o *OutputArray) EnableLegacyMode() {
    o.errorHandler = &LegacyErrorHandler{}
}
```

#### 2.6.2 Migration Helper
```go
// WriteCompat provides backward-compatible Write method
func (o *OutputArray) WriteCompat() {
    if err := o.Write(); err != nil {
        log.Fatal(err)
    }
}
```

### 2.7 Error Reporting

#### 2.7.1 Error Reporter
```go
type ErrorReporter interface {
    Report(err OutputError)
    Summary() ErrorSummary
}

type ErrorSummary struct {
    TotalErrors   int
    ByCategory    map[ErrorCode]int
    BySeverity    map[ErrorSeverity]int
    Suggestions   []string
    FixableErrors int
}
```

#### 2.7.2 Structured Error Output
```go
func (e *baseError) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Code        ErrorCode         `json:"code"`
        Severity    string           `json:"severity"`
        Message     string           `json:"message"`
        Context     ErrorContext     `json:"context,omitempty"`
        Suggestions []string         `json:"suggestions,omitempty"`
        Cause       string           `json:"cause,omitempty"`
    }{
        Code:        e.code,
        Severity:    e.severity.String(),
        Message:     e.message,
        Context:     e.context,
        Suggestions: e.suggestions,
        Cause:       e.causeString(),
    })
}
```

### 2.8 Usage Examples

#### 2.8.1 Basic Error Handling
```go
func main() {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("json")
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Value"},
    }
    
    // Add data...
    
    // New error-returning API
    if err := output.Write(); err != nil {
        // Handle error appropriately
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        
        // Check if it's an OutputError for more details
        if outErr, ok := err.(format.OutputError); ok {
            for _, suggestion := range outErr.Suggestions() {
                fmt.Fprintf(os.Stderr, "Try: %s\n", suggestion)
            }
        }
        os.Exit(1)
    }
}
```

#### 2.8.2 Advanced Validation
```go
// Custom validation
output.AddValidator(format.ValidatorFunc(func(o interface{}) error {
    out := o.(*format.OutputArray)
    for _, holder := range out.Contents {
        if val, ok := holder.Contents["Price"].(float64); ok {
            if val < 0 {
                return format.NewValidationError(
                    format.ErrConstraintViolation,
                    "negative price not allowed",
                ).WithContext(format.ErrorContext{
                    Field: "Price",
                    Value: val,
                })
            }
        }
    }
    return nil
}))

// Validate before processing
if err := output.Validate(); err != nil {
    // Handle validation errors
}
```

#### 2.8.3 Lenient Mode with Recovery
```go
settings.ErrorMode = format.ErrorModeLenient
settings.RecoveryStrategies = []format.RecoveryStrategy{
    format.NewFormatFallbackStrategy("table", "csv", "json"),
    format.NewDefaultValueStrategy(map[string]interface{}{
        "Status": "Unknown",
        "Count":  0,
    }),
}

output := format.OutputArray{Settings: settings}

// Will attempt recovery instead of failing
err := output.Write()
if err != nil {
    // Only fatal errors reach here
}

// Check for warnings/recovered errors
if summary := output.ErrorSummary(); summary.TotalErrors > 0 {
    fmt.Printf("Completed with %d errors\n", summary.TotalErrors)
}
```

## 3. Implementation Order Recommendation

### Should Error Handling be Implemented Before Transformation Pipeline?

**Yes, Error Handling should be implemented FIRST.** Here's why:

#### 3.1 Foundation for Other Features

1. **Clean API Design**: Transformation pipeline can be designed with proper error handling from the start
2. **No Technical Debt**: Avoid retrofitting error handling into transformation code
3. **Better Testing**: Can write comprehensive tests for transformations with error cases

#### 3.2 Immediate Benefits

1. **Stability**: Current users get immediate stability improvements
2. **Gradual Migration**: Users can migrate from `log.Fatal` at their pace
3. **Production Ready**: Makes the library suitable for production use cases

#### 3.3 Transformation Pipeline Benefits

1. **Error Context**: Transformations can provide rich error context
2. **Validation Pipeline**: Transform validators can use the validation framework
3. **Recovery Options**: Failed transformations can trigger recovery strategies

#### 3.4 Implementation Phases

**Phase 1: Core Error Handling (2-3 weeks)**
- Replace log.Fatal with error returns
- Implement basic error types
- Add backward compatibility mode

**Phase 2: Validation Framework (1-2 weeks)**
- Implement validator interface
- Add built-in validators
- Integrate with OutputArray

**Phase 3: Recovery Strategies (1 week)**
- Implement recovery handlers
- Add fallback strategies
- Error reporting

**Phase 4: Transformation Pipeline (3-4 weeks)**
- Build on error handling foundation
- Use validation framework
- Leverage recovery strategies

This order ensures a solid foundation and better overall architecture.