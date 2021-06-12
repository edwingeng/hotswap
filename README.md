*`Hotswap`* provides a solution for reloading your `go` code without either restarting your server or interrupting any ongoing procedure. *`Hotswap`* is built upon the plugin mechanism.

# Major Features

- Reload your code like a breeze
- Run different versions of a plugin in complete isolation
- Invoke plugin functions from its host program with `Plugin.InvokeFunc()`
- Expose in-plugin data and functions with `PluginManager.Vault.DataBag` and/or `PluginManager.Vault.Extension`
- Handle asynchronous jobs using the latest code with `live function`, `live type`, and `live data`
- Link plugins statically for easy debugging
- Expose functions to other plugins with `Export()`
- Depend on other plugins with `Import()`

# Getting Started

```
go install github.com/edwingeng/hotswap/cli/hotswap
```

# Build a Plugin from Source Code

```
Usage:
  hotswap build [flags] <pluginDir> <outputDir> -- [buildFlags]

Examples:
hotswap build plugin/foo bin
hotswap build -v plugin/foo bin -- -race
hotswap build --staticLinking plugin/foo pluginHost

Flags:
      --debug               enable the debug mode
      --exclude string      go-regexp matching files to exclude from included
      --goBuild             if --goBuild=false, skip the go build procedure (default true)
  -h, --help                help for build
      --include string      go-regexp matching files to include in addition to .go files
      --leaveTemps          do not delete temporary files
      --prefixLive string   the case-insensitive name prefix of live functions and live types (default "live_")
      --staticLinking       generate code for static linking instead of building a plugin
  -v, --verbose             enable the verbose mode
```

# Demos

You can find these examples under the `demo` directory. To have a direct experience, start a server with `run.sh` and reload its plugin(s) with `reload.sh`.

1. `hello` demonstrates the basic usage, including how to organize host program and plugin, how to build plugin, how to load plugin on server startup, how to use `InvokeEach`, and finally how to reload.
2. `extension` shows how to define a custom extension and how to use `PluginManager.Vault.Extension`. A small hint: `WithExtensionNewer()`
3. `livex` is somewhat complex. It shows how to work with `live function`, `live type`, and `live data`.
4. `slink` is an example of plugin static-linking, with which debugging a plugin with a debugger (delve) under MacOS and Windows becomes possible.
5. `trine` is the last example. It demonstrates the plugin dependency mechanism.

# Required Functions

A plugin must have the following functions defined in its root package.

``` go
// OnLoad gets called after all plugins are successfully loaded and all dependencies are properly initialized.
func OnLoad(data interface{}) error {
  return nil
}

// OnInit gets called after the execution of all OnLoad functions.
func OnInit(sharedVault *vault.Vault) error {
  return nil
}

// OnFree gets called at some time after a reload.
func OnFree() {
}

// Export returns an object to be exported to other plugins.
func Export() interface{} {
  return nil
}

// Import returns an object indicating the dependencies of the plugin.
func Import() interface{} {
  return nil
}

// InvokeFunc invokes the specified function.
func InvokeFunc(name string, params ...interface{}) (interface{}, error) {
  return nil, nil
}

// Reloadable indicates whether the plugin is reloadable.
func Reloadable() bool {
  return true
}
```

# Order of Execution during Plugin Reload

```
1. Reloadable
2. Export
3. Import
4. OnLoad
5. OnInit
```

# Attentions

- Build the host program with `CGO_ENABLED=1` and the `-trimpath` flag.
- Manage your code with `git` (other VCS are not supported yet).
- Do **not** define any global variable in a reloadable plugin unless it can be discarded at any time.
- Do **not** create any long-running goroutine in a plugin.
- The same struct in different versions of a plugin is actually **not** the same at runtime. Use `live function`, `live type`, and `live data` to avoid the trap.
- The code of your host program should **never** import any package of any plugin and the code of a plugin should **never** import any package of other plugins.
- Old versions won't be removed from the memory because of the limitation of golang plugin. However, *`Hotswap`* offers you a chance, the `OnFree` function, to clear data caches.
- It is highly recommended to keep the code of your host program and all its plugins in a same repository.

# Live Things

- `live function` is a type of function whose name is prefixed with `live_` (case-insensitive). Live functions are automatically collected and stored in `PluginManager.Vault.LiveFuncs`.
- `live type` is a type of struct whose name is prefixed with `live_` (case-insensitive). Live types are automatically collected and stored in `PluginManager.Vault.LiveTypes`.
- [`live data`](https://github.com/edwingeng/live) is a type guardian. Convert your data into a `live data` object when scheduling an asynchronous job and restore your data from the `live data` object when handling the job.
- See the demo `livex` for details.
