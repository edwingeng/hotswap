![Banner](imgs/banner.jpg?raw=true "Hotswap")

*`Hotswap`* 为 `go` 语言代码热更提供了一套相当完整的解决方案，热更过程不会中断或阻塞任何执行中的函数，更不会重启服务器。此方案建立在 `go` 语言的 `plugin` 机制之上。

# 核心功能

- 轻松热更代码
- 完全隔离新老版本
- 通过 `Plugin.InvokeFunc()` 从宿主调用插件中的函数
- 通过 `PluginManager.Vault.DataBag` 和 `PluginManager.Vault.Extension` 对外暴露插件中的数据和函数
- 通过 `live function`, `live type` 和 `live data` 用最新代码执行异步任务
- 支持静态链接插件，以方便调试
- 通过 `Export()` 向其它插件暴露函数
- 通过 `Import()` 声明、建立对其它插件的依赖

# 安装

```
go install github.com/edwingeng/hotswap/cli/hotswap
```

# 编译插件

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
      --prefixLive string   the case-insensitive name prefix of live functions/types (default "live_")
      --staticLinking       generate code for static linking instead of building a plugin
  -v, --verbose             enable the verbose mode
```

# 示例

你可以在 `demo` 目录下找到这些例子。为了更直观的体验，运行 `run.sh` 启动服务器，再运行 `reload.sh` 热更插件。

1. `hello` 展示了这套方案的基本用法, 包括怎样组织宿主和插件、怎样编译宿主和插件、怎样在服务器启动时加载插件、怎样使用 `InvokeEach`、以及怎样热更。
2. `extension` 是个关于自定义扩展的例子，它可以告诉你 `PluginManager.Vault.Extension` 的用法。小提示: `WithExtensionNewer()`。
3. `livex` 比较复杂. 它展示了 `live function`, `live type` 和 `live data` 的用法。
4. `slink` 展示了静态链接的使用方法。在 MacOS 和 Windows 下，用静态链接才能上调试器（delve）调试。
5. `trine` 是最后一个例子，它展示了插件的依赖机制。

# 必须定义的函数

每个插件都要在其根 package 下定义以下函数：

``` go
// OnLoad gets called after all plugins are successfully loaded and all dependencies are
// properly initialized.
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

# 插件加载过程中上面函数的执行顺序

```
1. Reloadable
2. Export
3. Import
4. OnLoad
5. OnInit
```

# 注意事项

- 编译宿主程序时，要加上环境变量 `CGO_ENABLED=1`，并指定编译参数 `-trimpath`。
- 用 `git` 管理代码，其它 VCS 尚不支持。
- 不要在可热更的插件里定义全局变量，除非这些变量实际上从不改变，或其值可随时丢弃。
- 不要在插件里启动长时间运行的 goroutine。
- 小心插件里定义的类型，因为在运行时 `go` 认为不同插件版本中的同一类型是不同类型，数据实例无法相互赋值。你可以通过 `live function`, `live type` 和 `live data` 规避这一陷阱。
- 宿主代码不要 import 任何插件的任何 package；任何插件都不要 import 其它插件的任何 package。
- 热更后，旧版插件会继续留在内存中，永不释放，这是 `plugin` 的限制。不过你有个清理缓存的机会：`OnFree`。
- 强烈建议：用同一个代码仓库管理宿主程序和所有插件。

# Live Things

- `live function` 是以 `live_` 为名字前缀（大小写不敏感）的函数，所有这类函数都会被自动收集起来并存入 `PluginManager.Vault.LiveFuncs`。例如：
``` go
func live_Foo(jobData live.Data) error {
      return nil
}
```
- `live type` 是以 `live_` 为名字前缀（大小写不敏感）的（struct）类型，所有这类 struct 都会被自动收集起来并存入 `PluginManager.Vault.LiveTypes`。例如：
``` go
type Live_Bar struct {
      N int
}
```
- [`live data`](https://github.com/edwingeng/live) 是个类型隔离器。你可以在创建异步任务时把任务数据转成 `live data` 对象，再在执行该任务时把数据恢复回来。
- 例子 `livex` 包含更多细节。
