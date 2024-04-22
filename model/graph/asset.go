package graph

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	"path"
	"strings"
	"sync"
)

// Asset represents an asset
type Asset struct {
	Name string
	URI  string
	storage.Object
}

var assetExtension = []string{".json", ".yaml", ".yml", ".txt", ""}

type AssetManager struct {
	baseURL  string
	basePath string
	assets   map[string]*Asset
	fs       afs.Service
	options  []storage.Option
	mux      sync.RWMutex
}

func (m *AssetManager) LoadWorkflow(ctx context.Context, URL string) *Asset {
	m.mux.Lock()
	defer m.mux.Unlock()
	asset, err := m.loadAsset(ctx, URL)
	if err != nil {
		return nil
	}
	m.assets[URL] = &Asset{URI: URL}
	parent, _ := url.Split(URL, file.Scheme)
	m.assets[parent] = &Asset{URI: parent}
	return asset
}

func (m *AssetManager) LoadAsset(ctx context.Context, loc string) (*Asset, bool, error) {
	URL := loc
	if url.IsRelative(loc) {
		URL = url.Join(m.baseURL, loc)
	}
	extensions := assetExtension
	if path.Ext(loc) != "" {
		extensions = []string{""}
	}
	for _, ext := range extensions {
		candidate := URL + ext
		m.mux.RLock()
		asset := m.assets[candidate]
		m.mux.RUnlock()
		if asset != nil {
			return asset, false, nil
		}
		if asset, _ = m.loadAsset(ctx, candidate); asset != nil {
			return asset, true, nil
		}
	}
	return nil, false, fmt.Errorf("failed to load asset: %v", loc)
}

func (m *AssetManager) loadAsset(ctx context.Context, candidate string) (*Asset, error) {
	object, err := m.fs.Object(ctx, candidate, m.options...)
	if err != nil {
		return nil, err
	}

	URL := object.URL()
	aPath := url.Path(URL)
	if index := strings.Index(aPath, m.basePath); index != -1 {
		aPath = aPath[index+len(m.basePath)+1:]
	}

	asset := &Asset{URI: aPath, Object: object}
	name := object.Name()
	if ext := path.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}
	asset.Name = name
	m.assets[object.URL()] = asset
	return asset, nil
}

func NewAssetManager(baseURL string, opts ...storage.Option) *AssetManager {
	return &AssetManager{
		baseURL:  baseURL,
		options:  opts,
		basePath: url.Path(baseURL),
		assets:   make(map[string]*Asset),
		fs:       afs.New(),
	}
}
