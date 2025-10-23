#!/usr/bin/env python3
"""
Filter and transform: Only keep users older than 30, add age_group field
Usage: rocksdb-cli transform users --script=examples/filter_by_age.py --dry-run
"""

import json

def should_process(key: str, value: str) -> bool:
    """Filter: Only process users with age > 30"""
    try:
        data = json.loads(value)
        return 'age' in data and data['age'] > 30
    except:
        return False

def transform_value(key: str, value: str) -> str:
    """Add age_group field based on age"""
    try:
        data = json.loads(value)
        age = data.get('age', 0)
        
        if age < 25:
            data['age_group'] = 'young'
        elif age < 35:
            data['age_group'] = 'middle'
        else:
            data['age_group'] = 'senior'
        
        return json.dumps(data)
    except:
        return value
