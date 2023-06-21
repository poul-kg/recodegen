package typescript

import (
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"os"
	"recodegen/config"
	"regexp"
	"strings"
	"unicode/utf8"
)

const defExportName = "Types"

type Operations struct {
	Config    config.CodegenSchemaEntryConfig
	Ast       *ast.Schema
	fragments map[string]*ast.FragmentSpread
}

func (operations *Operations) String() string {
	typesPath := ""
	if operations.Config.Preset == "import-types" {
		typesPath = operations.Config.PresetConfig["typesPath"]
	}
	files := findFiles(operations.Config.Documents)
	//fmt.Printf("%s\n", files)
	srcOps := extractOperationsFromFiles(files)
	dstOps := operations.generateOperations(srcOps, typesPath)
	return dstOps
}

func findFiles(patterns []string) []string {
	var output []string
	fsys := os.DirFS(".")
	for _, pattern := range patterns {
		matches, _ := doublestar.Glob(fsys, pattern)
		for _, fileName := range matches {
			output = append(output, fileName)
		}
	}
	return output
}

func extractOperationsFromFiles(fileNames []string) string {
	// For each file...
	output := ""
	for _, fileName := range fileNames {
		output += findOperationInFile(fileName)
	}
	return output
}

func findOperationInFile(fileName string) string {
	output := ""
	fileContent := getFileContent(fileName)
	re := regexp.MustCompile("(?s)gql`(.*?)`")
	matches := re.FindAllStringSubmatch(fileContent, -1)
	if len(matches) == 0 {
		return output
	}
	for _, match := range matches {
		output += match[1]
	}
	return output
}

func getFileContent(fileName string) string {
	schemaBytes, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	return string(schemaBytes)
}

func (operations *Operations) generateOperations(queryStr string, typesPath string) string {
	output := ""
	// Parse the schema file
	astQuery, parseErr := gqlparser.LoadQuery(operations.Ast, queryStr)
	if parseErr != nil {
		fmt.Printf("Error while scanning: %s\n", operations.Config.Documents)
		panic(parseErr)
	}
	isImportTypes := false
	if typesPath != "" {
		isImportTypes = true
	}
	// Traverse and process the AST (example: print type names)
	for _, op := range astQuery.Operations {
		output += operations.generateOperationStr(op, isImportTypes)
	}
	if typesPath != "" {
		output = "import * as Types from \"" + typesPath + "\";\n" + output
	}
	return output
}

func (operations *Operations) generateOperationStr(astOp *ast.OperationDefinition, isImportTypes bool) string {
	operations.fragments = make(map[string]*ast.FragmentSpread)
	opVars := generateOperationVars(astOp, isImportTypes)
	ops := operations.generateOperation(astOp, isImportTypes)
	opFragments := operations.generateFragmentTypes(isImportTypes)
	return opFragments + opVars + ops
}

func (operations *Operations) generateFragmentTypes(isImportTypes bool) string {
	output := ""
	for _, fragment := range operations.fragments {
		output += "export type " + fragment.Name + "Fragment = {\n"
		output += operations.generateFragmentSpreadField(fragment, isImportTypes)
		output += "\n};\n"
	}
	return output
}

//func generateSpreadFragmentType(astOp *ast.OperationDefinition, isImportTypes bool) string {
//	output := ""
//	if astOp.SelectionSet != nil {
//		fragments := findSpreadFragments(astOp.SelectionSet)
//		if len(fragments) == 0 {
//			return output
//		}
//		for _, fragment := range fragments {
//			output += "export type " + fragment.Name + "Fragment = {\n"
//			output += generateFragmentSpreadField(fragment, isImportTypes)
//			output += "\n};\n"
//		}
//	}
//	return output
//}

