/*
Copyright 2026 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"emperror.dev/errors"
	"github.com/Masterminds/sprig/v3"
)

const (
	// DefaultLeftDelimiter is the default left-hand delimiter for templates.
	DefaultLeftDelimiter = "${"
	// DefaultRightDelimiter is the default right-hand delimiter for templates.
	DefaultRightDelimiter = "}"
)

const templateName = "config"

// Templater holds delimiters used to configure the template engine.
type Templater struct {
	leftDelimiter  string
	rightDelimiter string
}

// NewTemplater initializes a new templater object.
func NewTemplater(leftDelimiter, rightDelimiter string) Templater {
	return Templater{
		leftDelimiter:  leftDelimiter,
		rightDelimiter: rightDelimiter,
	}
}

// Template interpolates a data structure in a template.
func (t Templater) Template(templateText string, data interface{}) (*bytes.Buffer, error) {
	configTemplate, err := template.New(templateName).
		Funcs(sprig.TxtFuncMap()).
		Funcs(templateFuncs()).
		Delims(t.leftDelimiter, t.rightDelimiter).
		Parse(templateText)
	if err != nil {
		return nil, errors.WrapIf(err, "error parsing template")
	}

	buffer := bytes.NewBuffer(nil)

	err = configTemplate.ExecuteTemplate(buffer, templateName, data)
	if err != nil {
		return nil, errors.WrapIf(err, "error executing template")
	}

	return buffer, nil
}

// IsGoTemplate returns true if s is probably a Go template.
func (t Templater) IsGoTemplate(templateText string) bool {
	return strings.Contains(templateText, t.leftDelimiter) && strings.Contains(templateText, t.rightDelimiter)
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"file":     fileContent,
		"accessor": accessorPlaceholder,
	}
}

func accessorPlaceholder(path string) (string, error) {
	return fmt.Sprintf("__accessor__%s", strings.TrimRight(path, "/")), nil
}

func fileContent(path string) (string, error) {
	r, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(string(r), "\n", "\\n"), nil
}
