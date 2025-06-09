# go-output Progress Indicator Implementation Plan

## Phase 1: MVP Implementation

### Step 1: Create Progress Interface and Types
**File**: `progress.go`

- [x] Create `progress.go` with interface definition
  ```go
  type Progress interface {
      SetTotal(total int)
      SetCurrent(current int)
      Increment(n int)
      SetStatus(status string)
      SetColor(color ProgressColor)
      Complete()
      Fail(err error)
      IsActive() bool
  }
  ```
- [x] Define color constants (Default, Green, Red, Yellow, Blue)
- [x] Create `ProgressOptions` struct for configuration
- [x] Add documentation for all public APIs
- [x] Add copyright header and package documentation

### Step 2: Implement go-pretty Progress Wrapper
**File**: `progress_pretty.go`

+ [x] Create `PrettyProgress` struct with go-pretty progress.Writer and tracker
+ [x] Implement constructor `newPrettyProgress(settings *OutputSettings) *PrettyProgress`
+ [x] Implement `SetTotal(total int)` method
+ [x] Implement `SetCurrent(current int)` method
+ [x] Implement `Increment(n int)` method
+ [x] Implement `SetStatus(status string)` with message updates
+ [x] Implement `SetColor(color ProgressColor)` with style mapping
  - [x] Map ProgressColorGreen to go-pretty green style
  - [x] Map ProgressColorRed to go-pretty red style
  - [x] Map ProgressColorYellow to go-pretty yellow style
  - [x] Map ProgressColorBlue to go-pretty blue style
+ [x] Implement `Complete()` with success styling
+ [x] Implement `Fail(err error)` with error styling and message
+ [x] Implement `IsActive() bool` to check progress state
+ [x] Add internal `start()` and `stop()` methods for lifecycle
+ [x] Handle terminal detection (check if stdout is TTY)
+ [x] Add cleanup in case of panic/unexpected exit

### Step 3: Create No-op Implementation
**File**: `progress_noop.go`

- [ ] Create `NoOpProgress` struct
- [ ] Implement all Progress interface methods as no-ops
- [ ] Add minimal state tracking (current, total) for testing
- [ ] Ensure all methods are safe to call repeatedly
- [ ] Add debug logging option for troubleshooting

### Step 4: Add Factory Function and Integration
**File**: `format.go` (modifications)

- [ ] Add progress-related fields to `OutputSettings`:
  ```go
  type OutputSettings struct {
      // existing fields...
      ProgressEnabled bool
      ProgressOptions ProgressOptions
  }
  ```
- [ ] Create `NewProgress(settings *OutputSettings) Progress` function
- [ ] Add format detection logic:
  - [ ] Return `NoOpProgress` for JSON format
  - [ ] Return `NoOpProgress` for YAML format
  - [ ] Return `NoOpProgress` for CSV format
  - [ ] Return `NoOpProgress` for DOT format
  - [ ] Return `PrettyProgress` for table format
  - [ ] Return `PrettyProgress` for markdown format
  - [ ] Return `PrettyProgress` for HTML format (consider future enhancement)
- [ ] Add `EnableProgress()` and `DisableProgress()` methods to OutputSettings
- [ ] Add environment variable check (e.g., `GO_OUTPUT_PROGRESS=false` to disable)

### Step 5: Handle Edge Cases and Cleanup
**Files**: Various

- [ ] Add mutex protection for concurrent updates in PrettyProgress
- [ ] Ensure progress is properly stopped before other output:
  - [ ] Modify `Write()` method to check and stop active progress
  - [ ] Add progress cleanup to `OutputArray.Write()`
  - [ ] Add progress cleanup to `OutputSingle.Write()`
- [ ] Handle terminal resize gracefully
- [ ] Add context support for cancellation:
  - [ ] Add `SetContext(ctx context.Context)` method
  - [ ] Monitor context cancellation in progress loop
- [ ] Handle interrupt signals (SIGINT/SIGTERM):
  - [ ] Register signal handler
  - [ ] Cleanup progress bar on interrupt
  - [ ] Restore terminal state
