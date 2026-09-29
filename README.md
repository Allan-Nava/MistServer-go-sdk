# MistServer Go SDK
[![CI](https://github.com/Allan-Nava/MistServer-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/Allan-Nava/MistServer-go-sdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Allan-Nava/MistServer-go-sdk/mist.svg)](https://pkg.go.dev/github.com/Allan-Nava/MistServer-go-sdk/mist)

MistServer is a streaming media server that works well in any streaming environment even on a Raspberry Pi! It bridges the gap between dedicated media servers and web servers, performing the best of both worlds when it comes to media streaming delivery.

[Official Rest API Documentation](https://mistserver.org/documentation)

MistServer-go-sdk is a Go client library for interfacing with the MistServer API. It provides a convenient way to interact with the server and perform operations like creating and managing streams, retrieving server information, and managing access control.

## Installation

Use go get to install the library:

```bash
go get github.com/Allan-Nava/MistServer-go-sdk
```

Requires Go 1.23 or newer.

## Usage

```go
import (
	"errors"

	mist "github.com/Allan-Nava/MistServer-go-sdk/mist"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

logger, _ := zap.NewProduction()

client := mist.NewService(
	resty.New(),     // nil → resty.New()
	logger.Sugar(),  // nil → no-op logger
	mist.WithBaseURL("http://localhost:4242/api"),
	mist.WithUsername("admin"),
	mist.WithPassword("secret"),
)

pushes, err := client.PostPushList(mist.PostPushListRequest{PushList: true})
if errors.Is(err, mist.ErrUnauthorized) {
	// wrong credentials
}
```

The client performs MistServer's challenge-response login for you and caches it for a minute.

## API Reference
See [pkg.go.dev](https://pkg.go.dev/github.com/Allan-Nava/MistServer-go-sdk/mist) and the
[project site](https://allan-nava.github.io/MistServer-go-sdk/).

## Contribute
We welcome contributions to this project. If you want to contribute, please fork the repository and submit a pull request with your changes.

## Contributors

<!-- readme: contributors -start -->
<!-- readme: contributors -end -->

## License
MistServer-go-sdk is licensed under the MIT License. See LICENSE for more information.
