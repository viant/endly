package bigquery

import (
	"fmt"
	"google.golang.org/api/bigquery/v2"
	"strings"
)

//NewTableReference creates a table reference for table in the following syntax [project:]dataset.table
func NewTableReference(table string) (*bigquery.TableReference, error) {
	dotIndex := strings.LastIndex(table, ".")
	if dotIndex == -1 {
		return nil, fmt.Errorf("datasetID is missing, invalid table format: %v", table)
	}
	tableID := string(table[dotIndex+1:])
	datasetID := string(table[:dotIndex])
	projectID := ""
	if index := strings.Index(datasetID, ":"); index != -1 {
		projectID = string(datasetID[:index])
		datasetID = string(datasetID[index+1:])
	}
	return &bigquery.TableReference{
		TableId:   tableID,
		DatasetId: datasetID,
		ProjectId: projectID,
	}, nil
}
