# gopher-hole

`gopher-hole` is a repository for **generic Go functionality**: reusable
packages, utilities, and patterns that can be used across different Go
services and applications.

## Overview

This repo is organized as a collection of focused Go packages, including:

- `util/` for common helper functionality
- `metrics/` for metrics-related code and monitoring helpers
- `lender/` for model lending/serving functionality
- `internal/` for private implementation details not exposed outside this module

## Table of Contents

- [gopher-hole](#gopher-hole)
  - [Overview](#overview)
  - [Table of Contents](#table-of-contents)
  - [Module](#module)
  - [Getting started](#getting-started)
  - [Contributing](#contributing)
  - [Security Policy](#security-policy)
  - [License](#license)
  - [Third-Party Dependencies](#third-party-dependencies)

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

---

## Contributing

Maintainers are accepting patches and contributions to this project.
Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details about submitting
contributions to this project.

---

## Security Policy

Please see our [Security Policy](SECURITY.md) for details.

## License

This project is licensed under the [Apache 2.0 License](LICENSE).

<!-- markdownlint-disable MD013 -->
This project's logo features the Go Gopher mascot, [created by Renee French](https://go.dev/brand) and licensed under the [Creative Commons 4.0 Attribution License](https://creativecommons.org/licenses/by/4.0/).
<!-- markdownlint-enable MD013 -->

---

## Third-Party Dependencies

<!-- markdownlint-disable MD013 -->
| Dependency | License |
| --- | --- |
| `github.com/prometheus/client_golang` | [LICENSE](https://github.com/prometheus/client_golang/blob/main/LICENSE) |
| `github.com/prometheus/client_model` | [LICENSE](https://github.com/prometheus/client_model/blob/main/LICENSE) |
| `github.com/rs/zerolog` | [LICENSE](https://github.com/rs/zerolog/blob/master/LICENSE) |
| `github.com/stretchr/testify` | [LICENSE](https://github.com/stretchr/testify/blob/master/LICENSE) |
<!-- markdownlint-enable MD013 -->