- [ ] Add recovery from panics to ensure progress cleanup

### Step 6: Testing
**File**: `progress_test.go`, `progress_pretty_test.go`, `progress_noop_test.go`

- [ ] Test Progress interface compliance for both implementations
- [ ] Test basic progress operations:
  - [ ] SetTotal and SetCurrent
  - [ ] Increment with various values
  - [ ] SetStatus with different messages
- [ ] Test color changes:
  - [ ] Each color constant
  - [ ] Color changes during progress
  - [ ] Color in Complete() and Fail() states
- [ ] Test completion and failure states:
  - [ ] Normal completion
  - [ ] Failure with error message
  - [ ] State after completion/failure
- [ ] Test no-op implementation:
  - [ ] Ensure no output is produced
  - [ ] Verify state tracking works
- [ ] Test factory function logic:
  - [ ] Correct implementation for each format
  - [ ] Settings propagation
- [ ] Test concurrent usage:
  - [ ] Multiple goroutines updating progress
  - [ ] Race condition detection
- [ ] Mock terminal for testing:
  - [ ] Test TTY detection
  - [ ] Test non-TTY fallback
- [ ] Integration tests:
  - [ ] Progress with table output
  - [ ] Progress with multiple outputs

### Step 7: Documentation and Examples
**Files**: `examples/progress/`, README updates, inline documentation

- [ ] Create `examples/progress/basic/main.go`:
  - [ ] Simple progress bar example
  - [ ] Show total and increment usage
- [ ] Create `examples/progress/colors/main.go`:
  - [ ] Demonstrate color changes
  - [ ] Show success (green) and failure (red) scenarios
- [ ] Create `examples/progress/status/main.go`:
  - [ ] Dynamic status messages
  - [ ] Real-world use case (file processing, API calls, etc.)
- [ ] Update README.md:
  - [ ] Add progress feature to feature list
  - [ ] Add basic usage example
  - [ ] Document which output formats support progress
- [ ] Add inline documentation:
  - [ ] Package-level documentation for progress
  - [ ] Method documentation with examples
  - [ ] Document thread safety guarantees
- [ ] Create `examples/progress/with_output/main.go`:
  - [ ] Show progress followed by table output
  - [ ] Demonstrate proper cleanup

### MVP Checklist Summary
- [ ] All interface methods implemented
- [ ] go-pretty integration working
- [ ] No-op fallback working
- [ ] Terminal detection working
- [ ] Basic colors supported
- [ ] Thread-safe implementation
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Examples working

---

## Phase 2: Future Expansion Plan

### Option B: Runtime Style Selection

#### Step 1: Define Progress Styles
**File**: `progress_styles.go`

- [ ] Create progress style type and constants:
  ```go
  type ProgressStyle string
  
  const (
      ProgressStylePretty  ProgressStyle = "pretty"
      ProgressStyleSimple  ProgressStyle = "simple"
      ProgressStyleSpinner ProgressStyle = "spinner"
      ProgressStyleBar     ProgressStyle = "bar"
      ProgressStyleCustom  ProgressStyle = "custom"
  )
  ```
- [ ] Add style validation function
- [ ] Add style documentation

#### Step 2: Enhance OutputSettings
**File**: `format.go` (modifications)

- [ ] Add `ProgressStyle` field to `OutputSettings`
- [ ] Add `SetProgressStyle(style ProgressStyle)` method
- [ ] Add `ProgressStyleOptions` map for style-specific settings
- [ ] Add getter `GetProgressStyle() ProgressStyle`
- [ ] Update `NewOutputSettings()` to set default style
- [ ] Add validation in setter method

#### Step 3: Implement Additional Styles
**Files**: `progress_simple.go`, `progress_spinner.go`

##### SimpleProgress Implementation
- [ ] Create `SimpleProgress` struct
- [ ] Implement printf-style output (e.g., "Progress: 50/100 (50%)")
- [ ] Handle carriage returns for updates
- [ ] Implement all Progress interface methods
- [ ] Add configurable format string

