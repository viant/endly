package reporter

import (
	"bytes"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
)

//ReportRecord represents a report record
type ReportRecord struct {
	Id     int    `autoincrement:"true"`
	Name   string `column:"name"`
	Type   string `column:"type"`
	Report string `column:"report"`
}

type reportDao struct {
}

func (d *reportDao) Persist(manager dsc.Manager, report Report) error {
	var records = make([]*ReportRecord, 0)
	err := manager.ReadAll(&records, "SELECT id, name, type, report FROM report WHERE name = ?", []interface{}{report.GetName()}, nil)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		records = append(records, &ReportRecord{})
	}

	records[0].Name = report.GetName()
	records[0].Type = report.GetType()
	records[0].Report, err  =toolbox.AsJSONText(report.Unwrap())
	if err != nil {
		return err
	}
	_, _, err = manager.PersistAll(&records, "report", nil)
	if err != nil {
		return err
	}
	return nil
}
