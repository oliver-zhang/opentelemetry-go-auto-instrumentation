// Copyright (c) 2024 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package preprocess

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/config"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/resource"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/shared"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/tool/util"
	"github.com/dave/dst"
)

const (
	ApiPackage     = "api"
	ApiImportPath  = "github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	ApiCallContext = "CallContext"
)

func initRuleDir() (err error) {
	if exist, _ := util.PathExists(OtelRules); exist {
		err = os.RemoveAll(OtelRules)
		if err != nil {
			return fmt.Errorf("failed to remove dir %v: %w", OtelRules, err)
		}
	}
	err = os.MkdirAll(OtelRules, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create dir %v: %w", OtelRules, err)
	}
	return nil
}

func (dp *DepProcessor) copyRules(pkgName string) (err error) {
	if len(dp.bundles) == 0 {
		return nil
	}
	// Find out which resource files we should add to project
	for _, bundle := range dp.bundles {
		for _, funcRules := range bundle.File2FuncRules {
			// Copy resource file into project as otel_rule_\d.go
			for _, rs := range funcRules {
				for _, rule := range rs {

					// If rule inserts raw code directly, skip adding any
					// further dependencies
					if rule.UseRaw {
						continue
					}
					// Find files where hooks defines in and copy a whole
					files, err := resource.FindRuleFiles(rule)
					if err != nil {
						return err
					}
					if len(files) == 0 {
						return fmt.Errorf("can not find resource for %v", rule)
					}
					// Although different rule hooks may instrument the same
					// function, we still need to create separate directories
					// for each rule because different rule hooks may depend
					// on completely identical code or types. We need to use
					// different package prefixes to distinguish them.
					dir := bundle.PackageName + util.RandomString(5)
					dp.rule2Dir[rule] = dir

					for _, file := range files {
						if !shared.IsGoFile(file) || shared.IsGoTestFile(file) {
							continue
						}

						ruleDir := filepath.Join(pkgName, dir)
						err = os.MkdirAll(ruleDir, 0777)
						if err != nil {
							return fmt.Errorf("failed to create dir %v: %w",
								ruleDir, err)
						}
						ruleFile := filepath.Join(ruleDir, filepath.Base(file))
						err = dp.copyRule(file, ruleFile, bundle)
						if err != nil {
							return fmt.Errorf("failed to copy rule %v: %w",
								file, err)
						}
					}
				}
			}
		}
	}

	return nil
}
func rectifyCallContext(astRoot *dst.File, bundle *resource.RuleBundle) {
	// We write hook code by using api.CallContext as the first parameter, but
	// the actual package name is not api. Given net/http package, the actual
	// package name is http, so we should rectify the package name in the hook
	// code to the correct package name. We did this by renaming the import path
	// of api to the correct package name, and add an alias name for "api", this
	// is required because CallContext is defined in the api package, and we can
	// omit the package name before, but we can't do that now because of renaming
	newAliasName := bundle.PackageName + util.RandomString(5)
	alias := ApiPackage
	spec := shared.FindImport(astRoot, ApiImportPath)
	if spec != nil {
		if spec.Name != nil {
			alias = spec.Name.Name
		}
	}
	// Check if the function has api.CallContext as the first parameter
	// If so, rename it to the correct package name
	for _, decl := range astRoot.Decls {
		if f, ok := decl.(*dst.FuncDecl); ok {
			foundCallContext := false

			params := f.Type.Params.List
			for _, param := range params {
				if sele, ok := param.Type.(*dst.SelectorExpr); ok {
					if x, ok := sele.X.(*dst.Ident); ok {
						if x.Name == alias && sele.Sel.Name == ApiCallContext {
							foundCallContext = true
							x.Name = newAliasName
							break
						}
					}
				}
			}
			if foundCallContext {
				spec.Path.Value = fmt.Sprintf("%q", bundle.ImportPath)
				spec.Name = &dst.Ident{Name: newAliasName}
			}
		}
	}
}
func makeHookPublic(astRoot *dst.File, bundle *resource.RuleBundle) {
	// Only make hook public, keep it as it is if it's not a hook
	hooks := make(map[string]bool)
	for _, funcRules := range bundle.File2FuncRules {
		for _, rs := range funcRules {
			for _, rule := range rs {
				hooks[rule.OnEnter] = true
				hooks[rule.OnExit] = true
			}
		}
	}
	for _, decl := range astRoot.Decls {
		if f, ok := decl.(*dst.FuncDecl); ok {
			if _, ok := hooks[f.Name.Name]; !ok {
				continue
			}
			params := f.Type.Params.List
			for _, param := range params {
				if sele, ok := param.Type.(*dst.SelectorExpr); ok {
					if _, ok := sele.X.(*dst.Ident); ok {
						if sele.Sel.Name == ApiCallContext {
							f.Name.Name = strings.Title(f.Name.Name)
							break
						}
					}
				}
			}
		}
	}
}