//func findSpreadFragments(selectionSet ast.SelectionSet) []*ast.FragmentSpread {
//	var arr []*ast.FragmentSpread
//	for _, selection := range selectionSet {
//		fragment, isFragment := selection.(*ast.FragmentSpread)
//		if isFragment {
//			arr = append(arr, fragment)
//		}
//		field, isField := selection.(*ast.Field)
//		if isField {
//			if field.SelectionSet != nil {
//				arr = append(arr, findSpreadFragments(field.SelectionSet)...)
//			}
//		}
//	}
//	return arr
//}

func generateOperationVars(astOp *ast.OperationDefinition, isImportTypes bool) string {
	operationVars := ""
	if isImportTypes {
		operationVars = "export type " + normalizeOpName(astOp.Name) + UcFirst(string(astOp.Operation)) +
			"Variables = " + defExportName + ".Exact<{\n"
	} else {
		operationVars = "export type " + normalizeOpName(astOp.Name) + UcFirst(string(astOp.Operation)) + "Variables = Exact<{\n"
	}

	for _, varDef := range astOp.VariableDefinitions {
		operationVars += spacing + generateVariable(varDef, isImportTypes) + "\n"
	}
	operationVars += "}>;"
	return operationVars
}

func (operations *Operations) generateOperation(astOp *ast.OperationDefinition, isImportTypes bool) string {
	output := ""
	if isImportTypes {
		output = "export type " + normalizeOpName(astOp.Name) + UcFirst(string(astOp.Operation)) + " = " +
			defExportName + ".Exact<{\n"
	} else {
		output = "export type " + normalizeOpName(astOp.Name) + UcFirst(string(astOp.Operation)) + " = Exact<{\n"
	}
	if astOp.Operation == "query" {
		output += "__typename?: 'query_root',\n"
	}
	if astOp.Operation == "mutation" {
		output += "__typename?: 'mutation_root',\n"
	}
	for _, selection := range astOp.SelectionSet {
		// ast.Selection
		innerAstField, isField := selection.(*ast.Field)
		if isField {
			output += operations.generateOpField(innerAstField, isImportTypes)
		}
	}
	output += "}>;\n"
	return output
}

func (operations *Operations) generateFragmentSpreadField(astFragmentSpread *ast.FragmentSpread, isImportTypes bool) string {
	output := ""
	// list of fields: fieldName: fieldType
	for _, fragment := range astFragmentSpread.Definition.SelectionSet {
		innerAstField, isField := fragment.(*ast.Field)
		if isField {
			output += operations.generateOpField(innerAstField, isImportTypes) + "\n"
		}
	}
	//__typename: astFragmentSpread.Definition.TypeCondition

	return output
}

//func generateFragmentFieldName()

func (operations *Operations) generateOpField(astField *ast.Field, isImportTypes bool) string {
	if astField.Name == "__typename" {
		return ""
	}
	output := ""
	if astField.SelectionSet == nil {
		output += spacing + generateOpFieldName(astField) + ": " +
			generateOpFieldType(astField.Definition.Type, isImportTypes) + ";"
	} else {
		opStr := ""
		closeStr := ""
		if astField.Definition.Type.NamedType == "" {
			// array
			opStr += "Array<{\n"
			closeStr += "}>;\n"
		} else {
			// object
			opStr += "{\n"
			closeStr += "};\n"
		}
		output += generateOpFieldName(astField) + ": " + opStr
		output += "__typename?: '" + getUnderscoreTypeName(astField) + "',\n"
		for _, selection := range astField.SelectionSet {
			innerAstField, isField := selection.(*ast.Field)
			if isField {
				output += operations.generateOpField(innerAstField, isImportTypes)
			} else {
				astFragmentSpread, isFragmentSpread := selection.(*ast.FragmentSpread)
				if isFragmentSpread {
					operations.addSpreadFragment(astFragmentSpread)
					output += operations.generateFragmentSpreadField(astFragmentSpread, isImportTypes)
				}
			}
		}
		output += closeStr
	}
	return output
}

