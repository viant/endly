package client

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly/cli"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	managerservice "github.com/viant/endly/service/manager"
)

func (c *Client) newEventListener() msg.Listener {
	writer := c.Output
	if !c.Color {
		writer = &cleanWriter{writer: writer}
	}
	return cli.NewEventListener(writer)
}

type cleanWriter struct{ writer io.Writer }

func (w *cleanWriter) Write(payload []byte) (int, error) {
	_, err := w.writer.Write([]byte(vtclean.Clean(string(payload), false)))
	return len(payload), err
}

func (c *Client) renderEvents(events []*managerservice.OperationEvent, after int64, listener msg.Listener) int64 {
	last := after
	for _, event := range events {
		if event == nil || event.Sequence <= after {
			continue
		}
		if event.Sequence > last {
			last = event.Sequence
		}
		var value interface{}
		switch {
		case event.Activity != nil:
			activity := *event.Activity
			if event.Activity.MetaTag != nil {
				meta := *event.Activity.MetaTag
				activity.MetaTag = &meta
			}
			value = &activity
		case event.ActivityEnd:
			value = &model.ActivityEndEvent{}
		case len(event.Messages) > 0:
			value = &msg.MessagesEvent{Reported: event.Messages}
		}
		if value != nil {
			transported := msg.NewEvent(value)
			transported.SetLoggable(true)
			listener(transported)
		}
	}
	return last
}

func (c *Client) renderSummary(operation *managerservice.Operation) {
	duration := time.Duration(0)
	if operation.StartedAt != nil && operation.FinishedAt != nil {
		duration = operation.FinishedAt.Sub(*operation.StartedAt)
	}
	statusColor := "32"
	if operation.Status != managerservice.OperationSucceeded {
		statusColor = "31"
	}
	fmt.Fprintf(c.Output, "\nOperation %s: %s (%s)\n", operation.ID, c.paint(operation.Status, statusColor, "1"), duration.Round(time.Millisecond))
	if operation.Error != "" {
		fmt.Fprintln(c.Output, c.paint("Error: "+operation.Error, "31"))
	}
	if operation.Result != nil {
		c.renderValue("Result", operation.Result, "32")
	}
}

func (c *Client) renderValue(label string, value interface{}, color string) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(c.Output, c.paint(fmt.Sprintf("%s: %v", label, value), color))
		return
	}
	text := string(data)
	if !strings.Contains(text, "\n") {
		fmt.Fprintln(c.Output, c.paint(fmt.Sprintf("%s: %s", label, text), color))
		return
	}
	fmt.Fprintln(c.Output, c.paint(label+":", color))
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintln(c.Output, c.paint("        "+line, color))
	}
}

func (c *Client) paint(text string, codes ...string) string {
	if !c.Color || len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}
