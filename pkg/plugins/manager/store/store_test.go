package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/backendplugin"
	"github.com/grafana/grafana/pkg/plugins/manager/fakes"
)

func TestStore_ProvideService(t *testing.T) {
	t.Run("Plugin sources are added in order", func(t *testing.T) {
		var addedPaths []string
		l := &fakes.FakeLoader{
			LoadFunc: func(ctx context.Context, src plugins.PluginSource) ([]*plugins.Plugin, error) {
				addedPaths = append(addedPaths, src.PluginURIs(ctx)...)
				return nil, nil
			},
		}

		srcs := &fakes.FakeSourceRegistry{ListFunc: func(_ context.Context) []plugins.PluginSource {
			return []plugins.PluginSource{
				&fakes.FakePluginSource{
					PluginClassFunc: func(ctx context.Context) plugins.Class {
						return plugins.ClassBundled
					},
					PluginURIsFunc: func(ctx context.Context) []string {
						return []string{"path1"}
					},
				},
				&fakes.FakePluginSource{
					PluginClassFunc: func(ctx context.Context) plugins.Class {
						return plugins.ClassExternal
					},
					PluginURIsFunc: func(ctx context.Context) []string {
						return []string{"path2", "path3"}
					},
				},
			}
		}}

		s, err := ProvideService(fakes.NewFakePluginRegistry(), srcs, l)
		_ = s.StartAsync(context.Background())
		_ = s.AwaitRunning(context.Background())
		defer s.StopAsync()

		require.NoError(t, err)
		require.Equal(t, []string{"path1", "path2", "path3"}, addedPaths)
	})
}

func TestStore_Plugin(t *testing.T) {
	t.Run("Plugin returns all non-decommissioned plugins", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-datasource"}}
		p1.RegisterClient(&DecommissionedPlugin{})
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-panel"}}

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		p, exists := ps.Plugin(context.Background(), p1.ID)
		require.False(t, exists)
		require.Equal(t, plugins.PluginDTO{}, p)

		p, exists = ps.Plugin(context.Background(), p2.ID)
		require.True(t, exists)
		require.Equal(t, p, p2.ToDTO())
	})
}

func TestStore_Plugins(t *testing.T) {
	t.Run("Plugin returns all non-decommissioned plugins by type", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "a-test-datasource", Type: plugins.TypeDataSource}}
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "b-test-panel", Type: plugins.TypePanel}}
		p3 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "c-test-panel", Type: plugins.TypePanel}}
		p4 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "d-test-app", Type: plugins.TypeApp}}
		p5 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "e-test-panel", Type: plugins.TypePanel}}
		p5.RegisterClient(&DecommissionedPlugin{})

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
				p3.ID: p3,
				p4.ID: p4,
				p5.ID: p5,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		pss := ps.Plugins(context.Background())
		require.Equal(t, pss, []plugins.PluginDTO{p1.ToDTO(), p2.ToDTO(), p3.ToDTO(), p4.ToDTO()})

		pss = ps.Plugins(context.Background(), plugins.TypeApp)
		require.Equal(t, pss, []plugins.PluginDTO{p4.ToDTO()})

		pss = ps.Plugins(context.Background(), plugins.TypePanel)
		require.Equal(t, pss, []plugins.PluginDTO{p2.ToDTO(), p3.ToDTO()})

		pss = ps.Plugins(context.Background(), plugins.TypeDataSource)
		require.Equal(t, pss, []plugins.PluginDTO{p1.ToDTO()})

		pss = ps.Plugins(context.Background(), plugins.TypeDataSource, plugins.TypeApp, plugins.TypePanel)
		require.Equal(t, pss, []plugins.PluginDTO{p1.ToDTO(), p2.ToDTO(), p3.ToDTO(), p4.ToDTO()})
	})
}

func TestStore_Routes(t *testing.T) {
	t.Run("Routes returns all static routes for non-decommissioned plugins", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "a-test-renderer", Type: plugins.TypeRenderer}, FS: fakes.NewFakePluginFiles("/some/dir")}
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "b-test-panel", Type: plugins.TypePanel}, FS: fakes.NewFakePluginFiles("/grafana/")}
		p3 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "c-test-secrets", Type: plugins.TypeSecretsManager}, FS: fakes.NewFakePluginFiles("./secrets"), Class: plugins.ClassCore}
		p4 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "d-test-datasource", Type: plugins.TypeDataSource}, FS: fakes.NewFakePluginFiles("../test")}
		p5 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "e-test-app", Type: plugins.TypeApp}, FS: fakes.NewFakePluginFiles("any/path")}
		p6 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "f-test-app", Type: plugins.TypeApp}}
		p6.RegisterClient(&DecommissionedPlugin{})

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
				p3.ID: p3,
				p4.ID: p4,
				p5.ID: p5,
				p6.ID: p6,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		sr := func(p *plugins.Plugin) *plugins.StaticRoute {
			return &plugins.StaticRoute{PluginID: p.ID, Directory: p.FS.Base()}
		}

		rs := ps.Routes()
		require.Equal(t, []*plugins.StaticRoute{sr(p1), sr(p2), sr(p4), sr(p5)}, rs)
	})
}

func TestStore_Renderer(t *testing.T) {
	t.Run("Renderer returns a single (non-decommissioned) renderer plugin", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-renderer", Type: plugins.TypeRenderer}}
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-panel", Type: plugins.TypePanel}}
		p3 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-app", Type: plugins.TypeApp}}

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
				p3.ID: p3,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		r := ps.Renderer(context.Background())
		require.Equal(t, p1, r)
	})
}

func TestStore_SecretsManager(t *testing.T) {
	t.Run("Renderer returns a single (non-decommissioned) secrets manager plugin", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-renderer", Type: plugins.TypeRenderer}}
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-panel", Type: plugins.TypePanel}}
		p3 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-secrets", Type: plugins.TypeSecretsManager}}
		p4 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-datasource", Type: plugins.TypeDataSource}}

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
				p3.ID: p3,
				p4.ID: p4,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		r := ps.SecretsManager(context.Background())
		require.Equal(t, p3, r)
	})
}

func TestStore_availablePlugins(t *testing.T) {
	t.Run("Decommissioned plugins are excluded from availablePlugins", func(t *testing.T) {
		p1 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-datasource"}}
		p1.RegisterClient(&DecommissionedPlugin{})
		p2 := &plugins.Plugin{JSONData: plugins.JSONData{ID: "test-app"}}

		ps := New(&fakes.FakePluginRegistry{
			Store: map[string]*plugins.Plugin{
				p1.ID: p1,
				p2.ID: p2,
			},
		}, &fakes.FakeSourceRegistry{}, &fakes.FakeLoader{})

		aps := ps.availablePlugins(context.Background())
		require.Len(t, aps, 1)
		require.Equal(t, p2, aps[0])
	})
}

type DecommissionedPlugin struct {
	backendplugin.Plugin
}

func (p *DecommissionedPlugin) Decommission() error {
	return nil
}

func (p *DecommissionedPlugin) IsDecommissioned() bool {
	return true
}
