/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import "flag"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "os"
import "os/exec"
import "path"
import "runtime"

import "git-wip-us.apache.org/repos/asf/lucy-clownfish.git/compiler/go/cfc"

var packageName string = "git-wip-us.apache.org/repos/asf/lucy.git/go/lucy"
var cfPackageName string = "git-wip-us.apache.org/repos/asf/lucy-clownfish.git/runtime/go/clownfish"
var charmonizerC string = "../common/charmonizer.c"
var charmonizerEXE string = "charmonizer"
var charmonyH string = "charmony.h"
var buildDir string
var hostSrcDir string
var buildGO string
var configGO string
var cfbindGO string
var installedLibPath string

func init() {
	_, buildGO, _, _ = runtime.Caller(1)
	buildDir = path.Dir(buildGO)
	hostSrcDir = path.Join(buildDir, "../c/src")
	configGO = path.Join(buildDir, "lucy", "config.go")
	cfbindGO = path.Join(buildDir, "lucy", "cfbind.go")
	var err error
	installedLibPath, err = cfc.InstalledLibPath(packageName)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	os.Chdir(buildDir)
	flag.Parse()
	action := "build"
	args := flag.Args()
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "build":
		build()
	case "clean":
		clean()
	case "test":
		test()
	case "install":
		install()
	default:
		log.Fatalf("Unrecognized action specified: %s", action)
	}
}

func current(orig, dest string) bool {

	destInfo, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			// If dest doesn't exist, we're not current.
			return false
		} else {
			log.Fatalf("Unexpected stat err: %s", err)
		}
	}

	// If source is newer than dest, we're not current.
	origInfo, err := os.Stat(orig)
	if err != nil {
		log.Fatalf("Unexpected: %s", err)
	}
	return origInfo.ModTime().Before(destInfo.ModTime())
}

func runCommand(name string, args ...string) {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func configure() {
	if !current(charmonizerC, charmonizerEXE) {
		runCommand("cc", "-o", charmonizerEXE, charmonizerC)
	}
	if !current(charmonizerEXE, charmonyH) {
		runCommand("./charmonizer", "--cc=cc", "--enable-c", "--enable-go",
			"--enable-makefile", "--host=go", "--", "-std=gnu99", "-O2")
	}
}

func runCFC() {
	hierarchy := cfc.NewHierarchy("autogen")
	hierarchy.AddSourceDir("../core")
	hierarchy.Build()
	autogenHeader := "Auto-generated by build.go.\n"
	coreBinding := cfc.NewBindCore(hierarchy, autogenHeader, "")
	modified := coreBinding.WriteAllModified(false)
	if modified {
		cfc.RegisterParcelPackage("Clownfish", cfPackageName)
		goBinding := cfc.NewBindGo(hierarchy)
		goBinding.SetHeader(autogenHeader)
		goBinding.SetSuppressInit(true)
		parcel := cfc.FetchParcel("Lucy")
		specClasses(parcel)
		packageDir := path.Join(buildDir, "lucy")
		goBinding.WriteBindings(parcel, packageDir)
		hierarchy.WriteLog()
	}
}

func specClasses(parcel *cfc.Parcel) {
	indexerBinding := cfc.NewGoClass(parcel, "Lucy::Index::Indexer")
	indexerBinding.SpecMethod("", "Close() error")
	indexerBinding.SpecMethod("Add_Doc", "AddDoc(doc interface{}) error")
	indexerBinding.SpecMethod("Commit", "Commit() error")
	indexerBinding.SetSuppressStruct(true)
	indexerBinding.Register()

	schemaBinding := cfc.NewGoClass(parcel, "Lucy::Plan::Schema")
	schemaBinding.SpecMethod("Spec_Field",
		"SpecField(field string, fieldType FieldType)")
	schemaBinding.Register()

	searcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::Searcher")
	searcherBinding.SpecMethod("Hits",
		"Hits(query interface{}, offset uint32, numWanted uint32, sortSpec SortSpec) (Hits, error)")
	searcherBinding.SpecMethod("Close", "Close() error")
	searcherBinding.Register()

	hitsBinding := cfc.NewGoClass(parcel, "Lucy::Search::Hits")
	hitsBinding.SpecMethod("Next", "Next(hit interface{}) bool")
	hitsBinding.SpecMethod("", "Error() error")
	hitsBinding.SetSuppressStruct(true)
	hitsBinding.Register()
}

func build() {
	configure()
	runCFC()
	runCommand("make", "-j", "static")
	writeConfigGO()
	runCommand("go", "build", packageName)
}

func test() {
	build()
	runCommand("go", "test", packageName)
}

func copyFile(source, dest string) {
	sourceFH, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFH.Close()
	destFH, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer destFH.Close()
	_, err = io.Copy(destFH, sourceFH)
	if err != nil {
		log.Fatalf("io.Copy from %s to %s failed: %s", source, dest, err)
	}
}

func installStaticLib() {
	tempLibPath := path.Join(buildDir, "liblucy.a")
	destDir := path.Dir(installedLibPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			log.Fatalf("Can't create dir '%s': %s", destDir, err)
		}
	}
	os.Remove(installedLibPath)
	copyFile(tempLibPath, installedLibPath)
}

func install() {
	build()
	runCommand("go", "install", packageName)
	installStaticLib()
}

func writeConfigGO() {
	if current(buildGO, configGO) {
		return
	}
	installedLibDir := path.Dir(installedLibPath)
	cfLibPath, err := cfc.InstalledLibPath(cfPackageName)
	if err != nil {
		log.Fatal(err)
	}
	cfLibDir := path.Dir(cfLibPath)
	content := fmt.Sprintf(
		"// Auto-generated by build.go, specifying absolute path to static lib.\n"+
			"package lucy\n"+
			"// #cgo CFLAGS: -I%s/../core\n"+
			"// #cgo CFLAGS: -I%s\n"+
			"// #cgo CFLAGS: -I%s/autogen/include\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -llucy\n"+
			"// #cgo LDFLAGS: -lclownfish\n"+
			"import \"C\"\n",
		buildDir, buildDir, buildDir, buildDir, installedLibDir, cfLibDir)
	ioutil.WriteFile(configGO, []byte(content), 0666)
}

func clean() {
	fmt.Println("Cleaning")
	if _, err := os.Stat("Makefile"); !os.IsNotExist(err) {
		runCommand("make", "clean")
	}
	files := []string{charmonizerEXE, "charmony.h", "Makefile", configGO, cfbindGO}
	for _, file := range files {
		err := os.Remove(file)
		if err == nil {
			fmt.Println("Removing", file)
		} else if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
}
