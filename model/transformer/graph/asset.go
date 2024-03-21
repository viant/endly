package graph

import (
	"context"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	"strings"
	"sync"
)

type Asset struct {
	Name string
	URI  string
	storage.Object
}

var AssetExtension = []string{".json", ".yaml", ".yml", ".txt", ""}

type AssetManager struct {
	baseURL string
	assets  map[string]*Asset
	fs      afs.Service
	options []storage.Option
	mux     sync.RWMutex
}

func (m *AssetManager) LoadAsset(ctx context.Context, loc string) (*Asset, error) {
	URL := loc
	if url.IsRelative(loc) {
		URL = url.Join(m.baseURL, loc)
	}

	for _, ext := range AssetExtension {
		m.mux.RLock()
		asset := m.assets[URL+ext]
		m.mux.RUnlock()
		if asset != nil {
			return asset, nil
		}
		if object, _ := m.fs.Object(ctx, URL+ext, m.options...); object != nil {
			URL := object.URL()
			if index := strings.LastIndex(URL, "."); index != -1 {
				URL = URL[:index]
			}

			m.assets[object.URL()] = asset
			return asset, nil
		}
	}
	return nil, fmt.Errorf("failed to load asset: %v", loc)
}
