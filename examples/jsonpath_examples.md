# JSONPath Query Examples

This document demonstrates the JSONPath query feature in rocksdb-cli.

## Prerequisites

You need a RocksDB database with JSON values. You can create test data using:

```bash
# Start the CLI
rocksdb-cli --db ./testdb

# Switch to a column family or use default
rocksdb[default]> usecf users

# Insert some JSON test data
rocksdb[users]> put user:001 {"id":1,"name":"Alice Smith","email":"alice@example.com","profile":{"age":30,"city":"New York","country":"USA","interests":["coding","reading","hiking"]},"orders":[{"id":1001,"date":"2024-01-15","total":99.99,"items":[{"product":"laptop","price":899.99},{"product":"mouse","price":29.99}]},{"id":1002,"date":"2024-02-20","total":149.99,"items":[{"product":"keyboard","price":149.99}]}]}

rocksdb[users]> put user:002 {"id":2,"name":"Bob Johnson","email":"bob@example.com","profile":{"age":25,"city":"San Francisco","country":"USA","interests":["gaming","music"]},"orders":[]}

rocksdb[users]> put user:003 {"id":3,"name":"Charlie Brown","email":"charlie@example.com","profile":{"age":35,"city":"Boston","country":"USA","interests":["sports","travel","photography"]},"orders":[{"id":2001,"date":"2024-03-01","total":299.99,"items":[{"product":"camera","price":299.99}]}]}
```

## Basic Examples

### 1. Get a Simple Field

```bash
# Get user's name
rocksdb[users]> jpath user:001 "$.name"
"Alice Smith"

# Get user's email
rocksdb[users]> jpath user:001 "$.email"
"alice@example.com"

# Get user's ID
rocksdb[users]> jpath user:001 "$.id"
1
```

### 2. Access Nested Fields

```bash
# Get user's age from profile
rocksdb[users]> jpath user:001 "$.profile.age"
30

# Get user's city
rocksdb[users]> jpath user:001 "$.profile.city"
"New York"

# Get user's country
rocksdb[users]> jpath user:001 "$.profile.country"
"USA"
```

### 3. Array Access

```bash
# Get first interest
rocksdb[users]> jpath user:001 "$.profile.interests[0]"
"coding"

# Get second interest
rocksdb[users]> jpath user:001 "$.profile.interests[1]"
"reading"

# Get all interests
rocksdb[users]> jpath user:001 "$.profile.interests[*]"
["coding","reading","hiking"]
```

### 4. Nested Arrays

```bash
# Get first order
rocksdb[users]> jpath user:001 "$.orders[0]"
{"id":1001,"date":"2024-01-15","total":99.99,"items":[...]}

# Get first order's ID
rocksdb[users]> jpath user:001 "$.orders[0].id"
1001

# Get first order's total
rocksdb[users]> jpath user:001 "$.orders[0].total"
99.99

# Get first item of first order
rocksdb[users]> jpath user:001 "$.orders[0].items[0]"
{"product":"laptop","price":899.99}

# Get product name of first item in first order
rocksdb[users]> jpath user:001 "$.orders[0].items[0].product"
"laptop"

# Get all order IDs
rocksdb[users]> jpath user:001 "$.orders[*].id"
[1001,1002]
```

## Advanced Examples

### 5. Pretty Printing

```bash
# Get entire profile with pretty formatting
rocksdb[users]> jpath user:001 "$.profile" --pretty
{
  "age": 30,
  "city": "New York",
  "country": "USA",
  "interests": [
    "coding",
    "reading",
    "hiking"
  ]
}

# Get first order with pretty formatting
rocksdb[users]> jpath user:001 "$.orders[0]" --pretty
{
  "id": 1001,
  "date": "2024-01-15",
  "total": 99.99,
  "items": [
    {
      "product": "laptop",
      "price": 899.99
    },
    {
      "product": "mouse",
      "price": 29.99
    }
  ]
}
```

### 6. Root Access

```bash
# Get entire JSON document
rocksdb[users]> jpath user:002 "$"
{"id":2,"name":"Bob Johnson","email":"bob@example.com","profile":{"age":25,"city":"San Francisco","country":"USA","interests":["gaming","music"]},"orders":[]}

# Get entire JSON with pretty formatting
rocksdb[users]> jpath user:002 "$" --pretty
{
  "id": 2,
  "name": "Bob Johnson",
  "email": "bob@example.com",
  "profile": {
    "age": 25,
    "city": "San Francisco",
    "country": "USA",
    "interests": [
      "gaming",
      "music"
    ]
  },
  "orders": []
}
```

