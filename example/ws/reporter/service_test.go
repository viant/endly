package reporter_test

import (
	"testing"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/endly/example/ws/reporter"
	"github.com/viant/toolbox/url"
	"github.com/stretchr/testify/assert"
	"os"
)

func Test_Service(t *testing.T) {

		if os.Getenv("RUN_TEST") == "" {
			return
		}

		config := &reporter.Config{}
		configResource := url.NewResource("endly/config/config.json")
		err := configResource.JSONDecode(config)
		if assert.Nil(t, err) {
			service, err := reporter.NewService(config)
			if assert.Nil(t, err) {

				var pivot = &reporter.PivotReport{

					Name:"report1",

					From:"expenditure",

					Values: []*reporter.AggregatedValue{
						{
							Function:"SUM",
							Column:"expenditure",
						},
					},

					Columns: []*reporter.AliasedColumn {
						{
							Name:"category",
							Alias:"",
						},
					},
					Groups :[]string{"year"},

				}

				service.Register(&reporter.RegisterReportRequest{
					ReportType:"pivot",
					Report:pivot,
				})

				response := service.Run(&reporter.RunReportRequest{
					Name:"report1",
					Datastore:"db1",
					Parameters:map[string]interface{}{},
				})

				assert.Equal(t, "", response.Error)

			}

		}

}