![Banner](imgs/banner.jpg?raw=true "Hotswap")

[简体中文版](./README.zh-CN.md)

*`Hotswap`* provides you a complete solution to reload your `go` code without restarting your server, interrupting or blocking any ongoing procedure. *`Hotswap`* is built upon the plugin mechanism.

# Major Features

- Reload your code like a breeze
- Run different versions of a plugin in complete isolation
- Invoke an in-plugin function from its host program with `Plugin.InvokeFunc()`
- Expose in-plugin data and functions with `PluginManager.Vault.Extension` and/or `PluginManager.Vault.DataBag`
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
hotswap build --staticLinking plugin/foo plugin

Flags:
      --debug               enable debug mode
      --exclude string      go-regexp matching files to exclude from included
      --goBuild             if --goBuild=false, skip the go build procedure (default true)
  -h, --help                help for build
      --include string      go-regexp matching files to include in addition to .go files
      --leaveTemps          do not delete temporary files
      --livePrefix string   case-insensitive name prefix of live functions and live types (default "live_")
      --staticLinking       generate code for static linking instead of building a plugin
  -v, --verbose             enable verbose mode
```

# Demos

You can find these examples under the `demo` directory. To have a direct experience, start a server with `run.sh` and reload its plugin(s) with `reload.sh`.

1. `hello` demonstrates the basic usage, including how to organize host and plugin, how to build them, how to load plugin on server startup, how to use `InvokeEach`, and how to reload.
2. `extension` shows how to define a custom extension and how to use `PluginManager.Vault.Extension`. A small hint: `WithExtensionNewer()`
3. `livex` is somewhat complex. It shows how to work with `live function`, `live type`, and `live data`.
4. `slink` is an example of plugin static-linking, with which debugging a plugin with a debugger (delve) under MacOS and Windows becomes possible.
5. `trine` is the last example. It demonstrates the plugin dependency mechanism.

# Required Functions

A plugin must have the following functions defined in its root package.

``` go
// OnLoad gets called after all plugins are successfully loaded and before the Vault is initialized.
func OnLoad(data interface{}) error {
    return nil
}

// OnInit gets called after the execution of all OnLoad functions. The Vault is ready now.
func OnInit(sharedVault *vault.Vault) error {
    return nil
}

// OnFree gets called at some time after a reload.
func OnFree() {
}

// Export returns an object to export to other plugins.
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
5. Vault Initialization
6. OnInit
```

# Attentions

- Build your host program with the environmental variable `CGO_ENABLED=1` and the `-trimpath` flag.
- Do **not** define any global variable in a reloadable plugin unless it can be discarded at any time or it actually never changes.
- Do **not** create any long-running goroutine in a plugin, it's error-prone.
- The same type in different versions of a plugin are actually **not** the same at runtime. Use `live function`, `live type`, and `live data` to avoid the trap.
- The code of your host program should **never** import any package of any plugin and the code of a plugin should **never** import any package of other plugins.
- Old versions won't be removed from the memory due to the limitation of golang plugin. However, *`Hotswap`* offers you a chance, the `OnFree` function, to clear caches.
- It is required to manage your code with `git` and `go module`.
- It is highly recommended to keep the code of your host program and all its plugins in a same repository.

# Live Things

- `live function` is a type of function whose name is prefixed with `live_` (case-insensitive). Live functions are automatically collected and stored in `PluginManager.Vault.LiveFuncs`. For example:
``` go
func live_Foo(jobData live.Data) error {
      return nil
}
```
- `live type` is a type of struct whose name is prefixed with `live_` (case-insensitive). Live types are automatically collected and stored in `PluginManager.Vault.LiveTypes`. For example:
``` go
type Live_Bar struct {
      N int
}
```
- [`live data`](https://github.com/edwingeng/live) is a type guardian. You can convert your data into a `live data` object when scheduling an asynchronous job and restore your data from the `live data` object when handling the job.
- See the demo `livex` for details.

# FAQ

- **How can I debug a plugin with a debugger?**

Build it with `--staticLink`. For more information, please refer to the demo `slink`.

- **Does `hotswap` work on Windows?**

Building with `--staticLink` works on Windows, but plugin reloading is not an option because Go's plugin mechanism doesn't support Windows.


&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
&nbsp;<br/>
