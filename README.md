# Redis Adapter [![Go](https://github.com/casbin/redis-adapter/actions/workflows/ci.yml/badge.svg)](https://github.com/casbin/redis-adapter/actions/workflows/ci.yml)[![Coverage Status](https://coveralls.io/repos/github/casbin/redis-adapter/badge.svg?branch=master)](https://coveralls.io/github/casbin/redis-adapter?branch=master)[![Go Report Card](https://goreportcard.com/badge/github.com/casbin/redis-adapter)](https://goreportcard.com/report/github.com/casbin/redis-adapter)[![Godoc](https://godoc.org/github.com/casbin/redis-adapter?status.svg)](https://godoc.org/github.com/casbin/redis-adapter)

Redis Adapter is the [Redis](https://redis.io/) adapter for [Casbin](https://github.com/casbin/casbin). With this library, Casbin can load policy from Redis or save policy to it.

## Installation

    go get github.com/casbin/redis-adapter/v3

## Simple Example

```go
package main

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/redis-adapter/v3"
)

func main() {
	// Direct Initialization:
	// Initialize a Redis adapter and use it in a Casbin enforcer:
	a, _ := redisadapter.NewAdapter("tcp", "127.0.0.1:6379") // Your Redis network and address.

	// Use the following if Redis has password like "123"
	// a, err := redisadapter.NewAdapterWithPassword("tcp", "127.0.0.1:6379", "123")

	// Use the following if you use Redis with a specific user 
	// a, err := redisadapter.NewAdapterWithUser("tcp", "127.0.0.1:6379", "username", "password")

	// Use the following if you use Redis connections pool
	// pool := &redis.Pool{}
	// a, err := redisadapter.NewAdapterWithPool(pool)

	// Initialization with different user options:
	// Use the following if you use Redis with passowrd like "123":
	// a, err := redisadapter.NewAdapterWithOption(redisadapter.WithNetwork("tcp"), redisadapter.WithAddress("127.0.0.1:6379"), redisadapter.WithPassword("123"))

	// Use the following if you use Redis with username, password, and TLS option:
	// var clientTLSConfig tls.Config
	// ...
	// a, err := redisadapter.NewAdapterWithOption(redisadapter.WithNetwork("tcp"), redisadapter.WithAddress("127.0.0.1:6379"), redisadapter.WithUsername("testAccount"), redisadapter.WithPassword("123456"), redisadapter.WithTls(&clientTLSConfig))

	e, _ := casbin.NewEnforcer("examples/rbac_model.conf", a)

	// Load the policy from DB.
	e.LoadPolicy()

	// Check the permission.
	e.Enforce("alice", "data1", "read")

	// Modify the policy.
	// e.AddPolicy(...)
	// e.RemovePolicy(...)

	// Save the policy back to DB.
	e.SavePolicy()
}
```

## Getting Help

- [Casbin](https://github.com/casbin/casbin)

## License

This project is under Apache 2.0 License. See the [LICENSE](LICENSE) file for the full license text.
