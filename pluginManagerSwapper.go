package hotswap

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/edwingeng/slog"
)

var (
	minFreeDelay = time.Second * 15
)

type ReloadCallback func(newManager, oldManager *PluginManager) error

type PluginManagerSwapper struct {
	slog.Logger
	current atomic.Value

	opts struct {
		pluginDir      string
		newExt         func() interface{}
		reloadCallback ReloadCallback
		freeDelay      time.Duration
	}

	staticPlugins map[string]*StaticPlugin
	reloadCounter int64

	mu sync.Mutex
}

func NewPluginManagerSwapper(pluginDir string, opts ...Option) *PluginManagerSwapper {
	swapper := &PluginManagerSwapper{Logger: slog.NewConsoleLogger()}
	swapper.opts.pluginDir = pluginDir
	swapper.opts.freeDelay = time.Minute * 5
	for _, opt := range opts {
		opt(swapper)
	}
	return swapper
}

func (wo *PluginManagerSwapper) Current() *PluginManager {
	v := wo.current.Load()
	pluginManager, _ := v.(*PluginManager)
	return pluginManager
}

func (wo *PluginManagerSwapper) LoadPlugins(data interface{}) (Details, error) {
	wo.mu.Lock()
	defer wo.mu.Unlock()

	cbs := []ReloadCallback{wo.opts.reloadCallback}
	if wo.staticPlugins != nil {
		return wo.loadStaticPlugins(data, cbs)
	}

	return wo.loadPluginsImpl(data, cbs)
}

func (wo *PluginManagerSwapper) loadPluginsImpl(data interface{}, cbs []ReloadCallback) (Details, error) {
	var absDir string
	if err := hutils.FindDirectory(wo.opts.pluginDir, "pluginDir"); err != nil {
		return nil, err
	} else if absDir, err = filepath.Abs(wo.opts.pluginDir); err != nil {
		return nil, err
	}

	a, err := ioutil.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, fi := range a {
		if fi.IsDir() {
			continue
		}
		if strings.HasSuffix(fi.Name(), hutils.FileNameExt) {
			files = append(files, filepath.Join(absDir, fi.Name()))
		}
	}

	return wo.loadPluginFiles(files, data, cbs)
}

func (wo *PluginManagerSwapper) loadPluginFiles(files []string, data interface{}, cbs []ReloadCallback) (Details, error) {
	if len(files) == 0 {
		return nil, nil
	}

	oldManager := wo.Current()
	newManager := newPluginManager(wo.Logger, wo.opts.newExt)
	if err := newManager.loadPlugins(files, oldManager, data); err != nil {
		return nil, err
	}
	if err := invokeReloadCallbacks(cbs, newManager, oldManager); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, f := range files {
		p := newManager.FindPlugin(pluginName(f))
		if p.Note != "" {
			result[p.File] = p.Note
		} else {
			result[p.File] = "ok"
		}
	}
	if oldManager != nil {
		go func() {
			delay := minFreeDelay
			if minFreeDelay < wo.opts.freeDelay {
				delay = wo.opts.freeDelay
			}
			time.Sleep(delay)
			oldManager.invokeEveryOnFree()
		}()
	}

	wo.current.Store(newManager)
	return result, nil
}

func invokeReloadCallbacks(cbs []ReloadCallback, newManager, oldManager *PluginManager) error {
	for _, cb := range cbs {
		if cb == nil {
			continue
		}
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("<hotswap> panic: %+v\n%s", r, debug.Stack())
				}
			}()
			return cb(newManager, oldManager)
		}()
		if err != nil {
			newManager.invokeEveryOnFree()
			return err
		}
	}
	return nil
}

func (wo *PluginManagerSwapper) Reload(data interface{}) (Details, error) {
	return wo.ReloadWithCallback(data, nil)
}

func (wo *PluginManagerSwapper) ReloadWithCallback(data interface{}, extra ReloadCallback) (Details, error) {
	if wo.staticPlugins != nil {
		return nil, errors.New("running under static linking mode")
	}

	wo.mu.Lock()
	defer wo.mu.Unlock()
	cbs := []ReloadCallback{wo.opts.reloadCallback}
	if extra != nil {
		cbs = append(cbs, extra)
	}
	details, err := wo.loadPluginsImpl(data, cbs)
	if err == nil {
		atomic.AddInt64(&wo.reloadCounter, 1)
	}
	return details, err
}

func (wo *PluginManagerSwapper) ReloadCounter() int64 {
	return atomic.LoadInt64(&wo.reloadCounter)
}

type Details map[string]string

func (d Details) String() string {
	var a []string
	for k := range d {
		a = append(a, k)
	}
	sort.Strings(a)

	var buf bytes.Buffer
	for i, k := range a {
		if i > 0 {
			_, _ = buf.WriteString(", ")
		}
		x := strings.TrimSuffix(filepath.Base(k), hutils.FileNameExt)
		_, _ = fmt.Fprintf(&buf, "%s: %s", x, d[k])
	}
	return buf.String()
}

type Option func(mgr *PluginManagerSwapper)

func WithLogger(log slog.Logger) Option {
	return func(mgr *PluginManagerSwapper) {
		mgr.Logger = log
	}
}

func WithFreeDelay(d time.Duration) Option {
	return func(mgr *PluginManagerSwapper) {
		mgr.opts.freeDelay = d
	}
}

func WithReloadCallback(cb ReloadCallback) Option {
	return func(mgr *PluginManagerSwapper) {
		mgr.opts.reloadCallback = cb
	}
}

func WithExtensionNewer(newExt func() interface{}) Option {
	return func(mgr *PluginManagerSwapper) {
		mgr.opts.newExt = newExt
	}
}

func WithStaticPlugins(plugins map[string]*StaticPlugin) Option {
	return func(mgr *PluginManagerSwapper) {
		mgr.staticPlugins = plugins
	}
}
