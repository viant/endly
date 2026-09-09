package mobile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"golang.org/x/term"
)

type replLineReader interface {
	ReadLine() (string, error)
	OwnsPrompt() bool
}

type scannerREPLLineReader struct {
	scanner *bufio.Scanner
}

func (r *scannerREPLLineReader) ReadLine() (string, error) {
	if r.scanner.Scan() {
		return r.scanner.Text(), nil
	}
	if err := r.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (r *scannerREPLLineReader) OwnsPrompt() bool { return false }

type terminalREPLLineReader struct {
	terminal *term.Terminal
}

func (r *terminalREPLLineReader) ReadLine() (string, error) { return r.terminal.ReadLine() }
func (r *terminalREPLLineReader) OwnsPrompt() bool          { return true }

type splitReadWriter struct {
	reader io.Reader
	writer io.Writer
}

func (r *splitReadWriter) Read(data []byte) (int, error)  { return r.reader.Read(data) }
func (r *splitReadWriter) Write(data []byte) (int, error) { return r.writer.Write(data) }

type terminalHistory struct {
	mu      sync.Mutex
	entries []string // newest first, matching term.History
	max     int
}

func newTerminalHistory(entries []string, max int) *terminalHistory {
	result := &terminalHistory{max: max}
	for _, entry := range entries {
		result.Add(entry)
	}
	return result
}

func (h *terminalHistory) Add(entry string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry == "" {
		return
	}
	h.entries = append([]string{entry}, h.entries...)
	if h.max > 0 && len(h.entries) > h.max {
		h.entries = h.entries[:h.max]
	}
}

func (h *terminalHistory) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.entries)
}

func (h *terminalHistory) At(index int) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.entries[index]
}

func newREPLLineReader(input io.Reader, output io.Writer, config REPLConfig, history []string) (replLineReader, io.Writer, func(), error) {
	inputFile, inputIsFile := input.(*os.File)
	outputFile, outputIsFile := output.(*os.File)
	if !inputIsFile || !outputIsFile || !term.IsTerminal(int(inputFile.Fd())) || !term.IsTerminal(int(outputFile.Fd())) {
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), 1_000_000)
		return &scannerREPLLineReader{scanner: scanner}, output, func() {}, nil
	}
	state, err := term.MakeRaw(int(inputFile.Fd()))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("enable raw REPL terminal: %w", err)
	}
	terminal := term.NewTerminal(&splitReadWriter{reader: input, writer: output}, config.Prompt)
	if width, height, sizeErr := term.GetSize(int(outputFile.Fd())); sizeErr == nil && width > 0 && height > 0 {
		_ = terminal.SetSize(width, height)
	}
	terminal.History = newTerminalHistory(history, config.MaxHistory)
	terminal.AutoCompleteCallback = func(line string, position int, key rune) (string, int, bool) {
		if key != '\t' {
			return line, position, false
		}
		completed, completedPosition, matches := CompleteREPLLine(line, position, config.Completions)
		if len(matches) > 1 && completed == line {
			_, _ = terminal.Write([]byte("\r\n" + strings.Join(matches, "  ") + "\r\n"))
		}
		return completed, completedPosition, true
	}
	terminal.SetBracketedPasteMode(true)
	restore := func() {
		terminal.SetBracketedPasteMode(false)
		_ = term.Restore(int(inputFile.Fd()), state)
	}
	return &terminalREPLLineReader{terminal: terminal}, terminal, restore, nil
}

var defaultREPLCompletions = []string{
	":clear-history", ":close", ":exit", ":find ", ":help", ":history", ":quit", ":screenshot", ":source", ":status", ":tree ",
	"app.", "device.", "expect(",
	".acceptAlert()", ".activateApp(", ".alertText()", ".attribute(", ".back()", ".backgroundApp(",
	".check()", ".clear()", ".click()", ".context()", ".contexts()", ".count()", ".dismissAlert()", ".doubleTap()", ".dragTo(",
	".enabled()", ".exists()", ".fill(", ".first()", ".getByAccessibilityId(", ".getByAndroidUiAutomator(", ".getByCSS(",
	".getByClassName(", ".getByID(", ".getByIOSClassChain(", ".getByIOSPredicate(", ".getByResourceId(", ".getByTestId(", ".getByText(", ".getByXPath(",
	".hideKeyboard()", ".home()", ".inputValue()", ".last()", ".launchApp(", ".locator(", ".longPress(", ".nth(",
	".orientation()", ".pressKey(", ".rect()", ".rotate(", ".scroll(", ".selected()", ".setContext(", ".setLocation(",
	".swipe(", ".tap()", ".terminateApp(", ".text()", ".type(", ".uncheck()", ".value()", ".waitForHidden(", ".waitForVisible(",
	".toBeEnabled(", ".toBeHidden(", ".toBeSelected(", ".toBeVisible(", ".toContain(", ".toEqual(", ".toExist(",
	".toHaveAttribute(", ".toHaveContext(", ".toHaveCount(", ".toHaveOrientation(", ".toHaveText(", ".toHaveValue(",
}

// CompleteREPLLine applies one completion at the current byte position and
// returns all matching candidates so an interactive terminal can display
// ambiguity without losing the partially typed command.
func CompleteREPLLine(line string, position int, additional []string) (string, int, []string) {
	if position < 0 || position > len(line) {
		return line, position, nil
	}
	prefix := line[:position]
	start := completionStart(prefix)
	fragment := prefix[start:]
	candidates := append([]string(nil), defaultREPLCompletions...)
	candidates = append(candidates, additional...)
	matches := make([]string, 0)
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, fragment) {
			matches = append(matches, candidate)
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return line, position, nil
	}
	replacement := matches[0]
	if len(matches) > 1 {
		replacement = longestCommonPrefix(matches)
		if len(replacement) <= len(fragment) {
			return line, position, matches
		}
	}
	completed := line[:start] + replacement + line[position:]
	return completed, start + len(replacement), matches
}

func completionStart(prefix string) int {
	trimmedStart := len(prefix) - len(strings.TrimLeft(prefix, " \t"))
	if trimmedStart < len(prefix) && prefix[trimmedStart] == ':' && !strings.ContainsAny(prefix[trimmedStart:], " \t") {
		return trimmedStart
	}
	if dot := strings.LastIndex(prefix, "."); dot >= 0 {
		return dot
	}
	if delimiter := strings.LastIndexAny(prefix, " \t=,("); delimiter >= 0 {
		return delimiter + 1
	}
	return 0
}

func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, result) && result != "" {
			result = result[:len(result)-1]
		}
	}
	return result
}
