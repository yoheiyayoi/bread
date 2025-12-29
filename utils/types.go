package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExportedType represents a single exported type from a Lua module
type ExportedType struct {
	Name string
}

// TypeExtractor handles scanning Lua files for exported types
type TypeExtractor struct {
	// Regex pattern to match "export type TypeName = ..."
	exportTypePattern *regexp.Regexp
}

// NewTypeExtractor creates a new TypeExtractor instance
func NewTypeExtractor() *TypeExtractor {
	return &TypeExtractor{
		// Match: export type TypeName = ... or export type TypeName<T> = ...
		// Captures the type name (including generic parameters)
		exportTypePattern: regexp.MustCompile(`^\s*export\s+type\s+(\w+)(?:<[^>]*>)?\s*=`),
	}
}

// ExtractTypesFromFile scans a single Lua file and returns all exported type names
func (te *TypeExtractor) ExtractTypesFromFile(filePath string) ([]ExportedType, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var types []ExportedType
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		matches := te.exportTypePattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			types = append(types, ExportedType{Name: matches[1]})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return types, nil
}

// ExtractTypesFromPackage scans a package directory for exported types
// It looks for init.lua, init.luau, or a file matching the package name
func (te *TypeExtractor) ExtractTypesFromPackage(packageDir string, packageName string) ([]ExportedType, error) {
	var allTypes []ExportedType

	// Priority order for main module files
	candidates := []string{
		filepath.Join(packageDir, "init.lua"),
		filepath.Join(packageDir, "init.luau"),
		filepath.Join(packageDir, packageName+".lua"),
		filepath.Join(packageDir, packageName+".luau"),
		filepath.Join(packageDir, "src", "init.lua"),
		filepath.Join(packageDir, "src", "init.luau"),
	}

	// Try each candidate file
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			types, err := te.ExtractTypesFromFile(candidate)
			if err != nil {
				continue
			}
			allTypes = append(allTypes, types...)
			// Found main entry file, stop looking
			break
		}
	}

	// Remove duplicates
	return te.deduplicateTypes(allTypes), nil
}

// deduplicateTypes removes duplicate type names while preserving order
func (te *TypeExtractor) deduplicateTypes(types []ExportedType) []ExportedType {
	seen := make(map[string]bool)
	var result []ExportedType

	for _, t := range types {
		if !seen[t.Name] {
			seen[t.Name] = true
			result = append(result, t)
		}
	}

	return result
}

// GenerateTypeReExports generates Luau code that re-exports types from a module
func (te *TypeExtractor) GenerateTypeReExports(types []ExportedType, moduleName string) string {
	if len(types) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, t := range types {
		sb.WriteString("export type ")
		sb.WriteString(t.Name)
		sb.WriteString(" = ")
		sb.WriteString(moduleName)
		sb.WriteString(".")
		sb.WriteString(t.Name)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateLinkFileWithTypes generates a complete link file with type re-exports
func (te *TypeExtractor) GenerateLinkFileWithTypes(
	requirePath string,
	types []ExportedType,
	moduleName string,
) string {
	var sb strings.Builder

	// Local variable for the required module
	sb.WriteString("local ")
	sb.WriteString(moduleName)
	sb.WriteString(" = ")
	sb.WriteString(requirePath)
	sb.WriteString("\n")

	// Type re-exports
	typeExports := te.GenerateTypeReExports(types, moduleName)
	if typeExports != "" {
		sb.WriteString(typeExports)
	}

	// Return the module
	sb.WriteString("return ")
	sb.WriteString(moduleName)
	sb.WriteString("\n")

	return sb.String()
}
