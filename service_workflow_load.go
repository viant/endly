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


//WorkflowLoadRequest represents workflow load request from the specified source
type WorkflowLoadRequest struct {
	Source *url.Resource
}


//WorkflowLoadResponse represents loaded workflow
type WorkflowLoadResponse struct {
	Workflow *Workflow
}
