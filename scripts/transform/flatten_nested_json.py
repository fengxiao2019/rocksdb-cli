#!/usr/bin/env python3
"""
Flatten nested JSON strings into proper JSON objects
Useful for entries like user:nested1 and user:nested2 from the test data
Usage: rocksdb-cli transform users --script=examples/flatten_nested_json.py --dry-run
"""

import json

def transform_value(key: str, value: str) -> str:
    """
    Parse nested JSON strings and flatten them into the parent object
    Example: {"profile": "{\"name\":\"Alice\"}"} 
          -> {"profile": {"name": "Alice"}}
    """
    try:
        data = json.loads(value)
        
        # Recursively parse string fields that look like JSON
        for field, field_value in list(data.items()):
            if isinstance(field_value, str):
                # Check if it looks like JSON
                if (field_value.startswith('{') and field_value.endswith('}')) or \
                   (field_value.startswith('[') and field_value.endswith(']')):
                    try:
                        # Try to parse as JSON
                        parsed = json.loads(field_value)
                        data[field] = parsed
                    except:
                        # Not valid JSON, keep as string
                        pass
        
        return json.dumps(data)
    except:
        return value

def should_process(key: str, value: str) -> bool:
    """Only process entries with nested JSON strings"""
    try:
        data = json.loads(value)
        # Check if any field value is a JSON string
        for field_value in data.values():
            if isinstance(field_value, str):
                if field_value.startswith('{') or field_value.startswith('['):
                    return True
        return False
    except:
        return False
