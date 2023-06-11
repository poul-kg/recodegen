# Recodegen
Faster GraphQL codegen for TypeScript projects

## Installation
### Binary
* you can download binaris from [Releases](https://github.com/poul-kg/recodegen/releases) page
### NPM
#### Globally via NPM
* `npm install -g @graphql-recodegen/cli`
* to run `recodegen -config=recodegen.json`
#### Locally via NPM
* `npm install --save-dev @graphql-recodegen/cli`
* to run `./node_modules/.bin/recodegen -config=recodegen.json`
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
* only two plugins supported: `plugins: ['typescript', 'typescript-operations']`
* you can't use `documents: ` like `documents: "dir/*.ts"` it should be `documents: ["dir/*.ts"]`
* basically two config examples you see below are the only supported options for now, everything else will be ignored or will throw an error

### Config 1 - separate files for generated schema and operations
The following config will:
* read GraphQL schema from `schema/backend.graphql` file
* generate `generated/schema.generated.ts` file with all schema types
* scan all `*.ts` files in `./backend` directory and try to extract GraphQL queries, which should be defined as

```TypeScript
 // backend/my-service.ts
 const query = gql`
   query findUser($id: uuid!) {
     first_name
     last_name
   }
 `;
```
* Writes operations into `generated/operations.ts`
* `operations.ts` will import schema types like `import * as Types from "./schema.generated";`

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
### Config 2 - single file for generated schema and operations
`recodegen.json`
```JSON
{
  "overwrite": true,
  "schema": "schema/backend.graphql",
  "generates": {
    "generated/schema.generated.ts": {
      "plugins": [
        "typescript", "typescript-operations"
      ],
      "documents": ["backend/**/*.ts"]
    }
  }
}
```
* `./recodegen` - to build schema file and operations file

## Known Issues
* [ ] some output is not properly formatted
* [x] types in the generated output may change order on every new generation
* [x] generated files are overwritten even if there is no change
* [ ] No Interface support
* [ ] No Union support
* [ ] no unit tests
* [ ] GraphQL fragment support is limited but something is supported
* [ ] some type names might differ a bit from what is generated by `@graphql-codegen/cli`
* it's pretty raw right now, was done pretty quickly to avoid constant disappointment with slow codegen