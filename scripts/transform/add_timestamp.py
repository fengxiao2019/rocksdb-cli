#!/usr/bin/env python3
"""
Add timestamp to all entries
Usage: rocksdb-cli transform users --script=examples/add_timestamp.py
"""

import json
from datetime import datetime

def transform_value(key: str, value: str) -> str:
    """Add processed_at timestamp to each entry"""
    try:
        data = json.loads(value)
        data['processed_at'] = datetime.utcnow().isoformat() + 'Z'
        data['transform_version'] = '1.0'
        return json.dumps(data)
    except:
        return value
