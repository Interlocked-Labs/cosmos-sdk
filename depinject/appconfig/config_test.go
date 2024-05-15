package appconfig_test

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"testing"

	"gotest.tools/v3/assert"

	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	internal "cosmossdk.io/depinject/internal/appconfig"
	"cosmossdk.io/depinject/internal/appconfig/testpb"
)

func expectContainerErrorContains(t *testing.T, option depinject.Config, contains string) {
	t.Helper()
	err := depinject.Inject(option)
	assert.ErrorContains(t, err, contains)
}

func TestCompose(t *testing.T) {
	opt := appconfig.LoadJSON([]byte(`{"modules":[{}]}`))
	expectContainerErrorContains(t, opt, "module is missing name")

	opt = appconfig.LoadJSON([]byte(`{"modules":[{"name": "a"}]}`))
	expectContainerErrorContains(t, opt, `module "a" is missing a config object`)

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": testpb.ModuleFoo
`))
	expectContainerErrorContains(t, opt, `unable to resolve`)

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": cosmos.app.v1alpha1.Config # this is not actually a module config type!
`))
	expectContainerErrorContains(t, opt, "does not have the option cosmos.app.v1alpha1.module")
	expectContainerErrorContains(t, opt, "registered modules are")
	expectContainerErrorContains(t, opt, "testpb.TestModuleA")

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": testpb.TestUnregisteredModule
`))
	expectContainerErrorContains(t, opt, "did you forget to import cosmossdk.io/core/internal/testpb")
	expectContainerErrorContains(t, opt, "registered modules are")
	expectContainerErrorContains(t, opt, "testpb.TestModuleA")

	var app App
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: runtime
  config:
   "@type": testpb.TestRuntimeModule
- name: a
  config:
   "@type": testpb.TestModuleA
- name: b
  config:
   "@type": testpb.TestModuleB
`))
	assert.NilError(t, depinject.Inject(opt, &app))
	buf := &bytes.Buffer{}
	app(buf)
	const expected = `got store key a
got store key b
running module handler a
result: hello
running module handler b
result: goodbye
`
	assert.Equal(t, expected, buf.String())

	opt = appconfig.LoadYAML([]byte(`
golang_bindings:
  - interfaceType: interfaceType/package.name 
    implementation: implementationType/package.name
  - interfaceType: interfaceType/package.nameTwo 
    implementation: implementationType/package.nameTwo
modules:
  - name: a
    config:
      "@type": testpb.TestModuleA
    golang_bindings:
      - interfaceType: interfaceType/package.name 
        implementation: implementationType/package.name
      - interfaceType: interfaceType/package.nameTwo 
        implementation: implementationType/package.nameTwo
`))
	assert.NilError(t, depinject.Inject(opt))

	// module registration failures:
	appconfig.RegisterModule(&testpb.TestNoModuleOptionModule{})
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
   "@type": testpb.TestNoGoImportModule
`))
	expectContainerErrorContains(t, opt, "module should have the option cosmos.app.v1alpha1.module")

	internal.ModuleRegistry = map[reflect.Type]*internal.ModuleInitializer{} // reset module registry
	appconfig.RegisterModule(&testpb.TestNoGoImportModule{})
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
   "@type": testpb.TestNoGoImportModule
`))
	expectContainerErrorContains(t, opt, "module should have ModuleDescriptor.go_import specified")
}

func TestComposeGogoTypes(t *testing.T) {
	opt := appconfig.LoadJSON([]byte(`{"modules":[{}]}`))
	expectContainerErrorContains(t, opt, "module is missing name")

	opt = appconfig.LoadJSON([]byte(`{"modules":[{"name": "a"}]}`))
	expectContainerErrorContains(t, opt, `module "a" is missing a config object`)

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": testpb.ModuleFoo
`))
	expectContainerErrorContains(t, opt, `unable to resolve`)

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": cosmos.app.v1alpha1.Config # this is not actually a module config type!
`))
	expectContainerErrorContains(t, opt, "does not have the option cosmos.app.v1alpha1.module")
	expectContainerErrorContains(t, opt, "registered modules are")
	expectContainerErrorContains(t, opt, "testpb.TestModuleA")

	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
    "@type": testpb.TestGogoUnregisteredModule
