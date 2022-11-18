package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/edwingeng/hotswap/cli/hotswap/g"
	"github.com/edwingeng/hotswap/internal/hutils"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

var (
	hotswapBureauPackageNames = [...]string{
		"hotbureau",
		"hotswapbureau",
	}
)

const (
	hotswapBureauFile           = "hotswap.bureau.go"
	hotswapMainFile             = "hotswap.main.go"
	hotswapLiveFile             = "hotswap.live.go"
	hotswapStaticPluginInitFile = "hotswap.staticPluginInit.%s.go"
	hotswapStaticPluginsFile    = "hotswap.staticPlugins.go"
)

var hotswapFiles = map[string]struct{}{
	hotswapBureauFile: {},
	hotswapMainFile:   {},
	hotswapLiveFile:   {},
}

var (
	//go:embed hotswapBureau.tpl
	tplHotswapBureau []byte
	//go:embed hotswapMain.tpl
	tplHotswapMain string
	//go:embed hotswapLive.tpl
	tplHotswapLive string
	//go:embed hotswapStaticPluginInit.tpl
	tplHotswapStaticPluginInit string
	//go:embed hotswapStaticPlugins.tpl
	tplHotswapStaticPlugins string
)

var (
	haltProgram = make(chan struct{})
	timing      struct {
		copyFilesStart       time.Time
		copyFiles            time.Duration
		processPackagesStart time.Time
		processPackages      time.Duration
		buildStart           time.Time
		build                time.Duration
		totalStart           time.Time
		total                time.Duration
	}
)

var buildCmd buildCmdT

const (
	buildExamples = `hotswap build plugin/foo bin
hotswap build -v plugin/foo bin -- -race
hotswap build --staticLinking plugin/foo pluginHost`
)

var buildCmdCobra = &cobra.Command{
	Use:     "build [flags] <pluginDir> <outputDir> -- [buildFlags]",
	Short:   "Build a plugin",
	Example: buildExamples,
	Run:     buildCmd.execute,
}

func init() {
	rootCmd.AddCommand(buildCmdCobra)
	cmd := buildCmdCobra
	cmd.Flags().BoolVarP(&buildCmd.verbose,
		"verbose", "v", false, "enable the verbose mode")
	cmd.Flags().BoolVar(&buildCmd.leaveTemps,
		"leaveTemps", false, "do not delete temporary files")
	cmd.Flags().BoolVar(&buildCmd.goBuild,
		"goBuild", true, "if --goBuild=false, skip the go build procedure")
	cmd.Flags().BoolVar(&buildCmd.staticLinking,
		"staticLinking", false, "generate code for static linking instead of building a plugin")
	cmd.Flags().BoolVar(&buildCmd.cleanOnly,
		"cleanOnly", false, "clean static linking files, only")
	cmd.Flags().BoolVar(&buildCmd.debug,
		"debug", false, "enable the debug mode")
	cmd.Flags().StringVar(&buildCmd.livePrefix,
		"livePrefix", "live_", "the case-insensitive name prefix of live functions and live types")
	cmd.Flags().StringVar(&buildCmd.include,
		"include", "", "go-regexp matching files to include in addition to .go files")
	cmd.Flags().StringVar(&buildCmd.exclude,
		"exclude", "", "go-regexp matching files to exclude from included")

	if err := cmd.Flags().MarkHidden("cleanOnly"); err != nil {
		panic(err)
	}
}

type buildCmdT struct {
	verbose       bool
	leaveTemps    bool
	goBuild       bool
	staticLinking bool
	cleanOnly     bool
	debug         bool
	livePrefix    string
	pluginDir     string
	outputDir     string
	include       string
	exclude       string

	pluginPkgPath string
	tmpDirName    string
	tmpDir        string
	tmpPkgPath    string

	rexInclude *regexp.Regexp
	rexExclude *regexp.Regexp
}

