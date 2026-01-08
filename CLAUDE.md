# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Go client library for the Honeybadger API (`github.com/honeybadger-io/api-go`). It provides type-safe access to Honeybadger's v2 Data API for managing projects, faults, check-ins, uptime monitoring, teams, and more.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestFunctionName ./...

# Run tests with verbose output
go test -v ./...
```

## Architecture

### Client Structure

The main `Client` struct (client.go) uses a builder pattern for configuration and contains service pointers for each API resource:

```go
client := hbapi.NewClient().
    WithAuthToken("token").
    WithBaseURL("https://api.honeybadger.io")
```

All API requests go through `client.newRequest()` which automatically:
- Prepends `/v2` to all paths
- Sets HTTP Basic Auth with the API token as username
- Sets JSON content headers

### Service Pattern

Each API resource has its own service file (e.g., `projects.go`, `faults.go`, `uptime.go`) following a consistent pattern:
- Service struct with a `client` pointer
- Methods that construct paths, create requests, and decode responses
- Associated types defined in `types.go`

### Types

All API request/response types are centralized in `types.go`. Common patterns:
- `*Response` types for list endpoints (contain `Results` slice)
- `*Request` types wrap params in a nested struct (e.g., `{"project": {...}}`)
- `*Params` types for create/update parameters
- `*Options` types for query parameters with `url:"..."` tags

### Error Handling

Custom error handling in `errors.go` wraps HTTP errors with status codes and response bodies via `WrapError()`.