### 7. Working with Different Column Families

```bash
# Query with explicit column family
rocksdb[default]> jpath users user:001 "$.name"
"Alice Smith"

# Switch to column family first
rocksdb[default]> usecf users
Switched to column family: users

# Then query without specifying CF
rocksdb[users]> jpath user:001 "$.name"
"Alice Smith"
```

## Real-World Use Cases

### E-commerce Order Analysis

```bash
# Get customer email for support
rocksdb[users]> jpath user:001 "$.email"

# Get all order totals for accounting
rocksdb[users]> jpath user:001 "$.orders[*].total"

# Get specific order details
rocksdb[users]> jpath user:001 "$.orders[0].date"
rocksdb[users]> jpath user:001 "$.orders[0].items[*].product"
```

### User Profile Management

```bash
# Get demographic information
rocksdb[users]> jpath user:001 "$.profile.age"
rocksdb[users]> jpath user:001 "$.profile.city"

# Get user preferences
rocksdb[users]> jpath user:001 "$.profile.interests[*]"
```

### Data Extraction for Analytics

```bash
# Extract specific fields for CSV export (manual process)
rocksdb[users]> jpath user:001 "$.id"
1
rocksdb[users]> jpath user:001 "$.profile.city"
"New York"
rocksdb[users]> jpath user:001 "$.profile.age"
30
```

## Error Handling Examples

### Invalid Key

```bash
rocksdb[users]> jpath user:999 "$.name"
Key 'user:999' not found in column family 'users'
```

### Invalid JSON

```bash
# If you have non-JSON data
rocksdb[users]> put badkey "not json"
OK
rocksdb[users]> jpath badkey "$.name"
Error: Value for key 'badkey' is not valid JSON
Value: not json
```

### Invalid JSONPath

```bash
rocksdb[users]> jpath user:001 "$[invalid"
JSONPath query error: [error details]
```

## Comparison: get vs jpath

### Using `get` command

```bash
# Returns entire JSON (can be very large)
rocksdb[users]> get user:001
{"id":1,"name":"Alice Smith","email":"alice@example.com","profile":{...},"orders":[...]}

# With pretty formatting (better but still entire document)
rocksdb[users]> get user:001 --pretty
{
  "id": 1,
  "name": "Alice Smith",
  "email": "alice@example.com",
  "profile": {
    ...entire profile...
  },
  "orders": [
    ...all orders...
  ]
}
```

### Using `jpath` command

```bash
# Returns only what you need
rocksdb[users]> jpath user:001 "$.email"
"alice@example.com"

# More efficient for large documents
rocksdb[users]> jpath user:001 "$.orders[0].id"
1001
```

## Performance Tips

1. **Be Specific**: Use precise paths to minimize data processing
   - Good: `$.orders[0].total`
   - Less efficient: `$` then manually parse

2. **Use Pretty Print Sparingly**: Only use `--pretty` when you need formatted output for viewing

3. **Array Wildcards**: Be careful with `[*]` on large arrays
   - Returns all items which could be large

4. **Nested Queries**: JSONPath is more efficient than multiple `get` calls
   - One `jpath` call vs multiple `get` and manual parsing

## Combining with Other Commands

### With `prefix` scan

```bash
# First find all user keys
rocksdb[users]> prefix user:

# Then query specific fields from each
rocksdb[users]> jpath user:001 "$.email"
rocksdb[users]> jpath user:002 "$.email"
rocksdb[users]> jpath user:003 "$.email"
```

### With `scan` command

```bash
# Scan a range of users
rocksdb[users]> scan user:000 user:999 --values=no

# Then query specific users
rocksdb[users]> jpath user:001 "$.profile.city"
```

### With `jsonquery` command

```bash
# Find all users in New York
rocksdb[users]> jsonquery profile.city "New York"

# Then extract specific fields from results
rocksdb[users]> jpath user:001 "$.email"
rocksdb[users]> jpath user:001 "$.profile.age"
```

## Summary

The `jpath` command is ideal for:
- ✅ Extracting specific fields from JSON
- ✅ Navigating nested structures
- ✅ Working with arrays
- ✅ Reducing network/display overhead with large documents
- ✅ Building data pipelines and scripts

Use `get` for:
- Viewing entire documents
- When you need multiple fields from the same level
- Initial data exploration

Use `jsonquery` for:
- Searching across multiple keys
- Finding entries by field value
- Discovery and filtering

