// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph

import "text/template"

const (
	dotTemplateSrc = `digraph {
	graph[fontname="Go"];
	node[shape=box,fontname="Go"];
	{{range $node := .Nodes}}
	"{{.Name}}" [URL="?node={{urlquery .Name}}"{{if gt .Multiplicity 1}},shape=box3d{{end}}];
		{{range $k, $t := .InputArgs}}
	"{{$node.Name}}.{{$k}}" [shape=point,xlabel="{{$k}}"];
    "{{$node.Name}}.{{$k}}" -> "{{$node.Name}}";
		{{- end}}
		{{range $k, $t := .OutputArgs}}
	"{{$node.Name}}.{{$k}}" [shape=point,xlabel="{{$k}}"];
	"{{$node.Name}}" -> "{{$node.Name}}.{{$k}}";
		{{- end}}
	{{- end}}
	{{range $index, $chan := .Channels}}
		{{if .IsSimple}}
	"{{index .Writers 0}}" -> "{{index .Readers 0}}" [URL="?channel={{$index}}",fontname="Go Mono"];
	    {{- else}}
	"$c{{$index}}" [URL="?channel={{$index}}",shape=point,fontname="Go Mono"];
			{{range $chan.Readers}}
    "$c{{$index}}" -> "{{.}}";
			{{- end}}
			{{range $chan.Writers}}
    "{{.}}" -> "$c{{$index}}";
			{{- end}}
		{{- end}}
	{{- end}}
}`

	// Fun fact: go/format fixes trailing commas in function args.
	goTemplateSrc = `{{if .IsCommand}}
// The {{.PackageName}} command was automatically generated by Shenzhen Go.
package main
{{else}}
// Package {{.PackageName}} was automatically generated by Shenzhen Go.
package {{.PackageName}} {{if ne .PackagePath .PackageName}} // import "{{.PackagePath}}"{{end}}
{{end}}

{{template "golang-defs" .}}

{{range .Nodes}}
func {{.Identifier}}({{range $name, $pin := .Pins}}{{$name}} {{$pin.Type}},{{end}}) {
	{{.ImplHead}}
	{{if eq .Multiplicity 1 -}}
	func(instanceNumber, multiplicity int) {
		{{.ImplBody}}
	}(0, 1)
	{{- else -}}
	var multWG sync.WaitGroup
	multWG.Add({{.Multiplicity}})
	for n:=0; n<{{.Multiplicity}}; n++ {
		go func(instanceNumber, multiplicity int) {
			defer multWG.Done()
			{{.ImplBody}}
		}(n, {{.Multiplicity}})
	}
	multWG.Wait()
	{{- end}}
	{{.ImplTail}}
}
{{end}}

{{if .IsCommand}}
func main() {
{{else}}
// Run executes all the goroutines associated with the graph that generated 
// this package, and waits for any that were marked as "wait for this to 
// finish" to finish before returning.
func Run() {
{{end}}
	{{- range $n, $c := .Channels}}
	c{{$n}} := make(chan {{$c.Type}}, {{$c.Cap}})
	{{- end}}

	var wg sync.WaitGroup
	{{range .Nodes}}
		{{if .Wait -}}
	wg.Add(1)
	go func() {
		{{.Identifier}}({{range .Pins}}{{.Value}},{{end}})
		wg.Done()
	}()
		{{else}}
	go {{.Identifier}}({{range .Pins}}{{.Value}},{{end}})
		{{- end}}
	{{- end}}

	// Wait for the end
	wg.Wait()
}`

	goDefinitionsTemplateSrc = `import (
	{{range .AllImports}}
	{{.}}
	{{- end}}
)
`

	goRunnerTemplateSrc = `package main

	import "{{.PackagePath}}"

	func main() {
		{{.PackageName}}.Run()
	}
`
)

var (
	dotTemplate           = template.Must(template.New("dot").Parse(dotTemplateSrc))
	goTemplate            = template.Must(template.New("golang").Parse(goTemplateSrc))
	goRunnerTemplate      = template.Must(template.New("golang-runner").Parse(goRunnerTemplateSrc))
	goDefinitionsTemplate = template.Must(goTemplate.New("golang-defs").Parse(goDefinitionsTemplateSrc))
)