func getUnderscoreTypeName(astField *ast.Field) string {
	if astField.Definition.Type.NamedType != "" {
		return astField.Definition.Type.NamedType
	} else if astField.Definition.Type.Elem.NamedType != "" {
		return astField.Definition.Type.Elem.NamedType
	} else {
		return astField.Name
	}
}

func (operations *Operations) addSpreadFragment(fragment *ast.FragmentSpread) {
	if operations.fragments == nil {
		operations.fragments = make(map[string]*ast.FragmentSpread)
	}
	operations.fragments[fragment.Name] = fragment
}

func generateOpFieldName(astField *ast.Field) string {
	if astField.Definition.Type.NonNull == false {
		return astField.Alias + "?"
	}
	return astField.Alias
}

// Upper Case first letter of a string + convert UPPERCASE words to Uppsercase instead
func UcFirst(input string) string {
	r, size := utf8.DecodeRuneInString(input)
	input = strings.ToUpper(string(r)) + input[size:]

	input = strings.ReplaceAll(input, "ID", "Id")

	//// Regex to match uppercase sub-words
	//re := regexp.MustCompile(`[A-Z][A-Z]+`)
	//
	//// Function to convert a match to title case
	//convertToTitleCase := func(m string) string {
	//	return strings.Title(strings.ToLower(m))
	//}
	//
	//// Apply the function to all matches
	//input = re.ReplaceAllStringFunc(input, convertToTitleCase)

	return input
}

func normalizeOpName(opName string) string {
	return UcFirst(opName)
}

func fixTitleCase(input string) string {
	// Regex to match uppercase sub-words
	re := regexp.MustCompile(`[A-Z][A-Z]+`)

	// Function to convert a match to title case
	convertToTitleCase := func(m string) string {
		return strings.Title(strings.ToLower(m))
	}

	// Apply the function to all matches
	output := re.ReplaceAllStringFunc(input, convertToTitleCase)

	return output
}

func generateVariable(varDef *ast.VariableDefinition, isImportTypes bool) string {
	output := varDef.Variable
	if varDef.Type.NonNull == false {
		output += "?: "
	} else {
		output += ": "
	}
	if isImportTypes {
		output += generateFieldTypeImported(varDef.Type) + ";"
	} else {
		output += generateFieldType(varDef.Type) + ";"
	}
	return output
}

func generateFieldTypeImported(astType *ast.Type) string {
	normalName := wrapScalar(astType.Name())
	normalName = defExportName + "." + normalName

	if astType.NamedType != "" {
		if astType.NonNull == false {
			normalName = defExportName + ".Maybe<" + normalName + ">"
		}
		return normalName
	}

	if astType.NonNull == false {
		normalName = defExportName + ".Maybe<Array<" + normalName + ">>"
		return normalName
	}

	return "Array<" + normalName + ">"
}

func generateOpFieldType(astType *ast.Type, isImportTypes bool) string {
	normalName := wrapOpScalar(astType.Name(), isImportTypes)

	if astType.NamedType != "" {
		if astType.NonNull == false {
			normalName = normalName + " | " + "null"
		}
		return normalName
	}

	if astType.NonNull == false {
		normalName = normalName + "[] | null"
	}

	return normalName + "[]"
}

func wrapOpScalar(typeName string, isImportTypes bool) string {
	typeName = normalizedName(typeName)
	scalars := []string{
		"Boolean", "String", "Int", "Float8", "Float", "Bigint", "Timestamp", "Timestamptz",
		"Numeric", "Uuid", "Json", "Jsonb", "Polygon", "Point", "Date", "date",
	}
	for _, scalar := range scalars {
		if typeName == scalar {
			switch scalar {
			case "Boolean":
				return "boolean"
			case "String":
				return "string"
			case "Int":
				return "number"
			case "Float8":
				return "number"
			case "Float":
				return "number"
			case "Bigint":
				return "number"
			case "Timestamp":
				return "string"
			case "Timestamptz":
				return "string"
			}
			return "any"
		}
	}
	if isImportTypes {
		typeName = "Types." + typeName
	}
	return typeName
}
