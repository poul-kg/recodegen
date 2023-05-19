package main

import (
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

var spacing = "  "

func main() {
	schemaAst := getSchemaAst("schema/heroku-connect.graphql")
	generateSchema(schemaAst, "types.ts")

	files := findFiles("schema/services/*.ts")
	srcOps := extractOperationsFromFiles(files)
	dstOps := generateOperations(schemaAst, srcOps)
	appendToFile("types.ts", dstOps)
}

func findFiles(pattern string) []string {
	var output []string
	fsys := os.DirFS(".")
	matches, _ := doublestar.Glob(fsys, pattern)
	for _, fileName := range matches {
		output = append(output, fileName)
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
	schemaBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	fileContentStr := string(schemaBytes)
	return fileContentStr
}

func getSchemaAst(inputFileName string) *ast.Schema {
	schemaStr := getFileContent(inputFileName)
	source := &ast.Source{
		Name:  inputFileName,
		Input: schemaStr,
	}

	// Parse the schema file
	schemaAst, parseErr := gqlparser.LoadSchema(source)
	if parseErr != nil {
		panic(parseErr)
	}
	return schemaAst
}

func generateSchema(schemaAst *ast.Schema, outputFileName string) {
	// Traverse and process the AST (example: print type names)
	var enumOnly = ""
	var typesOnly = ""
	var objectsOnly = ""
	for _, def := range schemaAst.Types {
		if def.Kind == ast.Enum {
			enumOnly += genEnum(def)
		}
		if def.Kind == ast.InputObject {
			typesOnly += genInputObject(def)
		}
		if def.Kind == ast.Object {
			objectsOnly += generateObject(def)
		}
		if def.Kind == ast.Interface {
			//fmt.Printf("%s\n", def.Name)
		}
	}
	writeFile(outputFileName, getTypesHeader()+enumOnly+typesOnly+objectsOnly)
}

func generateOperations(schemaAst *ast.Schema, queryStr string) string {
	output := ""
	// Parse the schema file
	astQuery, parseErr := gqlparser.LoadQuery(schemaAst, queryStr)
	if parseErr != nil {
		panic(parseErr)
	}

	// Traverse and process the AST (example: print type names)
	for _, op := range astQuery.Operations {
		output += generateOperationStr(op)
	}
	return output
}

func generateOperationStr(astOp *ast.OperationDefinition) string {
	return generateOperationVars(astOp) + "\n" + generateOperation(astOp)
}

func generateOperationVars(astOp *ast.OperationDefinition) string {
	operationVars := "export type " + UcFirst(astOp.Name) + UcFirst(string(astOp.Operation)) + "Variables = Exact<{\n"
	for _, varDef := range astOp.VariableDefinitions {
		operationVars += spacing + generateVariable(varDef) + "\n"
	}
	operationVars += "}>;"
	return operationVars
}

func generateOperation(astOp *ast.OperationDefinition) string {
	output := "export type " + UcFirst(astOp.Name) + UcFirst(string(astOp.Operation)) + " = Exact<{\n"
	for _, selection := range astOp.SelectionSet {
		output += generateOpField(selection)
	}
	output += "}>;\n"
	return output
}

func generateOpField(selection ast.Selection) string {
	astField, ok := selection.(*ast.Field)
	if !ok {
		panic("Unable to cast")
	}
	output := ""
	if astField.SelectionSet == nil {
		output += spacing + generateOpFieldName(astField) + ": " + generateOpFieldType(astField.Definition.Type)
	} else {
		opStr := ""
		closeStr := ""
		if astField.Definition.Type.NamedType == "" {
			// array
			opStr += "Array<{\n"
			closeStr += "}>\n"
		} else {
			// object
			opStr += "{\n"
			closeStr += "}\n"
		}
		output += generateFieldName(astField.Definition) + ": " + opStr
		for _, selection := range astField.SelectionSet {
			output += spacing + generateOpField(selection) + ";\n"
		}
		output += closeStr
	}
	return output
}

func generateVariable(varDef *ast.VariableDefinition) string {
	output := varDef.Variable
	if varDef.Type.NonNull == false {
		output += "?: "
	} else {
		output += ": "
	}
	output += generateFieldType(varDef.Type) + ";"
	return output
}

// Upper Case first letter of a string
func UcFirst(input string) string {
	r, size := utf8.DecodeRuneInString(input)
	input = strings.ToUpper(string(r)) + input[size:]
	return input
}

func generateObject(def *ast.Definition) string {
	objectName := normalizedName(def.Name)
	desc := generateDesc(def.Description)
	header := fmt.Sprintf("\nexport type %s = {", objectName)
	if len(desc) > 0 {
		header = "\n" + desc + header
	}
	body := ""
	footer := "\n};\n"

	for _, field := range def.Fields {
		if field.Name == "__schema" {
			continue
		}
		if field.Name == "__type" {
			continue
		}
		if len(field.Description) > 0 {
			body += "\n" + spacing + generateDesc(field.Description)
		}
		body += "\n" + spacing + generateFieldName(field) + ": " + generateFieldType(field.Type) + ";"
	}
	return header + body + footer
}

func generateDesc(desc string) string {
	if len(desc) > 0 {
		return fmt.Sprintf("/** %s */", desc)
	}
	return ""
}

func genEnum(def *ast.Definition) string {
	var tmpl = `export enum %s {
%s
}

`
	desc := generateDesc(def.Description)
	if len(desc) > 0 {
		tmpl = desc + "\n" + tmpl
	}
	var enumValues []string
	for _, enumVal := range def.EnumValues {
		desc := ""
		if len(enumVal.Description) > 0 {
			desc = fmt.Sprintf("%s/** %s */", spacing, enumVal.Description)
		}
		// /** desc */
		// enumName = 'enumValue'
		enumItem := fmt.Sprintf("%s\n%s%s = '%s',", desc, spacing, getEnumItemName(enumVal.Name), enumVal.Name)
		enumValues = append(enumValues, enumItem)
	}
	enumName := normalizedName(def.Name)
	return fmt.Sprintf(tmpl, enumName, strings.Join(enumValues, "\n"))
}

func normalizedName(snakeCase string) string {
	words := strings.Split(snakeCase, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}

	upperCamelCase := strings.Join(words, "_")
	return upperCamelCase
}

func getEnumItemName(snakeCase string) string {
	words := strings.Split(snakeCase, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}

	upperCamelCase := strings.Join(words, "")
	return upperCamelCase
}

func getTypesHeader() string {
	return `export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  bigint: any;
  date: any;
  float8: any;
  timestamp: any;
  timestamptz: any;
};

`
}

func writeFile(fileName string, data string) {
	f, err := os.Create(fileName)

	if err != nil {
		log.Fatal(err)
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	_, err2 := f.WriteString(data)

	if err2 != nil {
		log.Fatal(err2)
	}
}

func appendToFile(fileName string, data string) {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	_, err = f.WriteString(data)
	if err != nil {
		panic(err)
	}
}

func genInputObject(def *ast.Definition) string {
	desc := genDesc(def)
	start := desc + "export type " + normalizedName(def.Name) + " = {\n"
	var fields []string
	for _, field := range def.Fields {
		fieldStr := "  " + generateFieldName(field) + ": " + genInputFieldType(field.Type) + ";\n"
		fields = append(fields, fieldStr)
	}
	return start + strings.Join(fields, "") + "};\n\n"
}

func generateFieldType(astType *ast.Type) string {
	normalName := wrapScalar(normalizedName(astType.Name()))

	if astType.NamedType != "" {
		if astType.NonNull == false {
			normalName = "Maybe<" + normalName + ">"
		}
		return normalName
	}

	if astType.NonNull == false {
		normalName = "Maybe<Array<" + normalName + ">>"
	}

	return "Array<" + normalName + ">"
}

func generateOpFieldType(astType *ast.Type) string {
	normalName := wrapOpScalar(astType.Name())

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

// given type like "Int" will wrap it into "Scalars['Int']"
// if given type is not scalar, return as is
func wrapScalar(typeName string) string {
	scalars := []string{"Boolean", "String", "Int", "Float8", "Float", "Bigint", "Timestamp", "Timestamptz"}
	for _, scalar := range scalars {
		if typeName == scalar {
			switch scalar {
			case "Bigint":
				fallthrough
			case "Float8":
				fallthrough
			case "Timestamp":
				fallthrough
			case "Timestamptz":
				return "Scalars['" + strings.ToLower(typeName) + "']"
			}
			return "Scalars['" + typeName + "']"
		}
	}
	return typeName
}

func wrapOpScalar(typeName string) string {
	typeName = normalizedName(typeName)
	scalars := []string{"Boolean", "String", "Int", "Float8", "Float", "Bigint", "Timestamp", "Timestamptz"}
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
	return typeName
}

func genInputFieldType(astType *ast.Type) string {
	normalName := wrapScalar(normalizedName(astType.Name()))

	if astType.NamedType != "" {
		if astType.NonNull == false {
			normalName = "InputMaybe<" + normalName + ">"
		}
		return normalName
	}

	if astType.NonNull == false {
		normalName = "InputMaybe<Array<" + normalName + ">>"
	}

	return "Array<" + normalName + ">"
}

func generateFieldName(astFieldDef *ast.FieldDefinition) string {
	return astFieldDef.Name + genNullable(astFieldDef)
}

func generateOpFieldName(astField *ast.Field) string {
	if astField.Definition.Type.NonNull == false {
		return astField.Alias + "?"
	}
	return astField.Alias
}

func genNullable(astFieldDef *ast.FieldDefinition) string {
	if astFieldDef.Type.NonNull {
		return ""
	}
	return "?"
}

func genDesc(astDef *ast.Definition) string {
	if len(astDef.Description) > 0 {
		return "/** " + astDef.Description + " */\n"
	}
	return ""
}
