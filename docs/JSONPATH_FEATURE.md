# JSONPath Query Feature

## Overview

This document describes the JSONPath query feature added to rocksdb-cli, which allows querying JSON values stored in RocksDB using JSONPath expressions.

## What is JSONPath?

JSONPath is a query language for JSON, similar to XPath for XML. It allows you to extract specific data from JSON documents using path expressions.

## Implementation

### New Dependencies

- Added `github.com/oliveagle/jsonpath` library for JSONPath support

### New Functions in `internal/jsonutil`

1. **`QueryJSONPath(jsonData string, jsonPathExpr string) (string, error)`**
   - Queries a JSON string using a JSONPath expression
   - Returns the result as a JSON string
   - Handles errors for invalid JSON or JSONPath expressions

2. **`IsValidJSON(s string) bool`**
   - Validates if a string is valid JSON
   - Used to pre-validate data before JSONPath queries

### New Command: `jpath` (alias: `jsonpath`)

The `jpath` command allows you to query JSON values using JSONPath expressions.

**Syntax:**
```
jpath [<cf>] <key> <jsonpath> [--pretty] [--smart=true|false]
```

**Parameters:**
- `<cf>` - Column family (optional, uses current CF if not specified)
- `<key>` - The key containing JSON data
- `<jsonpath>` - JSONPath expression to query
- `--pretty` - Format output as pretty JSON (optional)
- `--smart` - Enable smart key conversion (default: true)

## Usage Examples

### Simple Field Access
```bash
# Get a simple field from JSON
rocksdb[default]> jpath user:123 "$.name"
"Alice"

# With explicit column family
rocksdb[default]> jpath users user:123 "$.name"
"Alice"
```

### Nested Field Access
```bash
# Access nested fields
rocksdb[default]> jpath user:123 "$.profile.email"
"alice@example.com"

# Multiple levels deep
rocksdb[default]> jpath user:123 "$.address.city.name"
"New York"
```

### Array Access
```bash
# Get first item in array
rocksdb[default]> jpath order:456 "$.items[0].product"
"laptop"

# Get specific index
rocksdb[default]> jpath data:789 "$.numbers[2]"
3

# Get all array items
rocksdb[default]> jpath products:001 "$.tags[*]"
["electronics","computers","laptops"]
```

### Complex Queries
```bash
# Get entire JSON document
rocksdb[default]> jpath user:123 "$"
{"name":"Alice","age":30,"profile":{"email":"alice@example.com"}}

# With pretty formatting
rocksdb[default]> jpath user:123 "$" --pretty
{
  "name": "Alice",
  "age": 30,
  "profile": {
    "email": "alice@example.com"
  }
}
```

## Common JSONPath Expressions

| Expression | Description | Example |
|------------|-------------|---------|
| `$` | Root element | Get entire JSON |
| `$.field` | Direct field access | `$.name` |
| `$.field1.field2` | Nested field access | `$.user.email` |
| `$.items[0]` | Array index (0-based) | First item |
| `$.items[*]` | All array items | All items |
| `$.items[-1]` | Last array item | Last item |

## Error Handling

The command handles various error scenarios:

1. **Key not found**: Shows standard "key not found" error
2. **Invalid JSON**: "Error: Value for key 'xxx' is not valid JSON"
3. **Invalid JSONPath**: "JSONPath query error: [error details]"
4. **Column family not found**: Standard CF error handling

## Integration with Help System

The new command is integrated into the CLI help system:

```bash
rocksdb[default]> help
Available commands:
  ...
  jpath [<cf>] <key> <jsonpath> [--pretty] - Query JSON value using JSONPath
  ...
```

Additional help information is shown in the notes section:
```
- jpath queries JSON values using JSONPath expressions ($.field, $.items[0], etc.)
  Example: jpath user:123 "$.profile.email" extracts email from user profile
```

## Testing

### Unit Tests

All JSONPath functionality is thoroughly tested in `internal/jsonutil/jsonutil_test.go`:

- `TestQueryJSONPath` - Tests various JSONPath queries
  - Simple field access
  - Nested field access
  - Array indexing
  - Wildcard selection
  - Invalid JSON handling
  - Invalid JSONPath expressions
  - Edge cases

- `TestIsValidJSON` - Tests JSON validation
  - Valid JSON objects, arrays, primitives
  - Invalid JSON
  - Edge cases

### Integration Tests

Command-level tests in `internal/command/command_test.go`:

- Simple field access
- Nested field access
- Array index access
- Invalid JSON value handling
- Key not found handling
- Explicit column family usage
- Usage help display

### Test Results

All tests pass successfully:

```bash
$ go test -v ./internal/jsonutil/
=== RUN   TestQueryJSONPath
=== RUN   TestQueryJSONPath/simple_field_access
=== RUN   TestQueryJSONPath/nested_field_access
=== RUN   TestQueryJSONPath/array_index_access
...
--- PASS: TestQueryJSONPath (0.00s)
...
PASS
ok      rocksdb-cli/internal/jsonutil
```

## Benefits

1. **Precise Data Extraction**: Extract only the data you need from JSON values
2. **Complex Queries**: Navigate nested structures and arrays easily
3. **Pretty Printing**: Optional pretty formatting for better readability
4. **Error Handling**: Clear error messages for invalid inputs
5. **Integrated**: Works seamlessly with existing CLI features (smart keys, column families, etc.)

## Comparison with Existing Commands

| Command | Purpose | Use Case |
|---------|---------|----------|
| `get` | Get entire value | When you need the full JSON |
| `jpath` | Query part of JSON | When you need specific fields |
| `jsonquery` | Find entries by field value | Search across multiple keys |

## Example Workflow

```bash
# Start CLI
$ rocksdb-cli --db ./mydb

# Switch to users column family
rocksdb[default]> usecf users
Switched to column family: users

# Get full user record
rocksdb[users]> get user:12345 --pretty
{
  "id": 12345,
  "name": "Alice Smith",
  "email": "alice@example.com",
  "profile": {
    "age": 30,
    "city": "New York",
    "interests": ["coding", "reading", "hiking"]
  },
  "orders": [
    {"id": 1001, "total": 99.99},
    {"id": 1002, "total": 149.99}
  ]
}

# Extract just the email
rocksdb[users]> jpath user:12345 "$.email"
"alice@example.com"

# Get user's city
rocksdb[users]> jpath user:12345 "$.profile.city"
"New York"

# Get all interests
rocksdb[users]> jpath user:12345 "$.profile.interests[*]"
["coding","reading","hiking"]

# Get first order total
rocksdb[users]> jpath user:12345 "$.orders[0].total"
99.99
```

## Technical Notes

1. **JSONPath Library**: Uses `github.com/oliveagle/jsonpath` which supports standard JSONPath syntax
2. **Return Format**: Results are always returned as valid JSON strings
3. **Smart Keys**: Works with the smart key conversion feature for binary keys
4. **Performance**: Efficient for large JSON documents as only requested data is extracted
5. **Thread-Safety**: Functions are stateless and thread-safe

## Future Enhancements

Potential improvements for future versions:

1. **Filter Expressions**: Support for `$.items[?(@.price < 100)]` syntax
2. **Recursive Descent**: Support for `$..field` to search all levels
3. **Batch Queries**: Query multiple keys with the same JSONPath
4. **Output Formats**: Support for CSV, TSV output from array results
5. **JSONPath in Search**: Integrate JSONPath into the `search` command

## Conclusion

The JSONPath query feature significantly enhances the CLI's ability to work with JSON data stored in RocksDB. It provides a powerful, standards-based way to extract specific data from complex JSON structures without retrieving entire values.