func (dp *DepProcessor) copyRule(path, target string,
	bundle *resource.RuleBundle) error {
	text, err := util.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read rule file %v: %w", path, err)
	}
	text = shared.RemoveGoBuildComment(text)
	astRoot, err := shared.ParseAstFromSource(text)
	if err != nil {
		return fmt.Errorf("failed to parse ast from source: %w", err)
	}
	// Rename package name nevertheless
	astRoot.Name.Name = filepath.Base(filepath.Dir(target))

	// Rename api.CallContext to correct package name if present
	rectifyCallContext(astRoot, bundle)

	// Make hook functions public
	makeHookPublic(astRoot, bundle)

	// Copy used rule into project
	_, err = shared.WriteAstToFile(astRoot, target)
	if err != nil {
		return fmt.Errorf("failed to write ast to %v: %w", target, err)
	}
	if config.GetConf().Verbose {
		util.Log("Copy dependency %v to %v", path, target)
	}
	return nil
}

func (dp *DepProcessor) initRules(pkgName string) (err error) {
	c := fmt.Sprintf("package %s\n", pkgName)
	imports := make(map[string]string)

	assigns := make([]string, 0)
	for _, bundle := range dp.bundles {
		if len(bundle.File2FuncRules) == 0 {
			continue
		}
		addedImport := false
		for _, funcRules := range bundle.File2FuncRules {
			for _, rs := range funcRules {
				for _, rule := range rs {
					util.Assert(rule.OnEnter != "" || rule.OnExit != "",
						"sanity check")
					if rule.UseRaw {
						continue
					}
					var aliasPkg string
					if !addedImport {
						if bundle.PackageName == OtelPrintStackImportPath {
							aliasPkg = OtelPrintStackPkgAlias
						} else {
							aliasPkg = bundle.PackageName + util.RandomString(5)
						}
						imports[bundle.ImportPath] = aliasPkg
						addedImport = true
					} else {
						aliasPkg = imports[bundle.ImportPath]
					}
					if rule.OnEnter != "" {
						// @@Dont use filepath.Join here, because this is import
						// path presented in Go source code, which should always
						// use forward slash
						rd := fmt.Sprintf("%s/%s", OtelRules, dp.rule2Dir[rule])
						path, err := dp.getImportPathOf(rd)
						if err != nil {
							return fmt.Errorf("failed to get import path: %w",
								err)
						}
						imports[path] = dp.rule2Dir[rule]
						assigns = append(assigns,
							fmt.Sprintf("\t%s.%s = %s.%s\n",
								aliasPkg,
								shared.GetVarNameOfFunc(rule.OnEnter),
								dp.rule2Dir[rule],
								shared.MakePublic(rule.OnEnter),
							),
						)
					}
					if rule.OnExit != "" {
						rd := fmt.Sprintf("%s/%s", OtelRules, dp.rule2Dir[rule])
						path, err := dp.getImportPathOf(rd)
						if err != nil {
							return fmt.Errorf("failed to get import path: %w",
								err)
						}
						imports[path] = dp.rule2Dir[rule]
						assigns = append(assigns,
							fmt.Sprintf(
								"\t%s.%s = %s.%s\n",
								aliasPkg,
								shared.GetVarNameOfFunc(rule.OnExit),
								dp.rule2Dir[rule],
								shared.MakePublic(rule.OnExit),
							),
						)
					}
					assigns = append(assigns, fmt.Sprintf(
						"\t%s.%s = %s\n",
						aliasPkg,
						OtelGetStackDef,
						OtelGetStackImplCode,
					))
					assigns = append(assigns, fmt.Sprintf(
						"\t%s.%s = %s\n",
						aliasPkg,
						OtelPrintStackDef,
						OtelPrintStackImplCode,
					))
				}
			}
		}
	}

	// Imports
	if len(assigns) > 0 {
		imports[OtelPrintStackImportPath] = OtelPrintStackPkgAlias
		imports[OtelGetStackImportPath] = OtelGetStackAliasPkg
	}
	for k, v := range imports {
		c += fmt.Sprintf("import %s %q\n", v, k)
	}

	// Assignments
	c += " func init() { \n"
	for _, assign := range assigns {
		c += assign
	}
	c += "}\n"

	initTarget := filepath.Join(OtelRules, OtelSetupInst)
	_, err = util.WriteFile(initTarget, c)
	if err != nil {
		return err
	}
	return err
}