`))
	expectContainerErrorContains(t, opt, "did you forget to import cosmossdk.io/core/internal/testpb")
	expectContainerErrorContains(t, opt, "registered modules are")
	expectContainerErrorContains(t, opt, "testpb.TestModuleA")

	var app App
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: runtime
  config:
   "@type": testpb.TestGogoRuntimeModule
- name: a
  config:
   "@type": testpb.TestGogoModuleA
- name: b
  config:
   "@type": testpb.TestGogoModuleB
`))
	assert.NilError(t, depinject.Inject(opt, &app))
	buf := &bytes.Buffer{}
	app(buf)
	const expected = `got store key a
got store key b
running module handler a
result: hello
running module handler b
result: goodbye
`
	assert.Equal(t, expected, buf.String())

	opt = appconfig.LoadYAML([]byte(`
golang_bindings:
  - interfaceType: interfaceType/package.name 
    implementation: implementationType/package.name
  - interfaceType: interfaceType/package.nameTwo 
    implementation: implementationType/package.nameTwo
modules:
  - name: a
    config:
      "@type": testpb.TestGogoModuleA
    golang_bindings:
      - interfaceType: interfaceType/package.name 
        implementation: implementationType/package.name
      - interfaceType: interfaceType/package.nameTwo 
        implementation: implementationType/package.nameTwo
`))
	assert.NilError(t, depinject.Inject(opt))

	// module registration failures:
	appconfig.RegisterModule(&testpb.TestGogoNoModuleOptionModule{})
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
   "@type": testpb.TestNoGoImportModule
`))
	expectContainerErrorContains(t, opt, "module should have the option cosmos.app.v1alpha1.module")

	internal.ModuleRegistry = map[reflect.Type]*internal.ModuleInitializer{} // reset module registry
	appconfig.RegisterModule(&testpb.TestGogoNoGoImportModule{})
	opt = appconfig.LoadYAML([]byte(`
modules:
- name: a
  config:
   "@type": testpb.TestNoGoImportModule
`))
	expectContainerErrorContains(t, opt, "module should have ModuleDescriptor.go_import specified")
}

//
// Test Module Initialization Logic
//

func init() {
	appconfig.RegisterModule(&testpb.TestGogoRuntimeModule{},
		appconfig.Provide(ProvideRuntimeState, ProvideStoreKey, ProvideApp),
	)

	appconfig.RegisterModule(&testpb.TestGogoModuleA{},
		appconfig.Provide(ProvideModuleA),
	)

	appconfig.RegisterModule(&testpb.TestGogoModuleB{},
		appconfig.Provide(ProvideModuleB),
	)

	appconfig.RegisterModule(&testpb.TestRuntimeModule{},
		appconfig.Provide(ProvideRuntimeState, ProvideStoreKey, ProvideApp),
	)

	appconfig.RegisterModule(&testpb.TestModuleA{},
		appconfig.Provide(ProvideModuleA),
	)

	appconfig.RegisterModule(&testpb.TestModuleB{},
		appconfig.Provide(ProvideModuleB),
	)
}

func ProvideRuntimeState() *RuntimeState {
	return &RuntimeState{}
}

func ProvideStoreKey(key depinject.ModuleKey, state *RuntimeState) StoreKey {
	sk := StoreKey{name: key.Name()}
	state.storeKeys = append(state.storeKeys, sk)
	return sk
}

func ProvideApp(state *RuntimeState, handlers map[string]Handler) App {
	return func(w io.Writer) {
		sort.Slice(state.storeKeys, func(i, j int) bool {
			return state.storeKeys[i].name < state.storeKeys[j].name
		})

		for _, key := range state.storeKeys {
			_, _ = fmt.Fprintf(w, "got store key %s\n", key.name)
		}

		var modNames []string
		for modName := range handlers {
			modNames = append(modNames, modName)
		}

		sort.Strings(modNames)
		for _, name := range modNames {
			_, _ = fmt.Fprintf(w, "running module handler %s\n", name)
			_, _ = fmt.Fprintf(w, "result: %s\n", handlers[name].DoSomething())
		}
	}
}

type App func(writer io.Writer)

type RuntimeState struct {
	storeKeys []StoreKey
}

type StoreKey struct{ name string }

type Handler struct {
	DoSomething func() string
}

func (h Handler) IsOnePerModuleType() {}

func ProvideModuleA(key StoreKey) (KeeperA, Handler) {
	return keeperA{key: key}, Handler{DoSomething: func() string {
		return "hello"
	}}
}

type keeperA struct {
	key StoreKey
}

type KeeperA interface {
	Foo()
}

func (k keeperA) Foo() {}

func ProvideModuleB(key StoreKey, a KeeperA) (KeeperB, Handler) {
	return keeperB{key: key, a: a}, Handler{
		DoSomething: func() string {
			return "goodbye"
		},
	}
}

type keeperB struct {
	key StoreKey
	a   KeeperA
}

type KeeperB interface {
	isKeeperB()
}

func (k keeperB) isKeeperB() {}
