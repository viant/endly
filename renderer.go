package endly

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/viant/toolbox"
	"golang.org/x/net/context"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

var minColumns = 120

var colors = map[string]func(arg interface{}) aurora.Value{
	"red":     aurora.Red,
	"green":   aurora.Green,
	"blue":    aurora.Blue,
	"bold":    aurora.Bold,
	"brown":   aurora.Brown,
	"gray":    aurora.Gray,
	"cyan":    aurora.Cyan,
	"magenta": aurora.Magenta,
	"inverse": aurora.Inverse,
}

//Renderer represents a renderer
type Renderer struct {
	ErrorColor string
	writer     io.Writer
	minColumns int
	lines      int
}

//Printf formats and print supplied text with arguments
func (r *Renderer) Printf(text string, args ...interface{}) {
	r.Print(aurora.Sprintf(text, args...))
}

//Sprintf returns formatted text with arguments
func (r *Renderer) Sprintf(text string, args ...interface{}) string {
	return aurora.Sprintf(text, args...)
}

//Println returns formatted text with arguments
func (r *Renderer) Println(text string) {
	r.Print(text + "\n")
}

//Print prints supplied message
func (r *Renderer) Print(message string) {
	r.writer.Write([]byte(message))
}

//ColorText returns text with ANCI color
func (r *Renderer) ColorText(text string, textColors ...string) string {
	for _, color := range textColors {
		if color, has := colors[color]; has {
			text = aurora.Sprintf("%v", color(text))
		}
	}
	return text
}

//Columns reutnrs terminal column count
func (r *Renderer) Columns() int {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "tput", "cols")
	output, err := cmd.CombinedOutput()
	if err == nil {
		r.lines, err = strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			r.lines = minColumns
		}
	}
	if r.lines < minColumns {
		r.lines = minColumns
	}
	return r.lines
}

func (r *Renderer) getColumnSize(headers []string, data [][]string, maxSize int) (int, []int) {
	var result = make([]int, len(headers))
	for i := 0; i < len(result); i++ {
		result[i] = len(headers[i]) + 1
	}
	for _, row := range data {
		for i := 0; i < len(row); i++ {
			cellLength := len(row[i]) + 1
			if cellLength > result[i] {
				result[i] = cellLength
			}
		}
	}
	var rowSize = 2
	for i := 0; i < len(result); i++ {
		rowSize += result[i]
	}
	if rowSize > maxSize {
		rowSize -= result[len(result)-1]
		remainder := maxSize - rowSize
		if remainder > 10 {
			result[len(result)-1] = remainder - 1
		}
	}
	rowSize = 1
	for i := 0; i < len(result); i++ {
		rowSize += result[i]
	}
	return rowSize, result
}

func (r *Renderer) printTableHeader(caption string, headers []string, data [][]string, maxSize int) {
	columnLength, columnSizes := r.getColumnSize(headers, data, maxSize)
	r.Println(r.ColorText(caption, "bold"))
	r.printRowSeparator(columnLength)
	r.Print("|")
	for i, size := range columnSizes {
		var header = fmt.Sprintf("%"+toolbox.AsString(size-1)+"s", headers[i])
		r.Print(r.ColorText(header, "bold"))
		r.Print("|")
	}
	r.Println("")
}

func (r *Renderer) printRowSeparator(length int) {
	for i := 0; i < length; i++ {
		r.Print("-")
	}
	r.Println("")
}

func (r *Renderer) normalizeCellData(data []string, columnSize []int) ([][]string, int) {
	var result = make([][]string, len(columnSize))
	var maxDepth = 1
	for i := 0; i < len(columnSize); i++ {
		if len(data[i]) <= columnSize[i] {
			result[i] = []string{data[i]}
		} else {
			result[i] = make([]string, 0)
			words := strings.Split(data[i], " ")
			text := words[0]
			for j := 1; j < len(words); j++ {
				if len(text+words[j]) > (columnSize[i] - 1) {
					result[i] = append(result[i], text)
					text = ""
				}
				if len(text) > 0 {
					text += " "
				}
				text += words[j]
			}
			if len(text) > 0 {
				result[i] = append(result[i], text)
			}
			if maxDepth < len(result[i]) {
				maxDepth = len(result[i])
			}
		}

	}
	return result, maxDepth
}

func (r *Renderer) printTableData(caption string, headers []string, data [][]string, maxSize int) {
	columnLength, columnSizes := r.getColumnSize(headers, data, maxSize)
	r.printRowSeparator(columnLength)
	for _, row := range data {
		cellData, rowCount := r.normalizeCellData(row, columnSizes)

		for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
			r.Print("|")
			for columnIndex := 0; columnIndex < len(cellData); columnIndex++ {
				var size = columnSizes[columnIndex]
				var cell = ""
				if rowIndex < len(cellData[columnIndex]) {
					cell = cellData[columnIndex][rowIndex]
				}
				r.Print(fmt.Sprintf("%-"+toolbox.AsString(size-1)+"s|", cell))
			}
			r.Println("")
		}
		r.printRowSeparator(columnLength)
	}

}

//PrintTable prints supplied  table data
func (r *Renderer) PrintTable(caption string, headers []string, data [][]string, maxSize int) {
	r.printTableHeader(caption, headers, data, maxSize)
	r.printTableData(caption, headers, data, maxSize)
}

//NewRenderer creates a new renderer
func NewRenderer(writer io.Writer, minColumns int) *Renderer {
	return &Renderer{
		ErrorColor: "red",
		minColumns: minColumns,
		writer:     writer,
	}
}