func (wo *buildCmdT) execute(cmd *cobra.Command, args []string) {
	defer func() {
		if r := recover(); r != nil {
			if wo.debug {
				_, _ = fmt.Fprintf(os.Stderr, "%s\n\n%s", r, debug.Stack())
			} else {
				_, _ = os.Stderr.WriteString(fmt.Sprintln(r))
			}
			os.Exit(1)
		}
	}()

	if len(args) != 2 {
		_, _ = os.Stderr.WriteString(cmd.UsageString())
		os.Exit(1)
	}

	wo.livePrefix = strings.ToLower(strings.TrimSpace(wo.livePrefix))
	if wo.livePrefix == "" {
		panic("--livePrefix cannot be empty")
	}

	timing.totalStart = time.Now()
	wo.pluginDir = args[0]
	wo.outputDir = args[1]
	if err := hutils.FindDirectory(wo.pluginDir, "pluginDir"); err != nil {
		panic(err)
	}
	if err := hutils.FindDirectory(wo.outputDir, "outputDir"); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		} else if wo.staticLinking {
			panic(err)
		}
		if err := os.MkdirAll(wo.outputDir, 0744); err != nil {
			panic(err)
		}
	}

	if _, err := exec.LookPath("go"); err != nil {
		panic(err)
	}
	if _, err := exec.LookPath("gofmt"); err != nil {
		panic(err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		panic(err)
	}

	absDir1, err := filepath.Abs(wo.pluginDir)
	if err != nil {
		panic(err)
	}
	absDir2, err := filepath.Abs(wo.outputDir)
	if err != nil {
		panic(err)
	}
	wo.pluginDir = absDir1
	wo.outputDir = absDir2

	if wo.staticLinking {
		if wo.outputDir == wo.pluginDir {
			panic("pluginDir and outputDir must not be identical")
		}
		if rel, err := filepath.Rel(wo.pluginDir, wo.outputDir); err == nil &&
			!strings.HasPrefix(rel, "..") {
			panic("outputDir must not be a subdirectory of pluginDir")
		}
	}

	_, pkgPath, err := hutils.PackageFromDirectory(wo.pluginDir)
	if err != nil {
		panic(fmt.Errorf("failed to determine the package path. err: %v", err))
	}
	wo.pluginPkgPath = pkgPath

	now := time.Now().Format("02150405")
	commitInfo := wo.commitInfo()
	if wo.staticLinking {
		wo.tmpDirName = filepath.Base(wo.pluginDir)
		wo.tmpDir = wo.pluginDir
		wo.tmpPkgPath = wo.pluginPkgPath
		goto next1
	}
	wo.tmpDirName = fmt.Sprintf("%s-%s-%s", filepath.Base(wo.pluginDir), commitInfo, now)
	wo.tmpDir = filepath.Join(filepath.Dir(wo.pluginDir), wo.tmpDirName)
	wo.tmpPkgPath = path.Join(path.Dir(pkgPath), wo.tmpDirName)

	if wo.include != "" {
		if wo.rexInclude, err = regexp.Compile(wo.include); err != nil {
			panic(fmt.Errorf("failed to compile the --include regular expression. err: %w", err))
		}
	}
	if wo.exclude != "" {
		if wo.rexExclude, err = regexp.Compile(wo.exclude); err != nil {
			panic(fmt.Errorf("failed to compile the --exclude regular expression. err: %w", err))
		}
	}

	if err := hutils.FindDirectory(wo.tmpDir, ""); err == nil {
		if err := os.RemoveAll(wo.tmpDir); err != nil {
			panic(err)
		}
	}
next1:

	if !wo.goBuild {
		wo.leaveTemps = true
	} else if wo.staticLinking {
		wo.leaveTemps = true
	}

	chSignal := make(chan os.Signal, 1)
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signal.Notify(chSignal, sigs...)
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case x := <-chSignal:
			signal.Reset(sigs...)
			wo.removeTmpDir()
			if x == syscall.SIGINT {
				go func() {
					time.Sleep(time.Second)
					fmt.Println("\nPress Ctrl-C again to terminate the program immediately.")
				}()
				close(haltProgram)
			}
		}
	}()
	defer func() {
		select {
		case chSignal <- syscall.SIGQUIT:
		default:
		}
		<-done
	}()

	var outputFile string
	if !wo.staticLinking {
		if runtime.GOOS == "windows" {
			_, _ = os.Stderr.WriteString("Go plugin does not support Windows at present. Use --staticLinking if you only want to debug.\n")
			os.Exit(1)
		}
		if os.Getenv("hotswap:checkRequiredPluginFuncs") != "0" {
			parseRequiredPluginFuncs(wo.pluginDir, "")
		}
		outputFile = wo.buildPlugin()
	} else if wo.cleanOnly {
		removeStaticFiles(buildCompletePluginArgs(wo, false, true, nil))
	} else {
		wo.genStaticPlugin()
	}

	if wo.verbose {
		timing.total = time.Since(timing.totalStart)
		wo.outputTiming()
	}
	if outputFile != "" {
		if wo.verbose {
			fmt.Println()
		}
		fmt.Println(outputFile)
	}
}

