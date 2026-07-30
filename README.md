# Honeybadger API Client for Go

Go clients for the Honeybadger API. There are two, one per API version:

| Import | API |
| --- | --- |
| `github.com/honeybadger-io/api-go/apiv3` | v3 — where new work belongs |
| `github.com/honeybadger-io/api-go/apiv2` | v2 — the Data API |

Which you need depends on the credential you hold. v3 takes scoped API tokens
(`hbt_` personal, `hba_` account) and OAuth access tokens, always as Bearer; it
rejects v2's older personal auth tokens outright. So moving to v3 means a new
credential, not only new code.

Both are built against a vendored OpenAPI bundle under `openapi/` — see
[openapi/README.md](openapi/README.md) for refreshing it, and
[openapi/GAPS.md](openapi/GAPS.md) for what v2 could do that v3 cannot yet.

> **Moving from v0.8.0:** the v2 services used to live in the module root. They
> are now in `apiv2`, so update the import path and use `apiv2.NewClient()`.
> Nothing else about v2's surface changed.

## Installation

```bash
go get github.com/honeybadger-io/api-go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/honeybadger-io/api-go/apiv2"
)

func main() {
    // Create a new client
    client := apiv2.NewClient().
        WithAuthToken("your-api-token")

    // List all projects
    projects, err := client.Projects.ListAll(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, project := range projects.Results {
        fmt.Printf("Project: %s (ID: %d)\n", project.Name, project.ID)
    }

    // List faults for a project
    faults, err := client.Faults.List(context.Background(), projectID, apiv2.FaultListOptions{
        Order: "recent",
        Limit: 10,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, fault := range faults.Results {
        fmt.Printf("Fault: %s - %s\n", fault.Klass, fault.Message)
    }
}
```

## Features

- Automatic pagination support
- Type-safe API responses
- Context support for cancellation and timeouts

## Documentation

For more information about the Honeybadger API, see the [official documentation](https://docs.honeybadger.io/api/#data-api).

## Development

Run the tests:

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add my amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
