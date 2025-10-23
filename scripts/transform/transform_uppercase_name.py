#!/usr/bin/env python3
"""
Simple transformation: Uppercase the 'name' field in JSON values
Usage: rocksdb-cli transform users --script=examples/transform_uppercase_name.py
"""

import json

# Transform function - will be called for each key-value pair
def transform_value(key: str, value: str) -> str:
    """Uppercase the name field"""
    try:
        data = json.loads(value)
        if 'name' in data:
            data['name'] = data['name'].upper()
        return json.dumps(data)
    except:
        return value

# Optional filter function
def should_process(key: str, value: str) -> bool:
    """Only process entries that have a 'name' field"""
    try:
        data = json.loads(value)
        return 'name' in data
    except:
        return False
