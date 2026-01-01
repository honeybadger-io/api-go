# Honeybadger API Client for Go

A Go client library for the Honeybadger API.

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

    hbapi "github.com/honeybadger-io/api-go"
)

func main() {
    // Create a new client
    client := hbapi.NewClient().
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
    faults, err := client.Faults.List(context.Background(), projectID, hbapi.FaultListOptions{
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