func (dp *DepProcessor) addRuleImport() error {
	ruleImportPath, err := dp.getImportPathOf(OtelRules)
	if err != nil {
		return fmt.Errorf("failed to get import path: %w", err)
	}
	err = dp.addExplicitImport(ruleImportPath)
	if err != nil {
		return fmt.Errorf("failed to add rule import: %w", err)
	}
	return nil
}

// Very hacky code here. We need to rewrite the localPrefix within the source
// to the real project module name. This is necessary because the localPrefix
// is used to identify whether the init task belongs to local project or not.
// Now that we are trying to reorder these tasks to the end of the init task
// list, we must know which one is the target we want to reorder. During the
// runtime, we are unable to know the real module name of the project, so we
// must done this during the compilation.
func (dp *DepProcessor) rewriteRules() error {
	// Rewrite localPrefix within the source to real project module name
	for _, bundle := range dp.bundles {
		if len(bundle.FileRules) == 0 {
			continue
		}
		for _, rule := range bundle.FileRules {
			if !strings.HasSuffix(rule.FileName, ReorderInitFile) {
				continue
			}
			astRoot, err := shared.ParseAstFromFile(rule.FileName)
			if err != nil {
				return fmt.Errorf("failed to parse ast from source: %w", err)
			}
			found := false
			dst.Inspect(astRoot, func(n dst.Node) bool {
				if basicLit, ok := n.(*dst.BasicLit); ok {
					if basicLit.Kind == token.STRING {
						quoted := fmt.Sprintf("%q", ReorderLocalPrefix)
						if basicLit.Value == quoted {
							gomod, err := shared.GetGoModPath()
							if err != nil {
								return false
							}
							moduleName, err := getModuleName(gomod)
							if err != nil {
								return false
							}
							found = true
							basicLit.Value = fmt.Sprintf("%q", moduleName)
							return false
						}
					}
				}
				return true
			})
			if !found {
				return fmt.Errorf("failed to find rewrite local prefix in %v",
					rule.FileName)
			} else {
				_, err = shared.WriteAstToFile(astRoot, rule.FileName)
				if err != nil {
					return fmt.Errorf("failed to write ast to %v: %w",
						rule.FileName, err)
				}
			}
		}
	}
	return nil
}

func (dp *DepProcessor) setupOtelSDK(pkgName string) error {
	setupTarget := filepath.Join(OtelRules, OtelSetupSDK)
	_, err := resource.CopyOtelSetupTo(pkgName, setupTarget)
	if err != nil {
		return fmt.Errorf("failed to copy otel setup sdk: %w", err)
	}
	return err
}

func (dp *DepProcessor) setupRules() (err error) {
	defer util.PhaseTimer("Setup")()
	err = initRuleDir()
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	err = dp.copyRules(OtelRules)
	if err != nil {
		return fmt.Errorf("failed to setup rules: %w", err)
	}
	err = dp.initRules(OtelRules)
	if err != nil {
		return fmt.Errorf("failed to setup initiator: %w", err)
	}
	err = dp.rewriteRules()
	if err != nil {
		return fmt.Errorf("failed to rewrite rules: %w", err)
	}
	err = dp.setupOtelSDK(OtelRules)
	if err != nil {
		return fmt.Errorf("failed to setup otel sdk: %w", err)
	}
	// Add rule import to all candidates
	err = dp.addRuleImport()
	if err != nil {
		return fmt.Errorf("failed to add rule import: %w", err)
	}
	return nil
}
