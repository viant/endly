package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox"
	"strings"
	"time"
	"path"
	"github.com/viant/toolbox/url"
	"github.com/viant/neatly"
)

//WorkflowRegisterRequest represents workflow register request
type WorkflowRegisterRequest struct {
	Workflow *Workflow
}

