package udf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

type xlsxChartSeries struct {
	chartType, direction, grouping string
	label, categories, values      string
	signedFill                     bool
}

type xlsxChart struct {
	references    []string
	legendPresent bool
	series        []xlsxChartSeries
}

func (c xlsxChart) result() map[string]interface{} {
	series := make([]map[string]interface{}, 0, len(c.series))
	for _, item := range c.series {
		series = append(series, map[string]interface{}{
			"type": item.chartType, "direction": xlsxOptionalString(item.direction),
			"grouping": xlsxOptionalString(item.grouping), "label": xlsxOptionalString(item.label),
			"categories": xlsxOptionalString(item.categories), "values": xlsxOptionalString(item.values),
			"signedFill": item.signedFill,
		})
	}
	return map[string]interface{}{"references": c.references, "legendPresent": c.legendPresent, "series": series}
}

func xlsxOptionalString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func inspectXlsxChart(raw []byte, budget *xlsxBudget) (xlsxChart, error) {
	result := xlsxChart{references: []string{}}
	d := xml.NewDecoder(bytes.NewReader(raw))
	err := scanXlsxPart(d, xml.Name{Space: xlsxChartNS, Local: "chartSpace"}, func(d *xml.Decoder, root xml.StartElement) error {
		seenChart := false
		err := scanXlsxElements(d, func(child xml.StartElement) error {
			if child.Name == (xml.Name{Space: xlsxChartNS, Local: "chart"}) {
				if seenChart {
					return fmt.Errorf("ambiguous XLSX chart definition")
				}
				seenChart = true
				return result.scanChart(d, budget)
			}
			return result.scanReferences(d, budget)
		})
		if err == nil && !seenChart {
			return fmt.Errorf("missing XLSX chart definition")
		}
		return err
	})
	return result, err
}

func (c *xlsxChart) scanChart(d *xml.Decoder, budget *xlsxBudget) error {
	return scanXlsxElements(d, func(child xml.StartElement) error {
		if child.Name.Space == xlsxChartNS {
			switch child.Name.Local {
			case "legend":
				c.legendPresent = true
			case "plotArea":
				return scanXlsxElements(d, func(plot xml.StartElement) error {
					if plot.Name.Space == xlsxChartNS && strings.HasSuffix(plot.Name.Local, "Chart") {
						return c.scanPlot(d, plot.Name.Local, budget)
					}
					return c.scanReferences(d, budget)
				})
			}
		}
		return c.scanReferences(d, budget)
	})
}

func (c *xlsxChart) scanPlot(d *xml.Decoder, kind string, budget *xlsxBudget) error {
	firstSeries := len(c.series)
	var direction, grouping string
	err := scanXlsxElements(d, func(child xml.StartElement) error {
		if child.Name.Space == xlsxChartNS {
			switch child.Name.Local {
			case "barDir", "grouping":
				value, err := xlsxAttribute(child, "val")
				if err != nil {
					return err
				}
				if child.Name.Local == "barDir" {
					direction = value
				} else {
					grouping = value
				}
				return d.Skip()
			case "ser":
				series := xlsxChartSeries{chartType: kind}
				if err := c.scanSeries(d, &series, budget); err != nil {
					return err
				}
				c.series = append(c.series, series)
				return nil
			}
		}
		return c.scanReferences(d, budget)
	})
	if err != nil {
		return err
	}
	for i := firstSeries; i < len(c.series); i++ {
		if err := budget.consumeText(kind, direction, grouping); err != nil {
			return err
		}
		c.series[i].direction, c.series[i].grouping = direction, grouping
	}
	return nil
}

func (c *xlsxChart) scanSeries(d *xml.Decoder, series *xlsxChartSeries, budget *xlsxBudget) error {
	return scanXlsxElements(d, func(child xml.StartElement) error {
		var role *string
		if child.Name.Space == xlsxChartNS {
			switch child.Name.Local {
			case "tx":
				role = &series.label
			case "cat", "xVal":
				role = &series.categories
			case "val", "yVal":
				role = &series.values
			}
		}
		return walkXlsxElements(d, func(d *xml.Decoder, element xml.StartElement) (bool, error) {
			if element.Name.Local == "invertSolidFillFmt" {
				series.signedFill = true
			}
			if element.Name == (xml.Name{Space: xlsxChartNS, Local: "f"}) {
				value, err := c.readReference(d, element, budget)
				if err != nil {
					return true, err
				}
				if role != nil && *role == "" {
					if err := budget.consumeText(value); err != nil {
						return true, err
					}
					*role = value
				}
				return true, nil
			}
			return false, nil
		})
	})
}

func (c *xlsxChart) scanReferences(d *xml.Decoder, budget *xlsxBudget) error {
	return walkXlsxElements(d, func(d *xml.Decoder, element xml.StartElement) (bool, error) {
		if element.Name == (xml.Name{Space: xlsxChartNS, Local: "f"}) {
			_, err := c.readReference(d, element, budget)
			return true, err
		}
		return false, nil
	})
}

func (c *xlsxChart) readReference(d *xml.Decoder, element xml.StartElement, budget *xlsxBudget) (string, error) {
	var value string
	if err := d.DecodeElement(&value, &element); err != nil {
		return "", err
	}
	if err := budget.consumeText(value); err != nil {
		return "", err
	}
	c.references = append(c.references, value)
	return value, nil
}
