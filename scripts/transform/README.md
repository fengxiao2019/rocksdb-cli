# Transform Scripts

Production-ready Python scripts for the `transform` command.

## Available Scripts

### 1. transform_uppercase_name.py
**Purpose:** Uppercase the 'name' field in JSON values

**Usage:**
```bash
rocksdb-cli transform --db mydb --cf users \
  --script=scripts/transform/transform_uppercase_name.py --dry-run
```

**What it does:**
- Filters: Only processes entries that have a 'name' field
- Transform: Converts the 'name' field to uppercase
- Example: `{"name":"alice"}` → `{"name":"ALICE"}`

### 2. filter_by_age.py
**Purpose:** Filter and tag users by age group

**Usage:**
```bash
rocksdb-cli transform --db mydb --cf users \
  --script=scripts/transform/filter_by_age.py --dry-run
```

**What it does:**
- Filters: Only processes entries where age > 30
- Transform: Adds 'age_group' field based on age ranges
  - age > 50: "senior"
  - age > 40: "middle"
  - age > 30: "adult"

### 3. flatten_nested_json.py
**Purpose:** Flatten nested JSON strings within JSON values

**Usage:**
```bash
rocksdb-cli transform --db mydb --cf users \
  --script=scripts/transform/flatten_nested_json.py --dry-run
```

**What it does:**
- Filters: Only processes entries with nested JSON strings
- Transform: Recursively parses and flattens JSON strings
- Example: 
  ```json
  {"profile": "{\"name\":\"Alice\"}"}
  →
  {"profile": {"name": "Alice"}}
  ```

### 4. add_timestamp.py
**Purpose:** Add processing timestamp to entries

**Usage:**
```bash
rocksdb-cli transform --db mydb --cf logs \
  --script=scripts/transform/add_timestamp.py --dry-run
```

**What it does:**
- Filters: Processes all entries
- Transform: Adds 'processed_at' field with current ISO timestamp
- Example: `{"id":1}` → `{"id":1,"processed_at":"2024-01-23T10:30:00Z"}`

## Script Structure

All transform scripts should follow this structure:

```python
#!/usr/bin/env python3
"""
Script description
Usage: rocksdb-cli transform --db mydb --cf users --script=this_script.py
"""

import json

def should_process(key: str, value: str) -> bool:
    """
    Filter function - return True to process, False to skip
    
    Args:
        key: The entry's key (string)
        value: The entry's value (string)
    
    Returns:
        bool: True to process this entry, False to skip
    """
    # Your filter logic here
    return True

def transform_value(key: str, value: str) -> str:
    """
    Transform function - modify the value
    
    Args:
        key: The entry's key (string)
        value: The entry's value (string, often JSON)
    
    Returns:
        str: The transformed value
    """
    # Your transformation logic here
    return value
```

## Testing Scripts

Before applying transformations to your database:

1. **Always use dry-run mode first:**
   ```bash
   rocksdb-cli transform --db mydb --cf users \
     --script=examples/your_script.py --dry-run --limit=10
   ```

2. **Test on a small dataset:**
   ```bash
   rocksdb-cli transform --db mydb --cf users \
     --script=examples/your_script.py --dry-run --limit=5
   ```

3. **Check the preview output carefully**

4. **Apply to production:**
   ```bash
   rocksdb-cli transform --db mydb --cf users \
     --script=examples/your_script.py --limit=1000
   ```

## Creating Custom Scripts

### Example: Add Field

```python
import json

def transform_value(key, value):
    data = json.loads(value)
    data['new_field'] = 'new_value'
    return json.dumps(data)
```

### Example: Filter by Key Pattern

```python
def should_process(key, value):
    return key.startswith('user:')
```

### Example: Complex Transformation

```python
import json
from datetime import datetime

def should_process(key, value):
    try:
        data = json.loads(value)
        return 'status' in data and data['status'] == 'active'
    except:
        return False

def transform_value(key, value):
    data = json.loads(value)
    data['updated_at'] = datetime.utcnow().isoformat()
    data['status'] = 'processed'
    return json.dumps(data)
```

## Best Practices

1. **Error Handling:** Always use try-except for JSON parsing
2. **Type Checking:** Verify field types before operations
3. **Preserve Data:** Don't remove fields unless necessary
4. **Test Thoroughly:** Use dry-run with small limits
5. **Document:** Add clear comments explaining the logic
6. **Idempotency:** Scripts should be safe to run multiple times

## Requirements

- Python 3.6+
- Standard library only (json, datetime, etc.)
- No external dependencies by default

## Safety Tips

⚠️  **Always backup your database before transformations**  
💡 **Start with --limit=1 to test on a single entry**  
📊 **Check statistics output before proceeding**  
🔍 **Use --dry-run to preview changes**  
✅ **Verify transformed data in preview**  

## Troubleshooting

### Script errors
- Check Python syntax with: `python3 -m py_compile your_script.py`
- Test functions independently before using with transform

### JSON parsing errors
- Verify value is valid JSON
- Add try-except blocks for non-JSON values

### Unexpected results
- Add verbose output: `--verbose` flag
- Test with smaller dataset first
- Check filter logic in should_process()

## Contributing

Feel free to add more example scripts following the structure above!
