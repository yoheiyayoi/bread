package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTypesFromFile(t *testing.T) {
	// Create a temporary Lua file with exported types
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.lua")

	content := `
-- Some comments
local module = {}

export type Config = {
	name: string,
	value: number,
}

export type Handler = (input: string) -> boolean

-- Regular function
function module.doSomething()
	return true
end

export type Result<T> = {
	success: boolean,
	data: T?,
	error: string?,
}

return module
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	extractor := NewTypeExtractor()
	types, err := extractor.ExtractTypesFromFile(testFile)

	if err != nil {
		t.Fatalf("ExtractTypesFromFile failed: %v", err)
	}

	expectedTypes := []string{"Config", "Handler", "Result"}
	if len(types) != len(expectedTypes) {
		t.Errorf("Expected %d types, got %d", len(expectedTypes), len(types))
	}

	for i, expected := range expectedTypes {
		if i >= len(types) {
			break
		}
		if types[i].Name != expected {
			t.Errorf("Expected type %s, got %s", expected, types[i].Name)
		}
	}
}

func TestExtractTypesFromPackage(t *testing.T) {
	// Create a temporary package structure
	tmpDir := t.TempDir()
	packageDir := filepath.Join(tmpDir, "mypackage")
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		t.Fatalf("Failed to create package dir: %v", err)
	}

	// Create init.lua with types
	initContent := `
local Package = {}

export type MyType = {
	id: number,
	name: string,
}

export type Callback = () -> ()

return Package
`
	if err := os.WriteFile(filepath.Join(packageDir, "init.lua"), []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create init.lua: %v", err)
	}

	extractor := NewTypeExtractor()
	types, err := extractor.ExtractTypesFromPackage(packageDir, "mypackage")

	if err != nil {
		t.Fatalf("ExtractTypesFromPackage failed: %v", err)
	}

	if len(types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(types))
	}
}

func TestGenerateLinkFileWithTypes(t *testing.T) {
	extractor := NewTypeExtractor()

	types := []ExportedType{
		{Name: "Config"},
		{Name: "Handler"},
		{Name: "Result"},
	}

	requirePath := `require(script.Parent._Index["author_package@1.0.0"]["package"])`
	result := extractor.GenerateLinkFileWithTypes(requirePath, types, "_Package")

	expected := `local _Package = require(script.Parent._Index["author_package@1.0.0"]["package"])
export type Config = _Package.Config
export type Handler = _Package.Handler
export type Result = _Package.Result
return _Package
`

	if result != expected {
		t.Errorf("Generated link file doesn't match expected.\nGot:\n%s\nExpected:\n%s", result, expected)
	}
}

func TestGenerateLinkFileWithNoTypes(t *testing.T) {
	extractor := NewTypeExtractor()

	var types []ExportedType

	requirePath := `require(script.Parent._Index["author_package@1.0.0"]["package"])`
	result := extractor.GenerateLinkFileWithTypes(requirePath, types, "_Package")

	expected := `local _Package = require(script.Parent._Index["author_package@1.0.0"]["package"])
return _Package
`

	if result != expected {
		t.Errorf("Generated link file doesn't match expected.\nGot:\n%s\nExpected:\n%s", result, expected)
	}
}

func TestDeduplicateTypes(t *testing.T) {
	extractor := NewTypeExtractor()

	types := []ExportedType{
		{Name: "Foo"},
		{Name: "Bar"},
		{Name: "Foo"}, // duplicate
		{Name: "Baz"},
		{Name: "Bar"}, // duplicate
	}

	result := extractor.deduplicateTypes(types)

	if len(result) != 3 {
		t.Errorf("Expected 3 unique types, got %d", len(result))
	}

	expectedOrder := []string{"Foo", "Bar", "Baz"}
	for i, expected := range expectedOrder {
		if result[i].Name != expected {
			t.Errorf("Expected type %s at position %d, got %s", expected, i, result[i].Name)
		}
	}
}
