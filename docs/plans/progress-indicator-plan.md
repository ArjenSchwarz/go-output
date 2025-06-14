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

- [x] Create `NoOpProgress` struct
- [x] Implement all Progress interface methods as no-ops
- [x] Add minimal state tracking (current, total) for testing
- [x] Ensure all methods are safe to call repeatedly
- [x] Add debug logging option for troubleshooting

### Step 4: Add Factory Function and Integration
**File**: `format.go` (modifications)

 - [x] Add progress-related fields to `OutputSettings`:
  ```go
  type OutputSettings struct {
      // existing fields...
      ProgressEnabled bool
      ProgressOptions ProgressOptions
  }
  ```
 - [x] Create `NewProgress(settings *OutputSettings) Progress` function
 - [x] Add format detection logic:
  - [x] Return `NoOpProgress` for JSON format
  - [x] Return `NoOpProgress` for YAML format
  - [x] Return `NoOpProgress` for CSV format
  - [x] Return `NoOpProgress` for DOT format
  - [x] Return `PrettyProgress` for table format
  - [x] Return `PrettyProgress` for markdown format
  - [x] Return `PrettyProgress` for HTML format (consider future enhancement)
 - [x] Add `EnableProgress()` and `DisableProgress()` methods to OutputSettings
 - [x] Add environment variable check (e.g., `GO_OUTPUT_PROGRESS=false` to disable)

### Step 5: Handle Edge Cases and Cleanup
**Files**: Various

 - [x] Add mutex protection for concurrent updates in PrettyProgress
- [ ] Ensure progress is properly stopped before other output:
  - [x] Modify `Write()` method to check and stop active progress
  - [x] Add progress cleanup to `OutputArray.Write()`
  - [ ] Add progress cleanup to `OutputSingle.Write()`
 - [x] Handle terminal resize gracefully
- [x] Add context support for cancellation:
  - [x] Add `SetContext(ctx context.Context)` method
  - [x] Monitor context cancellation in progress loop
- [x] Handle interrupt signals (SIGINT/SIGTERM):
  - [x] Register signal handler
  - [x] Cleanup progress bar on interrupt
  - [x] Restore terminal state
 - [x] Add recovery from panics to ensure progress cleanup

### Step 6: Testing
**File**: `progress_test.go`, `progress_pretty_test.go`, `progress_noop_test.go`

- [x] Test Progress interface compliance for both implementations
- [x] Test basic progress operations:
  - [x] SetTotal and SetCurrent
  - [x] Increment with various values
  - [x] SetStatus with different messages
- [x] Test color changes:
  - [x] Each color constant
  - [x] Color changes during progress
  - [x] Color in Complete() and Fail() states
- [x] Test completion and failure states:
  - [x] Normal completion
  - [x] Failure with error message
  - [x] State after completion/failure
- [x] Test no-op implementation:
  - [x] Ensure no output is produced
  - [x] Verify state tracking works
- [x] Test factory function logic:
  - [x] Correct implementation for each format
  - [x] Settings propagation
- [x] Test concurrent usage:
  - [x] Multiple goroutines updating progress
  - [ ] Race condition detection
- [x] Mock terminal for testing:
  - [x] Test TTY detection
  - [x] Test non-TTY fallback
- [x] Integration tests:
  - [x] Progress with table output
  - [ ] Progress with multiple outputs

### Step 7: Documentation and Examples
**Files**: `examples/progress/`, README updates, inline documentation

- [x] Create `examples/progress/basic/main.go`:
  - [x] Simple progress bar example
  - [x] Show total and increment usage
- [x] Create `examples/progress/colors/main.go`:
  - [x] Demonstrate color changes
  - [x] Show success (green) and failure (red) scenarios
- [x] Create `examples/progress/status/main.go`:
  - [x] Dynamic status messages
  - [x] Real-world use case (file processing, API calls, etc.)
- [x] Update README.md:
  - [x] Add progress feature to feature list
  - [x] Add basic usage example
  - [x] Document which output formats support progress
- [x] Add inline documentation:
  - [x] Package-level documentation for progress
  - [x] Method documentation with examples
  - [x] Document thread safety guarantees
- [x] Create `examples/progress/with_output/main.go`:
  - [x] Show progress followed by table output
  - [x] Demonstrate proper cleanup

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