func (wo *buildCmdT) outputTiming() {
	fmt.Println()
	fmt.Println("Timing:")
	fmt.Println(strings.Repeat("=", 30))

	const (
		copyFiles       = "Copy Files"
		processPackages = "Process Packages"
		build           = "Build"
		total           = "Total"
	)

	type timingItem struct {
		Title    string
		Duration time.Duration
	}

	a := []timingItem{
		{Title: copyFiles, Duration: timing.copyFiles},
		{Title: processPackages, Duration: timing.processPackages},
		{Title: build, Duration: timing.build},
		{Title: total, Duration: timing.total},
	}
	if wo.staticLinking {
		if wo.cleanOnly {
			a = []timingItem{a[1], a[3]}
		} else {
			a = []timingItem{a[3]}
		}
	}
	maxLen := len(processPackages)
	for _, v := range a {
		padding := strings.Repeat(" ", maxLen-len(v.Title))
		fmt.Printf("%s: %s%v\n", v.Title, padding, v.Duration.Round(time.Millisecond))
	}
}

func (wo *buildCmdT) removeTmpDir() {
	if !wo.leaveTemps {
		_ = os.RemoveAll(wo.tmpDir)
	}
}

func (wo *buildCmdT) commitInfo() string {
	cmd1 := exec.Command("git", "log", "-1", "--format=%H", "HEAD")
	cmd1.Dir = wo.pluginDir
	cmd1.Stderr = os.Stderr
	output1, err := cmd1.Output()
	if err != nil {
		panic(err)
	}

	cmd2 := exec.Command("git", "log", "-1", "--format=%ct", "HEAD")
	cmd2.Dir = wo.pluginDir
	cmd2.Stderr = os.Stderr
	output2, err := cmd2.Output()
	if err != nil {
		panic(err)
	}
	output2 = bytes.Trim(output2, "\r\n")

	n, err := strconv.Atoi(string(output2))
	if err != nil {
		panic(fmt.Errorf("failed to parse the timestamp of the last git commit. str: %s, err: %v", output2, err))
	}
	t := time.Unix(int64(n), 0).UTC().Format(hutils.CompactDateTimeFormat)
	return fmt.Sprintf("%s-%s", t, output1[:8])
}

func (wo *buildCmdT) genStaticPlugin() {
	fmt.Printf("Generating code for static plugin %s...\n", filepath.Base(wo.pluginDir))
	genStaticCode := func(args completePluginArgs, generated *generatedFiles) {
		genHotswapStaticPluginInit(args, generated)
		genHotswapStaticPlugins(args, generated)
	}
	completePlugin(buildCompletePluginArgs(wo, true, true, genStaticCode))
}

func countPluginInitFiles(args completePluginArgs) int {
	str1 := strings.ReplaceAll(hotswapStaticPluginInitFile, ".", `\.`)
	str2 := strings.ReplaceAll(str1, "%s", ".+?")
	rexName := regexp.MustCompile(str2)

	var counter int
	_ = filepath.WalkDir(args.outputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != args.outputDir {
				return fs.SkipDir
			}
		}
		if rexName.MatchString(filepath.Base(path)) {
			counter++
		}
		return nil
	})
	return counter
}

