// Command genschema bundles the source schemas for embedding in the parser.
package main

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	const source = "../../../../../Schema/picolang"
	sourceFS := os.DirFS(source)
	documents := make(map[string]any)
	if err := fs.WalkDir(sourceFS, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			return err
		}
		data, err := fs.ReadFile(sourceFS, name)
		if err != nil {
			return err
		}
		var document map[string]any
		if err := yaml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		documents[name] = document
		return nil
	}); err != nil {
		return err
	}
	for name, document := range documents {
		if err := rewriteReferences(document, name, documents); err != nil {
			return err
		}
	}
	root := map[string]any{
		"$comment": "Generated from Schema/picolang by go generate; DO NOT EDIT.",
		"$schema":  "https://json-schema.org/draft/2020-12/schema",
		"$ref":     "#/$defs/toplevel.schema.yaml",
		"$defs":    documents,
	}
	data, err := json.Marshal(root, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return os.WriteFile("schema.gen.json", append(data, '\n'), 0o600)
}

func rewriteReferences(value any, name string, documents map[string]any) error {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "$ref" {
				ref, ok := child.(string)
				if !ok {
					return fmt.Errorf("%s: non-string reference", name)
				}
				u, err := url.Parse(ref)
				if err != nil {
					return err
				}
				if u.IsAbs() || u.Host != "" || u.RawQuery != "" || (u.Fragment != "" && !strings.HasPrefix(u.Fragment, "/")) {
					return fmt.Errorf("%s: unsupported reference %q", name, ref)
				}
				target := name
				if u.Path != "" {
					target = path.Join(path.Dir(name), u.Path)
				}
				if _, exists := documents[target]; !exists {
					return fmt.Errorf("%s: missing reference %q", name, ref)
				}
				escaped := strings.ReplaceAll(strings.ReplaceAll(target, "~", "~0"), "/", "~1")
				value[key] = "#/$defs/" + escaped + u.Fragment
			} else if err := rewriteReferences(child, name, documents); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := rewriteReferences(child, name, documents); err != nil {
				return err
			}
		}
	}
	return nil
}
