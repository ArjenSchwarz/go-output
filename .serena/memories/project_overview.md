# Go-Output Project Overview

## Project Purpose
go-output v2 is a comprehensive Go library for outputting structured data in multiple formats with thread-safe operations and preserved key ordering. The library provides a unified interface to convert data into JSON, YAML, CSV, HTML, tables, markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

Version 2.0 represents a complete redesign with no backward compatibility, eliminating global state and providing modern Go 1.24+ features.

## Tech Stack
- **Language**: Go 1.21+ (targeting Go 1.24+ features in v2)
- **Key Dependencies**:
  - go-pretty v6.4.9 for table formatting and styling
  - dot v1.6.0 for DOT graph generation
  - aws-sdk-go-v2 for S3 integration
  - yaml.v3 for YAML processing
  - slug v1.13.1 for URL-safe string generation
  - color v1.16.0 for terminal colors

## Core Architecture
- **Document-Builder Pattern**: Immutable container for content and metadata with fluent API
- **Thread-Safe Operations**: All components designed for concurrent use
- **Key Order Preservation**: Maintains exact user-specified column ordering
- **Content Type System**: Tables, text, raw content, and hierarchical sections
- **Transform Pipeline**: Data transformation capabilities before rendering
- **Multiple Writers**: Output to stdout, files, S3, or multiple destinations

## Current Development Focus
The project is actively developing the **transformation-pipeline** enhancement, which evolves the v2 transformation system from simple post-rendering text manipulation to comprehensive data transformation framework.