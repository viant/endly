package run

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_generateRandomASCII(t *testing.T) {

	text := generateRandomASCII(10)
	assert.Equal(t, len(text), 10)
	fmt.Printf("%v\n", text)
}
