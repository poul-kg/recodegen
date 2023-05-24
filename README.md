# Recodegen
Faster GraphQL codegen for TypeScript projects

## Installation
### Binary
* you can download binaris from [Releases](https://github.com/poul-kg/recodegen/releases) page
### Build from Source
* `git clone git@github.com:poul-kg/recodegen.git`
* `cd recodegen`
* `cd cmd/recodegen`
* `go build` - you'll see a binary file created
* `go install` - to install `recodegen` and be able to run it from any directory. For this to work make sure your `PATH` contains output from `go env GOPATH` command

## Usage
* `./recodegen` - reads `recodegen.json` and tries to generate types
* `./recodegen -config=codegen.json` - can specify custom JSON file

## Configuration
* the idea was to re-use existing Apollo `@graphql-codegen/cli` JSON config format
* `plugins: ['typescript']` - will generate schema types
* `plugins: ['typescript-operations']` - will generate operations
* you can't use multiple plugins like `plugins: ['typescript', 'typescript-operations']` - only first value is supported by now
* you can't use `documents: ` like `documents: "dir/*.ts"` it should be `documents: ["dir/*.ts"]`
* basically two config examples you see below are the only supported options for now, everything else will be ignored or will throw an error

`recodegen.json`
```JSON
{
  "overwrite": true,
  "schema": "schema/backend.graphql",
  "generates": {
    "generated/schema.generated.ts": {
      "plugins": [
        "typescript"
      ]
    },
    "generated/operations.ts": {
      "preset": "import-types",
      "presetConfig": {
        "typesPath": "./schema.generated"
      },
      "plugins": [
        "typescript-operations"
      ],
      "documents": ["backend/**/*.ts"]
    }
  }
}
```

The following config will do the following:
* read GraphQL schema from `schema/backend.graphql` file
* generate `generated/schema.generated.ts` file with all schema types
* scan all `*.ts` files in `./backend` directory and try to extract GraphQL queries, which should be defined as

```TypeScript
 const query = gql`
   query findUser($id: uuid!) {
     first_name
     last_name
   }
 `;
```
* Writes operations into `generated/operations.ts`
* `operations.ts` will import schema types like `import * as Types from "./schema.generated";`
### If you want both schema and operations in the same file here is a config which will work for you

`recodegen.json`
```JSON
{
  "overwrite": true,
  "schema": "schema/backend.graphql",
  "generates": {
    "generated/schema.generated.ts": {
      "plugins": [
        "typescript"
      ]
    },
    "generated/operations.ts": {
      "plugins": [
        "typescript-operations"
      ],
      "documents": ["backend/**/*.ts"]
    }
  }
}
```
* `./recodegen` - to build schema file and operations file
* `cat generated/schema.generated.ts generated.operations.ts > generated/both.ts` - to combine two files

## Known Issues
* output is not property formatted
* GraphQL fragment support is limited but something is supported
* it's pretty raw right now, was done pretty quickly to avoid constant disappointment with slow codegen