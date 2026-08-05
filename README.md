# gopher-hole

`gopher-hole` is a repository for **generic Go functionality**: reusable
packages, utilities, and patterns that can be used across different Go
services and applications.

## What this repository contains

This repo is organized as a collection of focused Go packages, including:

- `util/` for common helper functionality
- `metrics/` for metrics-related code and monitoring helpers
- `lender/` for model lending/serving functionality
- `internal/` for private implementation details not exposed outside this module

## Module

Go module path:

```go
module github.com/sassoftware/gopher-hole
```

## Getting started

Clone the repository and run tests:

```bash
go test ./...
```
