package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

var spacing = "  "

func main() {
	schemaBytes, err := ioutil.ReadFile("schema/heroku-connect.graphql")
	if err != nil {
		panic(err)
	}

	schemaStr := string(schemaBytes)
	source := &ast.Source{
		Name:  "schema.graphql",
		Input: schemaStr,
	}

	// Parse the schema file
	doc, parseErr := gqlparser.LoadSchema(source)
	if parseErr != nil {
		panic(parseErr)
	}

	// Traverse and process the AST (example: print type names)
	var enumOnly = ""
	var typesOnly = ""
	var objectsOnly = ""
	for _, def := range doc.Types {
		if def.Kind == ast.Enum {
			enumOnly += genEnum(def)
		}
		if def.Kind == ast.InputObject {
			typesOnly += genInputObject(def)
		}
		if def.Kind == ast.Object {
			objectsOnly += generateObject(def)
		}
	}
	writeFile("types.ts", getTypesHeader()+enumOnly+typesOnly+objectsOnly)
}

type TemplateField struct {
	Name        string
	Type        string
	Description string
}

func getTemplateField(field *ast.FieldDefinition) TemplateField {
	return TemplateField{
		Name:        field.Name,
		Type:        field.Type.String(),
		Description: field.Description,
	}
}

func getTemplateFields(def *ast.Definition) []TemplateField {
	var fields []TemplateField
	for _, field := range def.Fields {
		fields = append(fields, getTemplateField(field))
	}
	return fields
}

// tried to use templates, don't like white space it introduces
// and weird {{- }} and {{ -}} syntax I need to use
func genObjectViaTemplate(def *ast.Definition) string {
	type TemplateInput struct {
		Name        string
		Description string
		Fields      []TemplateField
	}
	srcTemplate := `
{{ if .Description }}/** {{.Description }} */{{end}}
export type {{.Name}} = {
{{- range .Fields }}
  {{ if .Description -}}
  /** {{.Description }} */
  {{- end }}
  {{.Name}}: {{.Type}};
{{- end}}
}
`
	templateInput := TemplateInput{}
	templateInput.Name = normalizedName(def.Name)
	templateInput.Description = def.Description
	templateInput.Fields = getTemplateFields(def)

	tmpl, err := template.New("test").Parse(srcTemplate)
	if err != nil {
		panic(err)
	}
	var outputBuffer bytes.Buffer
	if err := tmpl.Execute(&outputBuffer, templateInput); err != nil {
		panic(err)
	}
	return outputBuffer.String()
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
