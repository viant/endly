**Http runner service**

Http runner sends one or more HTTP request to the specified endpoint; 
it manages cookie within [SendRequest](service_contract.go).

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| http/runner | send | Sends one or more http request to the specified endpoint. | [SendRequest](service_contract.go) | [SendResponse](service_contract.go) |


**Usage**

_Sequential and conditional request sending_

```go
	import (
		"log"
		"net/http"
		"github.com/viant/endly"
		"github.com/viant/toolbox"
		runner "github.com/viant/endly/testing/runner/http"
	)
	
	
	func main() {
        response := &runner.SendResponse{}
        
        err := endly.Run(context, &runner.SendRequest{
            Options:[]*toolbox.HttpOptions{
                 {
                    Key:"RequestTimeoutMs",
                    Value:12000,
                 },	
            },
            Requests: []*runner.Request{
                {
                    URL:    "http://127.0.0.1:8111/send1",
                    Method: "POST",
                    Body:   "some body",
                    Header:http.Header{
                        "User-Agent":[]string{"myUa"},
                    },
                },
                {   //Only run the second request in previous response body contains 'content1-2' fragment
                    When:   "${httpTrips.Response[0].Body}:/content1-2/",
                    URL:    "http://127.0.0.1:8111/send2",
                    Method: "POST",
                    Body:   "other body",
                },
            },
        }, response)
        if err != nil {
            log.Fatal(err)
        }
        
    }

```




