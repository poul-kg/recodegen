package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadConfigFromFile(fileName string) CodegenConfig {
	configData, _ := os.ReadFile(fileName)

	var codegenSchema CodegenConfig

	err := json.Unmarshal(configData, &codegenSchema)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Unable to parse JSON in config file: " + fileName)
		os.Exit(1)
	}

	return codegenSchema
}

// Structs schema.codegen.json
type CodegenConfig struct {
	Overwrite bool               `json:"overwrite"`
	Schema    string             `json:"schema"`
	Generates CodegenSchemaEntry `json:"generates"`
}

func (schema CodegenConfig) JSON() string {
	b, err := json.Marshal(schema)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fmt.Sprint(string(b))
}

func (schema CodegenConfig) JSONByte() []byte {
	b, err := json.Marshal(schema)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return b
}

type CodegenSchemaEntry map[string]CodegenSchemaEntryConfig

type CodegenPresetConfig map[string]string

type CodegenSchemaEntryConfig struct {
	Preset       string              `json:"preset"`
	PresetConfig CodegenPresetConfig `json:"presetConfig"`
	Plugins      []string            `json:"plugins"`
	Documents    []string            `json:"documents"`
}
