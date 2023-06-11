package typescript

import (
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"regexp"
	"sort"
	"strings"
)

const spacing = "  "

type Schema struct {
	Ast *ast.Schema
}

func (schema *Schema) String() string {
	// Traverse and process the AST (example: print type names)
	var enumOnly = ""
	var typesOnly = ""
	var objectsOnly = ""
	sortedTypeKeys := schema.getSortedTypeKeys()
	for _, key := range *sortedTypeKeys {
		def := schema.Ast.Types[key]
		if def.Kind == ast.Enum {
			enumOnly += genEnum(def)
		}
		if def.Kind == ast.InputObject {
			typesOnly += genInputObject(def)
		}
		if def.Kind == ast.Object {
			objectsOnly += schema.generateObject(def)
		}
		//if def.Kind == ast.Interface {
		//	fmt.Printf("Interface: %s\n", def.Name)
		//}
		//if def.Kind == ast.Union {
		//	fmt.Printf("Union: %s\n", def.Name)
		//}
		//if def.Kind == ast.Scalar {
		//fmt.Printf("Scalar: %s\n", def.Name)
		//}
	}
	return schema.getTypesHeader() + enumOnly + typesOnly + objectsOnly
}

func (schema *Schema) getSortedTypeKeys() *[]string {
	// Create a slice to hold the keys
	keys := make([]string, 0, len(schema.Ast.Types))
	for k := range schema.Ast.Types {
		keys = append(keys, k)
	}

	// Sort the keys
	sort.Strings(keys)

	return &keys
}

func (schema *Schema) getTypesHeader() string {
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
  Bigint: any;
  date: any;
  Date: any;
  Float8: any;
  Timestamp: any;
  Timestamptz: any;
  Json: any;
  Jsonb: any;
  Numeric: any;
  Point: any;
  Polygon: any;
  Uuid: any;
};

`
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
			desc = fmt.Sprintf("%s/** %s */\n", spacing, enumVal.Description)
		}
		// /** desc */
		// enumName = 'enumValue'
		enumItem := fmt.Sprintf("%s\n%s%s = '%s',", desc, spacing, getEnumItemName(enumVal.Name), enumVal.Name)
		enumValues = append(enumValues, enumItem)
	}
	enumName := normalizedName(def.Name)
	return fmt.Sprintf(tmpl, enumName, strings.Join(enumValues, "\n"))
}

func generateDesc(desc string) string {
	if len(desc) > 0 {
		return fmt.Sprintf("/** %s */", desc)
	}
	return ""
}

func normalizedName(snakeCase string) string {
	words := strings.Split(snakeCase, "_")
	caser := cases.Title(language.English, cases.NoLower)
	for i, word := range words {
		words[i] = caser.String(word)
	}

	upperCamelCase := strings.Join(words, "_")
	return upperCamelCase
}

func genInputObject(def *ast.Definition) string {
	desc := generateDesc(def.Description)
	if len(desc) > 0 {
		desc += "\n"
	}
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
		return normalName
	}

	return "Array<" + normalName + ">"
}

// given type like "Int" will wrap it into "Scalars['Int']"
// if given type is not scalar, return as is
func wrapScalar(typeName string) string {
	scalars := []string{
		"Boolean", "String", "Int", "Float8", "Float", "Bigint", "Timestamp", "Timestamptz",
		"Numeric", "Uuid", "Json", "Jsonb", "Polygon", "Point", "Date", "date",
	}
	for _, scalar := range scalars {
		if typeName == scalar {
			return "Scalars['" + typeName + "']"
		}
	}
	return typeName
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
		return normalName
	}

	return "Array<" + normalName + ">"
}

func getEnumItemName(snakeCase string) string {
	//re := regexp.MustCompile("_([A-Za-z])")
	//words := strings.Split(snakeCase, "_")
	words := splitUnderscoredString(snakeCase)
	for i, word := range words {
		caser := cases.Title(language.English)
		words[i] = caser.String(word)
	}

	upperCamelCase := strings.Join(words, "")
	return upperCamelCase
}

func splitUnderscoredString(snakeCase string) []string {
	// Compile the expression
	re := regexp.MustCompile("_([A-Za-z])") // match underscore when followed by a letter

	// Replace the underscore with a special character
	replaced := re.ReplaceAllString(snakeCase, "|$1")

	// Split the string on the special character
	split := strings.Split(replaced, "|")
	return split
}

func (schema *Schema) generateObject(def *ast.Definition) string {
	objectName := normalizedName(def.Name)
	desc := generateDesc(def.Description)
	header := fmt.Sprintf("\nexport type %s = {", objectName)
	args := ""
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
		if len(field.Arguments) > 0 {
			args += generateArgDefinition(objectName, field.Name, field.Arguments)
		}
		if len(field.Description) > 0 {
			body += "\n" + spacing + generateDesc(field.Description)
		}
		body += "\n" + spacing + generateFieldName(field) + ": " + generateFieldType(field.Type) + ";"
	}
	return header + body + footer + args
}

func generateArgDefinition(parentName string, fieldName string, args []*ast.ArgumentDefinition) string {
	output := "export type " + parentName + normalizedName(fieldName) + "Args = {\n"
	fields := ""
	for _, arg := range args {
		if len(arg.Description) > 0 {
			fields += spacing + "/** " + arg.Description + " */\n"
		}
		fields += spacing + arg.Name
		if arg.Type.NonNull == false {
			fields += "?: "
		} else {
			fields += ": "
		}
		fields += genInputFieldType(arg.Type) + ";\n"
	}
	output += fields + "};\n\n"
	return output
}
