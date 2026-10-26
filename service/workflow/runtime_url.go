package workflow

import (
	"fmt"
	"path"
	"strings"

	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox"
)

func runtimeWorkflowURI(request *RunRequest) string {
	source := request.requestedURL
	if source == "" {
		source = request.URL
	}
	resolved := model.WorkflowSelector(source).URL()
	if path.Ext(source) == "" {
		resolved = strings.TrimSuffix(resolved, ".csv")
	}
	return resolved
}

func runtimeWorkflowURLExplicit(request *RunRequest) bool {
	source := runtimeWorkflowURI(request)
	if strings.Contains(source, "/") {
		return true
	}
	switch strings.ToLower(path.Ext(source)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func (s *Service) loadRuntimeWorkflow(ctx *endly.Context, request *RunRequest) (*model.Workflow, error) {
	state := ctx.State()
	source := state.ExpandAsText(runtimeWorkflowURI(request))
	if source == "" || strings.Contains(source, "$") {
		return nil, fmt.Errorf("load runtime workflow %q: unresolved source", source)
	}
	// Use the canonical resource decoder and the existing Inlined.AsWorkflow
	// compiler. Its @request expansion owns instance -> default -> resource
	// fallback; a runtime URL never invents a mandatory per-instance test file.
	bases := []string{}
	if parent := LastWorkflow(ctx); parent != nil && parent.Source != nil {
		base, _ := toolbox.URLSplit(parent.Source.URL)
		bases = append(bases, toolbox.URLPathJoin(base, "default"), base)
	} else if ctx.Source != nil {
		base, _ := toolbox.URLSplit(ctx.Source.URL)
		bases = append(bases, toolbox.URLPathJoin(base, "default"), base)
	}
	bases = append(bases, "")
	loaded := &RunRequest{}
	resource, err := util.LoadResourceFromBaseURLs(bases, "@"+source, loaded)
	if err != nil {
		return nil, fmt.Errorf("load runtime workflow %q: %w", source, err)
	}
	loaded.Name, loaded.AssetURL, loaded.Source = request.Name, resource.URL, resource
	if err := loaded.Init(); err != nil {
		return nil, fmt.Errorf("compile runtime workflow %q: %w", resource.URL, err)
	}
	if loaded.workflow == nil {
		return nil, fmt.Errorf("runtime workflow %q has no pipeline", resource.URL)
	}
	return loaded.workflow, nil
}
