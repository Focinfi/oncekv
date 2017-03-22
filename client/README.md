### oncekv Client

A HTTP Client for oncekv.

#### Design
1. Fast fail.
2. Rember the last last server.
3. Clean API.

#### Example

```go
import "github.com/Focinfi/oncekv/client"

kv, err := client.NewKV(&client.Option{
  RequestTimeout:        requestTimeout,
  IdealResponseDuration: idealReponseDuration,
}) 

// or create a kv with default option
kv, err := client.DefaultKV()

// Put foo/bar
err := kv.Put("foo", "bar")
// Get foo 
val, err := kv.Get("foo")
```