package markdown

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ddddddO/gtree"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/project"
	"github.com/viant/endly/model/project/loader"
	"github.com/viant/endly/model/project/option"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
	"path"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	fs afs.Service
}

func (s *Service) Load(ctx context.Context, URL string, opts ...option.Option) (*project.Bundle, error) {
	options := option.NewOptions(opts...)
	content, err := s.fs.DownloadWithURL(ctx, URL, options.StorageOptions()...)
	if err != nil {
		return nil, err
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	if options.ProjectID == "" {
		options.ProjectID = "project"
	}
	reader := text.NewReader(content)
	root := md.Parser().Parse(reader)
	assets := newAssets()
	ts := time.Now().UnixMicro()
	baseURL := "mem://localhost/tmp/" + strconv.Itoa(int(ts))
	templateURL := url.Join(baseURL, options.ProjectID, "%s")
	defer s.fs.Delete(ctx, baseURL)
	err = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		status := ast.WalkStatus(ast.WalkContinue)
		switch n.Kind() {
		case ast.KindFencedCodeBlock:
			if prev := n.PreviousSibling(); prev != nil {
				value := strings.TrimSpace(string(prev.Text(content)))
				if path.Ext(value) != "" || strings.Contains(value, "/") {
					URL := fmt.Sprintf(templateURL, value)
					if !assets.shallUpload(URL) {
						return status, nil
					}
					err = s.loadFencedCode(ctx, n, URL, content)
					return status, err
				}
			}
		}
		return status, err
	})
	srv := loader.New()
	workflowURL := assets.rootWorkflowURL()
	bundle, err := srv.Load(ctx, workflowURL, option.WithEmbedFS(options.EmbedFS), option.WithDependencies(true), option.WithAssets(true))
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

func (s *Service) loadFencedCode(ctx context.Context, n ast.Node, URL string, content []byte) error {
	var lines []string
	segments := n.Lines()
	for i := 0; i < segments.Len(); i++ {
		segment := segments.At(i)
		line := segment.Value(content)
		lines = append(lines, string(line))
	}
	data := strings.Join(lines, "")
	if err := s.fs.Upload(ctx, URL, file.DefaultFileOsMode, strings.NewReader(data)); err != nil {
		return err
	}
	return nil
}

func (s *Service) workflowStructure(ctx context.Context, workflow *project.Workflow) (string, error) {

	root := gtree.NewRoot("")
	addNode := func(name string) {
		var node *gtree.Node
		splited := strings.Split(name, "/")
		for i, s := range splited {
			if i == 0 {
				node = root.Add(s)
				continue
			}
			node = node.Add(s)
		}
	}
	addNode(workflow.FileName())
	for _, asset := range workflow.Assets {
		addNode(asset.Location)
	}
	for _, asset := range workflow.Workflows {
		addNode(asset.URI)
	}
	buffer := &bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("## Workflow %s\n### Structure\n```text", workflow.Name))
	err := gtree.OutputProgrammably(buffer, root)
	if err != nil {
		return "", err
	}
	buffer.WriteString("```\n")
	return buffer.String(), nil
}

func (s *Service) Markdown(ctx context.Context, workflow *project.Workflow, opts ...option.Option) (string, error) {
	session := NewSession()
	return s.markdownWorkflow(ctx, workflow, session, opts)
}

func (s *Service) markdownWorkflow(ctx context.Context, workflow *project.Workflow, session *Session, opts []option.Option) (string, error) {
	builder := strings.Builder{}
	tree, err := s.workflowStructure(ctx, workflow)
	if err != nil {
		return "", err
	}
	builder.WriteString(tree)
	builder.WriteString("### Definition\n")
	builder.WriteString("#### " + workflow.URI + "\n")
	builder.WriteString("```yaml\n")
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return "", err
	}
	builder.Write(data)
	builder.WriteString("```\n")

	options := option.NewOptions(opts...)
	assetCount := 0
	if options.WithAssets {

		for _, asset := range workflow.Assets {
			if session.processedAssets[asset.Location] {
				continue
			}
			session.processedAssets[asset.Location] = true
			if asset.IsDir {
				continue
			}
			if assetCount == 0 {
				builder.WriteString("### Assets\n")
			}
			assetCount++
			ext := path.Ext(asset.Location)
			builder.WriteString("#### " + asset.Location + "\n")
			contentType := contentType(ext)
			builder.WriteString(fmt.Sprintf("```%s\n", contentType))
			builder.Write(asset.Source)
			if !bytes.HasSuffix(asset.Source, []byte("\n")) {
				builder.WriteString("\n")
			}
			builder.WriteString("```\n")
		}
	}
	if options.WithDependencies {
		for _, subWorkflow := range workflow.Workflows {
			data, err := s.markdownWorkflow(ctx, subWorkflow, session, opts)
			if err != nil {
				return "", err
			}
			builder.WriteString(data)
		}
	}
	return builder.String(), nil
}

func contentType(ext string) interface{} {
	ext = strings.Trim(ext, ".")
	switch ext {
	case "txt", "info":
		return "text"
	}
	return ext
}

func New() *Service {
	return &Service{fs: afs.New()}
}
