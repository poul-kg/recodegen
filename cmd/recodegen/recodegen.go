package main

import (
	"flag"
	"fmt"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"log"
	"os"
	"path/filepath"
	"recodegen/config"
	"recodegen/typescript"
	"runtime"
)

const VERSION = "v0.1.3"

func main() {
	configFileName := flag.String("config", "recodegen.json", "Configuration file name")
	versionFlag := flag.Bool("v", false, "Print version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if *configFileName == "" {
		*configFileName = "recodegen.json"
	}
	cliConfig := config.ReadConfigFromFile(*configFileName)
	schemaAst := getSchemaAst(cliConfig.Schema)

	for outputFileName, genConfig := range cliConfig.Generates {
		processInput(schemaAst, outputFileName, genConfig)
	}

	//fmt.Println("Parallel")
	//var wg sync.WaitGroup
	//
	//for outputFileName, genConfig := range cliConfig.Generates {
	//	// increment the WaitGroup counter
	//	wg.Add(1)
	//
	//	go func(schemaAst *ast.Schema, outputFileName string, genConfig config.CodegenSchemaEntryConfig) {
	//		// decrement the WaitGroup counter when the goroutine completes
	//		defer wg.Done()
	//
	//		processInput(schemaAst, outputFileName, genConfig)
	//	}(schemaAst, outputFileName, genConfig)
	//}
	//
	//// wait for all goroutines to finish
	//wg.Wait()

	PrintMemUsage()
}

func processInput(schemaAst *ast.Schema, outputFileName string, genConfig config.CodegenSchemaEntryConfig) {
	output := ""
	for _, plugin := range genConfig.Plugins {
		if plugin == "typescript" {
			//generateSchema(schemaAst, outputFileName)
			schema := typescript.Schema{Ast: schemaAst}
			output += schema.String()
		}

		if plugin == "typescript-operations" {
			operation := typescript.Operations{
				Ast:    schemaAst,
				Config: genConfig,
			}

			output += operation.String()
		}
	}

	existingFileContent := getFileContentIfExists(outputFileName)
	if *existingFileContent != output {
		fmt.Printf("[writing] %s\n", outputFileName)
		writeFile(outputFileName, output)
	} else {
		fmt.Printf("[skipping] %s\n", outputFileName)
	}
}

func getFileContentIfExists(fileName string) *string {
	schemaBytes, err := os.ReadFile(fileName)
	if err != nil {
		empty := ""
		return &empty
	}

	content := string(schemaBytes)
	return &content
}

func getFileContent(fileName string) string {
	schemaBytes, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	return string(schemaBytes)
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

func writeFile(fileName string, data string) {
	dir := filepath.Dir(fileName)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

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

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
