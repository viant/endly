package endly_test
//
//import (
//	"github.com/stretchr/testify/assert"
//	"github.com/viant/endly"
//	"github.com/viant/toolbox"
//	"github.com/viant/toolbox/url"
//	"testing"
//)
//
//func TestProcessService_Run(t *testing.T) {
//
//	if toolbox.FileExists("/usr/local/Cellar/mongodb/3.4.4/bin/mongod") {
//
//		manager := endly.NewManager()
//		srv, err := manager.Service(endly.ProcessServiceID)
//		assert.Nil(t, err)
//		assert.NotNil(t, srv)
//
//		context := manager.NewContext(toolbox.NewContext())
//		defer context.Close()
//
//		response := srv.Run(context, &endly.ProcessStartRequest{
//			Target: &url.Resource{
//				URL: "scp://127.0.0.1/",
//			},
//			Options: &endly.ExecutionOptions{
//				SystemPaths: []string{"/usr/local/Cellar/mongodb/3.4.4/bin"},
//			},
//			Command:   "mongod",
//			Arguments: []string{"--dbpath", "/tmp"},
//		})
//		assert.Equal(t, "", response.Error)
//		serviceResponse, ok := response.Response.(*endly.ProcessStartResponse)
//		assert.True(t, ok)
//		assert.True(t, serviceResponse.Info[0].Pid > 0)
//
//		response = srv.Run(context, &endly.ProcessStopRequest{
//			Target: &url.Resource{
//				URL: "scp://127.0.0.1/",
//			},
//			Pid: serviceResponse.Info[0].Pid,
//		})
//		assert.Equal(t, "", response.Error)
//	}
//}