##### SpinnerProgress Implementation
- [ ] Create `SpinnerProgress` struct
- [ ] Implement spinner animation (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
- [ ] Show spinner with status message
- [ ] Implement all Progress interface methods
- [ ] Add configurable spinner characters

#### Step 4: Update Factory Logic
**File**: `progress_factory.go`

- [ ] Create `createProgressForStyle()` function
- [ ] Implement style switch statement
- [ ] Add style registration map
- [ ] Handle unknown styles gracefully
- [ ] Add style availability detection

### Option C: Factory Pattern with Custom Implementations

#### Step 1: Define Factory Type
**File**: `progress.go` (additions)

- [ ] Add factory type definition:
  ```go
  type ProgressFactory func(settings *OutputSettings) Progress
  ```
- [ ] Add registry type definition
- [ ] Document factory pattern usage

#### Step 2: Enhance OutputSettings
**File**: `format.go` (modifications)

- [ ] Add `ProgressFactory` field to `OutputSettings`
- [ ] Add `SetProgressFactory(factory ProgressFactory)` method
- [ ] Add `RegisterProgressStyle(name string, factory ProgressFactory)`
- [ ] Add `GetRegisteredStyles() []string` method
- [ ] Handle nil factory gracefully

#### Step 3: Create Registry System
**File**: `progress_registry.go`

- [ ] Create global registry map with mutex
- [ ] Implement `RegisterProgressFactory()` function
- [ ] Implement `GetProgressFactory()` function
- [ ] Add default factories registration
- [ ] Add `UnregisterProgressFactory()` for testing
- [ ] Thread-safe registry operations

#### Step 4: Example Custom Implementation
**File**: `examples/custom_progress/main.go`

- [ ] Create example custom progress implementation
- [ ] Show factory registration
- [ ] Demonstrate usage with OutputSettings
- [ ] Include advanced features example

### Additional Future Enhancements

#### Multi-Progress Support
- [ ] Design multi-progress API
- [ ] Create progress group management
- [ ] Implement nested progress tracking
- [ ] Add progress aggregation

#### Rich Progress Information
- [ ] Add ETA calculation
- [ ] Add speed/throughput display
- [ ] Add data transfer indicators
- [ ] Add percentage with decimal places
- [ ] Add time elapsed display

#### Format-Specific Progress
- [ ] HTML: JavaScript-based progress bar
- [ ] JSON: Progress events as JSON lines
- [ ] Markdown: Progress as updating table
- [ ] CSV: Progress in comments or separate file

#### Advanced Features
- [ ] Progress persistence (save/resume)
- [ ] Progress logging to file
- [ ] Network-aware progress updates
- [ ] Progress webhooks/callbacks
- [ ] Distributed progress aggregation

### Testing Expansion
- [ ] Test each new progress style
- [ ] Test style switching at runtime
- [ ] Test custom factory registration
- [ ] Performance benchmarks
- [ ] Memory usage tests
- [ ] Long-running progress tests

### Documentation Expansion
- [ ] Document each progress style
- [ ] Add style comparison guide
- [ ] Document custom implementation guide
- [ ] Add troubleshooting section
- [ ] Create video demonstration

## Implementation Priority

1. **Week 1-2**: Complete MVP (Phase 1)
2. **Week 3**: Add Option B (Runtime Selection) with simple and spinner styles
3. **Week 4**: Add Option C (Factory Pattern) foundation
4. **Future**: Add advanced features based on user feedback

## Success Criteria

### MVP Success
- [ ] Progress shows in terminal for table output
- [ ] Colors indicate status clearly
- [ ] No interference with existing output
- [ ] Zero configuration required for basic use
- [ ] Works on Linux, macOS, and Windows

### Expansion Success
- [ ] Users can choose progress styles
- [ ] Custom implementations possible
- [ ] Backward compatibility maintained
- [ ] Performance overhead < 5%
- [ ] Memory usage reasonable for long-running tasks