func removeStaticFiles(args completePluginArgs) {
	files := make([]string, 0, 1024)
	err := filepath.WalkDir(args.pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := hotswapFiles[filepath.Base(path)]; !ok {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	a := make([]string, len(files))
	for i, f := range files {
		rel, err := filepath.Rel(args.pluginDir, f)
		if err != nil {
			panic(err)
		}
		a[i] = rel
	}

	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			panic(err)
		}
	}

	if args.cleanOnly {
		for _, pkgName := range hotswapBureauPackageNames {
			bureauDir := filepath.Join(args.pluginDir, pkgName)
			if err := hutils.FindDirectory(bureauDir, ""); err == nil {
				_ = os.RemoveAll(bureauDir)
			}
		}
	}

	pluginName := filepath.Base(args.pluginDir)
	pluginInitFile := filepath.Join(args.outputDir, fmt.Sprintf(hotswapStaticPluginInitFile, pluginName))
	var pluginInitFileRemoved bool
	if _, err := os.Stat(pluginInitFile); err == nil {
		err := os.RemoveAll(pluginInitFile)
		if err != nil {
			panic(err)
		}
		pluginInitFileRemoved = true
	}

	numInitFiles := countPluginInitFiles(args)
	pluginVarDefFile := filepath.Join(args.outputDir, hotswapStaticPluginsFile)
	var pluginVarDefFileWantRemoved bool
	if numInitFiles == 0 {
		if _, err := os.Stat(pluginVarDefFile); err == nil {
			pluginVarDefFileWantRemoved = true
		}
	}

	if args.verbose {
		if !args.cleanOnly {
			fmt.Println()
		}
		fmt.Printf("Removed Files (%s):\n", pluginName)
		fmt.Println(strings.Repeat("=", 30))
		sort.Strings(a)
		for _, f := range a {
			fmt.Println("\t" + f)
		}
		if pluginInitFileRemoved {
			fmt.Println()
			fmt.Println("Removed Files:")
			fmt.Println(strings.Repeat("=", 30))
			fmt.Println("\t" + fmt.Sprintf(hotswapStaticPluginInitFile, pluginName))
		}
		if pluginVarDefFileWantRemoved {
			fmt.Println()
			fmt.Println("Files May Need to Remove Manually")
			fmt.Println(strings.Repeat("=", 30))
			fmt.Println("\t" + hotswapStaticPluginsFile)
		}
	}
}

type pluginFunc struct {
	Name string
	Expr string
}

