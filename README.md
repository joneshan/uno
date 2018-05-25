# uno

A human-readable unique number generate service. Useful for room games,
just like board games.

## Install

```bash
go get -u github.com/acrazing/uno
```

## Usage

Import the package at first:

```go
import (
	"time"
	"context"
	"github.com/acrazing/uno"
)
```

### Use the worker directly

1. init a worker, and run it
  ```go
  w := uno.NewWorker()
  w.Init(&uno.Options{
    // the pool size to allocate free numbers
    PoolVolume: 100,
    // the min value of the number, must > 1
    MinValue: 1e5,
    // the max value of the number
    // must > MinValue
    // and the range of the number is [MinValue, MaxValue)
    MaxValue: 2e5,
    // time to live, if a number is rented, the consumer
    // must relet it the the time, else the rent will be
    // expired, and then will be returned automatically.
    TTL: time.Hour,
    // time to freeze, after return a number, it will not
    // be available in the duration, after the duration,
    // it will be freed, and will be reused for next rent.
    TTF: time.Hour,
  })
  go w.Run(context.Background())
  ```
2. rent a number
  ```go
  no := w.Rent()
  if no != 0 {
    // do any staff with the rented value
    // if no is 0, means exhausted
    print(no) // -> 10000
  }
  ```
3. relet a number
  ```go
  ok := w.Relet(no)
  if ok {
  	// relet successfully
  }
  ```
4. return a number
  ```go
  w.Return(no) // no return value
  ```
  
### Use gRPC Service

The grpc service messages is generated, you can use it directly,
you can get a example at [example/main.go](./example/main.go).

```go
addr := "127.0.0.1:1234"

// server
listener, err := net.Listen("tcp", addr)
if err != nil {
  panic(err)
}
server := grpc.NewServer()
// the package exported a global service instance
uno.RegisterUnoServer(server, uno.Service)
uno.Service.Init(&uno.Options{
  MinValue: 2,
  MaxValue: 5,
  TTF:      time.Second,
  TTL:      time.Second,
})
go uno.Service.Run(context.Background())
go server.Serve(listener)

// client
conn, _ := grpc.Dial(addr, grpc.WithInsecure())
client := uno.NewUnoClient(conn)
no, err := client.Rent(context.Background(), &uno.Empty{})
log.Printf("rent: %v, err: %v", no, err)
```

## License

```
The MIT License (MIT)

Copyright (c) 2018 acrazing

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
