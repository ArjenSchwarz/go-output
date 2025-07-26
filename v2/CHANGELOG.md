# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Core CollapsibleValue interface for cross-format expandable content
- DefaultCollapsibleValue implementation with functional options pattern
- Support for configurable character limits and truncation indicators
- Format-specific rendering hints system
- Comprehensive error handling with graceful fallbacks for nil details and empty summaries
- Unit tests covering all interface methods, edge cases, and character truncation functionality