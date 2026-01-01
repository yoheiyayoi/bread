package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExportedType holds info about an exported type from a Lua module
type ExportedType struct {
	Name     string
	Generics string // generic params like <T> or <T, S>, empty if none
}

// TypeExtractor scans Lua files for exported types
type TypeExtractor struct {
	exportTypePattern *regexp.Regexp
}

// NewTypeExtractor creates a new extractor
func NewTypeExtractor() *TypeExtractor {
	return &TypeExtractor{
		// matches "export type Foo" or "export type Foo<T>"
		// We'll parse the rest manually to handle nested generics
		exportTypePattern: regexp.MustCompile(`^\s*export\s+type\s+(\w+)`),
	}
}

// ExtractTypesFromFile reads a Lua file and pulls out all exported types
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
		loc := te.exportTypePattern.FindStringIndex(line)
		if loc != nil {
			// Found "export type Name"
			// Now look for generics starting after the name
			name := strings.Fields(line[loc[0]:loc[1]])[2] // export type Name -> Name is index 2
			rest := line[loc[1]:]

			generics := ""
			if strings.HasPrefix(strings.TrimSpace(rest), "<") {
				// Find the matching closing bracket
				startIdx := strings.Index(rest, "<")
				balance := 0
				for i, r := range rest[startIdx:] {
					if r == '<' {
						balance++
					} else if r == '>' {
						balance--
						if balance == 0 {
							generics = rest[startIdx : startIdx+i+1]
							break
						}
					}
				}
			}

			types = append(types, ExportedType{Name: name, Generics: generics})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return types, nil
}

// ExtractTypesFromPackage looks for the main entry file in a package and extracts types.
// Checks init.lua, init.luau, or files matching the package name.
func (te *TypeExtractor) ExtractTypesFromPackage(packageDir string, packageName string) ([]ExportedType, error) {
	var allTypes []ExportedType

	// check these files in order
	candidates := []string{
		filepath.Join(packageDir, "init.lua"),
		filepath.Join(packageDir, "init.luau"),
		filepath.Join(packageDir, packageName+".lua"),
		filepath.Join(packageDir, packageName+".luau"),
		filepath.Join(packageDir, "src", "init.lua"),
		filepath.Join(packageDir, "src", "init.luau"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			types, err := te.ExtractTypesFromFile(candidate)
			if err != nil {
				continue
			}
			allTypes = append(allTypes, types...)
			break // found it, we're done
		}
	}

	return te.deduplicateTypes(allTypes), nil
}

// deduplicateTypes removes duplicates while keeping the original order
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

// stripGenericDefaults strips default values from generics
// "<T, S = T>" becomes "<T, S>", "<Foo = Bar>" becomes "<Foo>"
func (te *TypeExtractor) stripGenericDefaults(generics string) string {
	if generics == "" {
		return ""
	}

	inner := strings.TrimPrefix(generics, "<")
	inner = strings.TrimSuffix(inner, ">")

	params := strings.Split(inner, ",")
	var cleanParams []string
	for _, param := range params {
		param = strings.TrimSpace(param)
		// chop off the default value if there is one
		if idx := strings.Index(param, "="); idx != -1 {
			param = strings.TrimSpace(param[:idx])
		}
		cleanParams = append(cleanParams, param)
	}

	return "<" + strings.Join(cleanParams, ", ") + ">"
}

// GenerateTypeReExports builds the type re-export statements for a module
func (te *TypeExtractor) GenerateTypeReExports(types []ExportedType, moduleName string) string {
	if len(types) == 0 {
		return ""
	}

	var lines []string
	for _, t := range types {
		rightSide := moduleName + "." + t.Name + te.stripGenericDefaults(t.Generics)
		line := fmt.Sprintf("export type %s%s = %s", t.Name, t.Generics, rightSide)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n") + "\n"
}

// GenerateLinkFileWithTypes builds a complete link file with require and type re-exports
func (te *TypeExtractor) GenerateLinkFileWithTypes(
	requirePath string,
	types []ExportedType,
	moduleName string,
	fullName string,
) string {
	var lines []string

	lines = append(lines, "--Bread")
	lines = append(lines, fmt.Sprintf("--%s", fullName))
	lines = append(lines, fmt.Sprintf("local %s = %s", moduleName, requirePath))

	typeExports := te.GenerateTypeReExports(types, moduleName)
	if typeExports != "" {
		lines = append(lines, strings.TrimSuffix(typeExports, "\n"))
	}

	lines = append(lines, "return "+moduleName)

	return strings.Join(lines, "\n") + "\n"
}
