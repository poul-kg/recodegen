package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

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
	var enumOnly string
	var typesOnly string
	for _, def := range doc.Types {
		if def.Kind == ast.Enum {
			enumOnly += genEnum(def)
		}
		if def.Kind == ast.InputObject {
			typesOnly += genInputObject(def)
		}
	}
	writeFile("types.ts", getTypesHeader()+enumOnly+typesOnly)
}

func genEnum(def *ast.Definition) string {
	var template = `export enum %s {
%s
}

`
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
	enumName := getTypeName(def.Name)
	return fmt.Sprintf(template, enumName, strings.Join(enumValues, "\n"))
}

func getTypeName(snakeCase string) string {
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

	defer f.Close()

	_, err2 := f.WriteString(data)

	if err2 != nil {
		log.Fatal(err2)
	}
}

func genInputObject(def *ast.Definition) string {
	desc := genDesc(def)
	start := desc + "export type " + getTypeName(def.Name) + " = {\n"
	var fields []string
	for _, field := range def.Fields {
		fieldStr := "  " + getFieldName(field) + ": " + genInputType(field.Type) + ";\n"
		fields = append(fields, fieldStr)
	}
	return start + strings.Join(fields, "") + "};\n\n"
}

func genInputType(astType *ast.Type) string {
	if astType.String() == "[Boolean!]" {
		return "InputMaybe<Array<Scalars['Boolean']>>"
	}
	if astType.String() == "Boolean" {
		return "InputMaybe<Scalars['Boolean']>"
	}
	if astType.String() == "[String!]" {
		return "InputMaybe<Array<Scalars['String']>>"
	}
	if astType.String() == "String" {
		return "InputMaybe<Scalars['String']>"
	}
	if astType.String() == "Int!" {
		return "Scalars['Int']"
	}
	if astType.String() == "Int" {
		return "InputMaybe<Scalars['Int']>"
	}
	if astType.String() == "[Int!]" {
		return "InputMaybe<Array<Scalars['Int']>>"
	}
	if astType.String() == "float8" {
		return "Maybe<Scalars['float8']>"
	}
	if astType.String() == "[float8!]" {
		return "InputMaybe<Array<Scalars['float8']>>"
	}
	if astType.String() == "bigint" {
		return "Maybe<Scalars['bigint']>"
	}
	if astType.String() == "[bigint!]" {
		return "InputMaybe<Array<Scalars['bigint']>>"
	}
	if astType.String() == "timestamp" {
		return "Maybe<Scalars['timestamp']>"
	}
	if astType.String() == "[timestamp!]" {
		return "InputMaybe<Array<Scalars['timestamp']>>"
	}
	if astType.String() == "timestamptz" {
		return "Maybe<Scalars['timestamptz']>"
	}
	if astType.String() == "[timestamptz!]" {
		return "InputMaybe<Array<Scalars['timestamptz']>>"
	}

	return getTypeName(astType.Name())
}

func getFieldName(astFieldDef *ast.FieldDefinition) string {
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
