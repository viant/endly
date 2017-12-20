package reporter

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

type AggregatedValue struct {
	Column   string
	Function string
}

type AliasedColumn struct {
	Name  string
	Alias string
}

type PivotReport struct {
	Name string

	From string

	Values []*AggregatedValue

	Columns []*AliasedColumn

	Groups []string

	Where string
}

func (r *PivotReport) GetName() string {
	return r.Name
}

func (r *PivotReport) GetType() string {
	return "pivot"
}

func (r *PivotReport) Unwrap() interface{} {
	return r
}

func (r *PivotReport) SQL(manager dsc.Manager, parameters map[string]interface{}) (string, error) {
	var result = ""
	if len(parameters) == 0 {
		parameters = make(map[string]interface{})
	}
	var context = data.Map(parameters)

	for _, value := range r.Values {
		value.Function = strings.ToUpper(value.Function)
	}

	var sqlColumns = append(make([]string, 0), r.Groups...)

	var whereClause = ""
	if r.Where != "" {
		whereClause = " WHERE " + r.Where
	}
	for _, column := range r.Columns {
		var SQL = fmt.Sprintf("SELECT %v AS name, COUNT(1) AS cnt  FROM %v %v GROUP BY 1 ORDER BY 2 DESC", column.Name, r.From, whereClause)
		SQL = context.ExpandAsText(SQL)
		columnValues := make([]*AggValue, 0)
		err := manager.ReadAll(&columnValues, SQL, nil, nil)
		if err != nil {
			return "", err
		}

		for _, columnValue := range columnValues {
			for _, value := range r.Values {
				var isSumAggregate = value.Function == "SUM"
				var elseValue = "NULL"
				if isSumAggregate {
					elseValue = "0"
				}
				columnValue.Name = strings.Replace(columnValue.Name, " ", "", len(columnValue.Name))
				matchValue := columnValue.Name
				if toolbox.AsInt(matchValue) == 0 && matchValue != "0" {
					matchValue = fmt.Sprintf("'%v'", matchValue)
				}
				var column = fmt.Sprintf("%v(CASE WHEN %v = %v THEN %v ELSE %v END) AS %v%v", value.Function, column.Name, matchValue, value.Column, elseValue, column.Alias, columnValue.Name)
				sqlColumns = append(sqlColumns, column)
			}
		}
	}
	var groupBy = ""
	var orderBy = ""

	if len(r.Groups) > 0 {
		var groupByPosition = make([]string, 0)
		var orderByPosition = make([]string, 0)
		for i := range r.Groups {
			groupByPosition = append(groupByPosition, toolbox.AsString(i+1))
			if i > 0 {
				orderByPosition = append(orderByPosition, fmt.Sprintf("%v DESC", 1+len(r.Groups)-i))
			}
		}
		groupBy = "GROUP BY " + strings.Join(groupByPosition, ",")
		if len(r.Groups) > 1 {
			orderBy = "ORDER BY " + strings.Join(orderByPosition, ",")
		}

	}
	result = fmt.Sprintf("SELECT %v \nFROM %v %v \n%v \n%v", strings.Join(sqlColumns, ",\n"), r.From, whereClause, groupBy, orderBy)
	result = context.ExpandAsText(result)

	fmt.Println(result)
	return result, nil
}

type AggValue struct {
	Name string
	Cnt  int
}

func PivotReportProvider(report interface{}) (Report, error) {
	converter := toolbox.Converter{}
	var result = &PivotReport{}
	err := converter.AssignConverted(result, report)
	if err != nil {
		return nil, err
	}
	return result, nil
}
