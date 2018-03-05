package endly

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestEvent_Type(t *testing.T) {
	{
		var event = NewEvent(NewSleepEvent(100))
		assert.EqualValues(t, "endly_SleepEvent", event.Type())
	}
	{
		var event = NewEvent(os.File{})
		assert.EqualValues(t, "os_File", event.Type())
	}
	{
		var value *os.File = nil
		var event = NewEvent(value)
		assert.EqualValues(t, "os_File", event.Type())
	}
	{
		var value interface{} = nil
		var event = NewEvent(value)
		assert.EqualValues(t, "<nil>", event.Type())
	}
}
