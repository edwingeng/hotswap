package hotswap

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"plugin"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/edwingeng/hotswap/vault"
	"go.uber.org/atomic"
)

var (
	ErrNotExist = errors.New("symbol does not exist")
)

type PluginFuncs struct {
	fOnLoad func(data interface{}) error
	fOnInit func(sharedVault *vault.Vault) error
	fOnFree func()

	fExport     func() interface{}
	fImport     func() interface{}
	InvokeFunc  func(name string, params ...interface{}) (interface{}, error)
	fReloadable func() bool

	hotswapLiveFuncs func() map[string]interface{}
	hotswapLiveTypes func() map[string]func() interface{}
}

type Plugin struct {
	Name      string
	File      string
	FileSha1  [sha1.Size]byte
	When      time.Time
	Note      string
	unchanged bool

	P           *plugin.Plugin `json:"-"`
	PluginFuncs `json:"-"`
	Deps        []string
	Refs        *atomic.Int64 `json:"-"`
	exported    interface{}
	reloadable  bool

	freeOnce *sync.Once
}

func newPlugin() *Plugin {
	return &Plugin{
		Refs:     atomic.NewInt64(1),
		freeOnce: &sync.Once{},
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	switch vv := reflect.ValueOf(v); vv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return vv.IsNil()
	default:
		return false
	}
}

func (wo *Plugin) Lookup(symName string, out interface{}) error {
	if symName == "" {
		return fmt.Errorf("symName cannot be empty. plugin: %s", wo.Name)
	}
	if isNil(out) {
		return fmt.Errorf("out cannot be nil. plugin: %s, symName: %s", wo.Name, symName)
	}

	outVal := reflect.ValueOf(out)
	if k := outVal.Type().Kind(); k != reflect.Ptr {
		return fmt.Errorf("out must be a pointer. plugin: %s, symName: %s", wo.Name, symName)
	}
	sym, err := wo.P.Lookup(symName)
	if err != nil {
		return ErrNotExist
	}

	symVal := reflect.ValueOf(sym)
	symTyp := symVal.Type()
	ele := outVal.Elem()
	eleTyp := ele.Type()
	switch {
	case symTyp.AssignableTo(eleTyp):
		ele.Set(symVal)
		return nil
	case symTyp.Kind() == reflect.Ptr && symTyp.Elem().AssignableTo(eleTyp):
		ele.Set(symVal.Elem())
		return nil
	default:
		return fmt.Errorf("failed to assign %s to out. plugin: %s, symTyp: %s, outTyp: %s",
			symName, wo.Name, symTyp.String(), outVal.Type().String())
	}
}

func (wo *Plugin) invokeExport() (_ interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("<hotswap:%s> panic: %+v\n%s", wo.Name, r, debug.Stack())
		}
	}()
	return wo.fExport(), nil
}

func (wo *Plugin) invokeImport() (_ interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("<hotswap:%s> panic: %+v\n%s", wo.Name, r, debug.Stack())
		}
	}()
	return wo.fImport(), nil
}

func (wo *Plugin) invokeReloadable() (_ bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("<hotswap:%s> panic: %+v\n%s", wo.Name, r, debug.Stack())
		}
	}()
	return wo.fReloadable(), nil
}