func parseRequiredPluginFuncs(pluginDir, pluginPkgName string) []pluginFunc {
	pluginFuncMap := map[string]string{
		"OnLoad":           "nil",
		"OnInit":           "nil",
		"OnFree":           "nil",
		"Export":           "nil",
		"Import":           "nil",
		"InvokeFunc":       "nil",
		"Reloadable":       "nil",
		"HotswapLiveFuncs": "nil",
		"HotswapLiveTypes": "nil",
	}
	if pluginPkgName == "" {
		delete(pluginFuncMap, "HotswapLiveFuncs")
		delete(pluginFuncMap, "HotswapLiveTypes")
	}

	var fset token.FileSet
	pkgs, err := parser.ParseDir(&fset, pluginDir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Recv != nil {
					continue
				}
				funcName := funcDecl.Name.Name
				if _, ok := pluginFuncMap[funcName]; ok {
					pluginFuncMap[funcName] = fmt.Sprintf("%s.%s", pluginPkgName, funcName)
				}
			}
		}
	}

	var a []string
	var missing []string
	for k, v := range pluginFuncMap {
		if v == "nil" {
			missing = append(missing, k)
		} else {
			a = append(a, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		panic("missing fundamental plugin functions: " + hutils.Join(missing...))
	}
	sort.Strings(a)

	var ret []pluginFunc
	for _, k := range a {
		ret = append(ret, pluginFunc{
			Name: k,
			Expr: pluginFuncMap[k],
		})
	}
	return ret
}

func genHotswapStaticPluginInit(args completePluginArgs, generated *generatedFiles) {
	pkgName, _, err := hutils.PackageFromDirectory(args.outputDir)
	if err != nil {
		panic(err)
	}
	pluginPkgName, pluginPkgPath, err := hutils.PackageFromDirectory(args.tmpDir)
	if err != nil {
		panic(err)
	}
	switch pluginPkgName {
	case "hotswap":
		pluginPkgName = "hotswap2"
	}

	tpl := template.Must(template.New("hotswapStaticPluginInit").
		Parse(tplHotswapStaticPluginInit))
	tplArgs := struct {
		PackageName   string
		PluginPkgName string
		PluginPkgPath string
		PluginName    string
		PluginFuncs   []pluginFunc
	}{
		PackageName:   pkgName,
		PluginPkgName: pluginPkgName,
		PluginPkgPath: pluginPkgPath,
		PluginName:    filepath.Base(args.pluginDir),
		PluginFuncs:   parseRequiredPluginFuncs(args.pluginDir, pluginPkgName),
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, &tplArgs); err != nil {
		panic(err)
	}
	fileName := fmt.Sprintf(hotswapStaticPluginInitFile, tplArgs.PluginName)
	file := filepath.Join(args.outputDir, fileName)
	if err := ioutil.WriteFile(file, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
	if args.gofmt {
		if err := hutils.Gofmt(file); err != nil {
			panic(err)
		}
	}

	generated.add(file, true)
}

func genHotswapStaticPlugins(args completePluginArgs, generated *generatedFiles) {
	pkgName, _, err := hutils.PackageFromDirectory(args.outputDir)
	if err != nil {
		panic(err)
	}

	tpl := template.Must(template.New("hotswapStaticPlugins").
		Parse(tplHotswapStaticPlugins))
	tplArgs := struct {
		PackageName string
	}{
		PackageName: pkgName,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, &tplArgs); err != nil {
		panic(err)
	}
	file := filepath.Join(args.outputDir, hotswapStaticPluginsFile)
	if err := ioutil.WriteFile(file, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
	if args.gofmt {
		if err := hutils.Gofmt(file); err != nil {
			panic(err)
		}
	}

	generated.add(file, true)
}

func (wo *buildCmdT) buildPlugin() string {
	if wo.goBuild {
		fmt.Printf("Building plugin %s...\n", filepath.Base(wo.pluginDir))
		if wo.verbose || wo.leaveTemps {
			fmt.Println("TempDir: " + wo.tmpDir)
		}
	} else {
		fmt.Println(wo.tmpDir)
	}

	timing.copyFilesStart = time.Now()
	files := wo.collectFiles()
	wo.copyFiles(files)
	timing.copyFiles = time.Since(timing.copyFilesStart)

	completePlugin(buildCompletePluginArgs(wo, false, false, nil))

	outputFileName := filepath.Base(wo.pluginDir) + hutils.FileNameExt
	outputFile := filepath.Join(wo.outputDir, outputFileName)

	var args []string
	args = append(args, "build")
	args = append(args, "-trimpath")
	args = append(args, "-buildmode=plugin")
	args = append(args, "-o", outputFile)
	args = append(args, g.BuildFlags...)
	if wo.verbose {
		fmt.Println()
		fmt.Println("Command: go " + strings.Join(args, " "))
		if !wo.goBuild {
			fmt.Println("\nSkip building.")
			return ""
		}
	} else {
		if !wo.goBuild {
			return ""
		}
	}

	timing.buildStart = time.Now()
	defer func() {
		timing.build = time.Since(timing.buildStart)
	}()

	goBuild := exec.Command("go", args...)
	goBuild.Dir = wo.tmpDir
	goBuild.Stdout = os.Stdout
	goBuild.Stderr = os.Stderr
	goBuild.Env = append(os.Environ(), "GO111MODULE=on")
	if err := goBuild.Run(); err != nil {
		panic(err)
	}

	return outputFile
}

func (wo *buildCmdT) collectFiles() []string {
	var rexMatch func(path string) bool
	switch {
	case wo.rexInclude != nil && wo.rexExclude != nil:
		rexMatch = func(path string) bool {
			return wo.rexInclude.MatchString(path) && !wo.rexExclude.MatchString(path)
		}
	case wo.rexInclude != nil:
		rexMatch = func(path string) bool {
			return wo.rexInclude.MatchString(path)
		}
	default:
		rexMatch = func(path string) bool {
			return false
		}
	}

	files := make([]string, 0, 1024)
	err := filepath.WalkDir(wo.pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(wo.pluginDir, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(wo.tmpDir, rel), 0744)
		}

		switch {
		case strings.HasSuffix(path, "_test.go"):
			return nil
		case strings.HasSuffix(path, ".go"):
			if _, ok := hotswapFiles[filepath.Base(path)]; ok {
				return nil
			}
		case rexMatch(filepath.Base(rel)):
		default:
			return nil
		}

		files = append(files, rel)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

func (wo *buildCmdT) copyFiles(files []string) {
	rexPkg := regexp.MustCompile(`(?m)^package\s+\S+`)
	oldImportPath1 := []byte(`"` + wo.pluginPkgPath + `"`)
	newImportPath1 := []byte(`"` + wo.tmpPkgPath + `"`)
	oldImportPath2 := []byte(`"` + wo.pluginPkgPath + `/`)
	newImportPath2 := []byte(`"` + wo.tmpPkgPath + `/`)
	oldImportPath3 := []byte("`" + wo.pluginPkgPath + "`")
	newImportPath3 := []byte("`" + wo.tmpPkgPath + "`")
	oldImportPath4 := []byte("`" + wo.pluginPkgPath + "/")
	newImportPath4 := []byte("`" + wo.tmpPkgPath + "/")

	replaceCode := func(rel string, data1 []byte) []byte {
		if filepath.Dir(rel) == "." {
			data1 = rexPkg.ReplaceAll(data1, []byte("package main"))
		}
		data2 := bytes.ReplaceAll(data1, oldImportPath1, newImportPath1)
		data3 := bytes.ReplaceAll(data2, oldImportPath2, newImportPath2)
		data4 := bytes.ReplaceAll(data3, oldImportPath3, newImportPath3)
		data5 := bytes.ReplaceAll(data4, oldImportPath4, newImportPath4)
		return data5
	}

	pending := make(chan string, len(files))
	for _, rel := range files {
		pending <- rel
	}

	burst := runtime.NumCPU() * 5
	chErr := make(chan error, burst)
	abort := make(chan struct{})
	reportErr := func(err error) {
		chErr <- err
		select {
		case <-abort:
		default:
			close(abort)
		}
	}

	var wg sync.WaitGroup
	wg.Add(burst)
	for i := 0; i < burst; i++ {
		go func() {
			defer wg.Done()
			var rel string
			for {
				select {
				case <-haltProgram:
					return
				case <-abort:
					return
				case rel = <-pending:
				default:
					return
				}

				abs := filepath.Join(wo.pluginDir, rel)
				data1, err := ioutil.ReadFile(abs)
				if err != nil {
					reportErr(err)
					return
				}
				data9 := data1
				if strings.HasSuffix(rel, ".go") {
					data9 = replaceCode(rel, data1)
				}
				tmpFile := filepath.Join(wo.tmpDir, rel)
				err = ioutil.WriteFile(tmpFile, data9, 0644)
				if err != nil {
					reportErr(err)
					return
				}
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-chErr:
		panic(err)
	default:
	}
}

type generatedFiles struct {
	mu    sync.Mutex
	files map[string]bool
}

func (wo *generatedFiles) add(file string, outside bool) {
	wo.mu.Lock()
	wo.files[file] = outside
	wo.mu.Unlock()
}

func (wo *generatedFiles) snapshot() map[string]bool {
	wo.mu.Lock()
	m := make(map[string]bool)
	for k, v := range wo.files {
		m[k] = v
	}
	wo.mu.Unlock()
	return m
}

type completePluginArgs struct {
	verbose       bool
	gofmt         bool
	clean         bool
	cleanOnly     bool
	livePrefix    string
	pluginDir     string
	outputDir     string
	pluginPkgPath string
	tmpDirName    string
	tmpDir        string
	tmpPkgPath    string
	genStaticCode func(completePluginArgs, *generatedFiles)
}

func buildCompletePluginArgs(cmd *buildCmdT, gofmt, clean bool, genStaticCode func(completePluginArgs, *generatedFiles)) completePluginArgs {
	return completePluginArgs{
		verbose:       cmd.verbose,
		gofmt:         gofmt,
		clean:         clean,
		cleanOnly:     cmd.cleanOnly,
		livePrefix:    cmd.livePrefix,
		pluginDir:     cmd.pluginDir,
		outputDir:     cmd.outputDir,
		pluginPkgPath: cmd.pluginPkgPath,
		tmpDirName:    cmd.tmpDirName,
		tmpDir:        cmd.tmpDir,
		tmpPkgPath:    cmd.tmpPkgPath,
		genStaticCode: genStaticCode,
	}
}

func parseHotswapComment(group *ast.CommentGroup) string {
	if group != nil {
		for _, comment := range group.List {
			if i := strings.Index(comment.Text, "hotswap:"); i >= 0 {
				return strings.TrimSpace(comment.Text[i+8:])
			}
		}
	}
	return ""
}

func genHotswapBureau(args completePluginArgs, generated *generatedFiles) {
	for _, pkgName := range hotswapBureauPackageNames {
		bureauDir := filepath.Join(args.pluginDir, pkgName)
		if err := hutils.FindDirectory(bureauDir, ""); err == nil {
			_ = os.RemoveAll(bureauDir)
		}
	}

	dir := filepath.Join(args.tmpDir, hotswapBureauPackageNames[0])
	if err := os.MkdirAll(dir, 0744); err != nil {
		panic(err)
	}
	file := filepath.Join(dir, hotswapBureauFile)
	if err := ioutil.WriteFile(file, tplHotswapBureau, 0644); err != nil {
		panic(err)
	}
	if args.gofmt {
		if err := hutils.Gofmt(file); err != nil {
			panic(err)
		}
	}

	generated.add(file, false)
}

func genHotswapMain(args completePluginArgs, livePackages map[string]*packages.Package, generated *generatedFiles) {
	var a []string
	for _, pkg := range livePackages {
		if pkg.PkgPath != args.tmpPkgPath {
			a = append(a, pkg.PkgPath)
		}
	}
	sort.Strings(a)

	pkgName := "main"
	if args.genStaticCode != nil {
		var err error
		pkgName, _, err = hutils.PackageFromDirectory(args.tmpDir)
		if err != nil {
			panic(err)
		}
	}

	tpl := template.Must(template.New("hotswapMain").Parse(tplHotswapMain))
	tplArgs := struct {
		PackageName       string
		BureauPackagePath string
		LivePackages      []string
	}{
		PackageName:       pkgName,
		BureauPackagePath: path.Join(args.tmpPkgPath, hotswapBureauPackageNames[0]),
		LivePackages:      a,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, &tplArgs); err != nil {
		panic(err)
	}
	file := filepath.Join(args.tmpDir, hotswapMainFile)
	if err := ioutil.WriteFile(file, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
	if args.gofmt {
		if err := hutils.Gofmt(file); err != nil {
			panic(err)
		}
	}

	generated.add(file, false)
}

func genHotswapLive(args completePluginArgs, dir string, pkg *packages.Package, liveFuncs, liveTypes []string, generated *generatedFiles) {
	tpl := template.Must(template.New("hotswapLive").Parse(tplHotswapLive))
	tplArgs := struct {
		PackageName       string
		BureauPackagePath string
		LiveFuncs         []string
		LiveTypes         []string
	}{
		PackageName:       pkg.Name,
		BureauPackagePath: path.Join(args.tmpPkgPath, hotswapBureauPackageNames[0]),
		LiveFuncs:         liveFuncs,
		LiveTypes:         liveTypes,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, &tplArgs); err != nil {
		panic(err)
	}
	file := filepath.Join(dir, hotswapLiveFile)
	if err := ioutil.WriteFile(file, buf.Bytes(), 0644); err != nil {
		panic(err)
	}
	if args.gofmt {
		if err := hutils.Gofmt(file); err != nil {
			panic(err)
		}
	}

	generated.add(file, false)
}

func completePlugin(args completePluginArgs) {
	timing.processPackagesStart = time.Now()
	defer func() {
		timing.processPackages = time.Since(timing.processPackagesStart)
	}()

	if err := os.Chdir(args.tmpDir); err != nil {
		panic(err)
	}

	var cfg packages.Config
	cfg.Mode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax
	pkgs, err := packages.Load(&cfg, args.tmpPkgPath+"/...")
	if err != nil {
		panic(err)
	}
	if args.verbose {
		fmt.Printf("Total Packages: %d\n", len(pkgs))
	}

	liveNames := make(map[string]string)
	liveFuncs := make(map[string][]string)
	liveTypes := make(map[string][]string)
	livePackages := make(map[string]*packages.Package)

	var numErrs int
	printErrorMessage := func(str string) {
		_, _ = os.Stderr.WriteString("Error: " + str)
		numErrs++
	}

	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 {
			continue
		}
		select {
		case <-haltProgram:
			return
		default:
		}
		dir := filepath.Dir(pkg.GoFiles[0])
		for _, synt := range pkg.Syntax {
			for _, decl := range synt.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					if funcDecl.Recv != nil {
						continue
					}
					funcName := funcDecl.Name.Name
					funcNameLower := strings.ToLower(funcName)
					if !strings.HasPrefix(funcNameLower, args.livePrefix) {
						continue
					}
					if _, ok := liveNames[funcName]; ok {
						printErrorMessage("duplicate live func/type name detected: " + funcName)
						continue
					}
					liveNames[funcName] = parseHotswapComment(funcDecl.Doc)
					liveFuncs[dir] = append(liveFuncs[dir], funcName)
					livePackages[dir] = pkg
				} else if genDecl, ok := decl.(*ast.GenDecl); ok {
					if genDecl.Tok != token.TYPE {
						continue
					}
					for _, spec := range genDecl.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						typeName := typeSpec.Name.Name
						typeNameLower := strings.ToLower(typeName)
						if !strings.HasPrefix(typeNameLower, args.livePrefix) {
							continue
						}
						if _, ok := typeSpec.Type.(*ast.StructType); !ok {
							printErrorMessage(fmt.Sprintf("%s (a live type) must be a simple struct. package: %s",
								typeName, pkg.PkgPath))
							continue
						}
						if _, ok := liveNames[typeName]; ok {
							printErrorMessage("duplicate live func/type name detected: " + typeName)
							continue
						}
						liveNames[typeName] = parseHotswapComment(typeSpec.Comment)
						liveTypes[dir] = append(liveTypes[dir], typeName)
						livePackages[dir] = pkg
					}
				}
			}
		}
	}
	if numErrs > 0 {
		panic(fmt.Errorf("%d errors occurred", numErrs))
	}

	if args.clean {
		removeStaticFiles(args)
	}

	if args.verbose {
		fmt.Println()
		fmt.Println("Live Functions:")
		fmt.Println(strings.Repeat("=", 30))
		var all []string
		for _, a := range liveFuncs {
			all = append(all, a...)
		}
		sort.Strings(all)
		for _, f := range all {
			fmt.Println("\t" + f)
		}
	}

	if args.verbose {
		fmt.Println()
		fmt.Println("Live Types:")
		fmt.Println(strings.Repeat("=", 30))
		var all []string
		for _, a := range liveTypes {
			all = append(all, a...)
		}
		sort.Strings(all)
		for _, f := range all {
			fmt.Println("\t" + f)
		}
	}

	var generated generatedFiles
	generated.files = make(map[string]bool)
	genHotswapBureau(args, &generated)
	genHotswapMain(args, livePackages, &generated)

	for k, pkg := range livePackages {
		select {
		case <-haltProgram:
			return
		default:
		}
		genHotswapLive(args, k, pkg, liveFuncs[k], liveTypes[k], &generated)
	}

	if args.genStaticCode != nil {
		args.genStaticCode(args, &generated)
	}

	if args.verbose {
		var all = generated.snapshot()
		var rels1, rels2 []string
		for abs, outside := range all {
			var rel string
			if outside {
				rel, err = filepath.Rel(args.outputDir, abs)
			} else {
				rel, err = filepath.Rel(args.tmpDir, abs)
			}
			if err != nil {
				rel = abs
			}
			if outside {
				rels2 = append(rels2, rel)
			} else {
				rels1 = append(rels1, rel)
			}
		}
		sort.Strings(rels1)
		sort.Strings(rels2)

		fmt.Println()
		fmt.Printf("Generated Files (%s):\n", filepath.Base(args.pluginDir))
		fmt.Println(strings.Repeat("=", 30))
		for _, f := range rels1 {
			fmt.Println("\t" + f)
		}
		if len(rels2) > 0 {
			fmt.Println()
			fmt.Println("Generated Files:")
			fmt.Println(strings.Repeat("=", 30))
			for _, f := range rels2 {
				fmt.Println("\t" + f)
			}
		}
	}
}
