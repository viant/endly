package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/common"
	"testing"
)

func Test_Expand(t *testing.T) {

	state := common.NewMap()
	state.Put("name", "etly")
	build := common.NewMap()
	state.Put("build", build)
	build.Put("Target", "app")
	build.Put("Args", "-Dmvn.test.skip")

	var text = endly.ExpandAsText(state, "a $vv-ee /usr/local/app_${name}v1 $build.Target $abc $build.Args")
	assert.Equal(t, "a $vv-ee /usr/local/app_etlyv1 app $abc -Dmvn.test.skip", text)

}

func Test_ExpandFun(t *testing.T) {

	state := common.NewMap()
	state.Put("name", "etly")
	build := common.NewMap()
	state.Put("build", build)
	build.Put("Target", "app")
	build.Put("Args", "-Dmvn.test.skip")

	var text = endly.ExpandAsText(state, "a $vv-ee /usr/local/app_${name}v1 $build.Target $abc $build.Args")
	assert.Equal(t, "a $vv-ee /usr/local/app_etlyv1 app $abc -Dmvn.test.skip", text)

}
