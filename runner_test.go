package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"log"
	"os/exec"
	"github.com/viant/toolbox"
)

func TestCliRunner_Run(t *testing.T) {

	exec.Command("rm", "-rf", "/tmp/endly/test/workflow/dsunit").CombinedOutput()
	toolbox.CreateDirIfNotExist("/tmp/endly/test/workflow/dsunit")
	runner := endly.NewCliRunner()
	err := runner.Run("test/runner/run.json")
	if err != nil {
		log.Fatal(err)
	}
}