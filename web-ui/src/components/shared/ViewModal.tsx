import { useState, useMemo } from 'react';
import { JsonView, allExpanded, defaultStyles } from 'react-json-view-lite';
import 'react-json-view-lite/dist/index.css';
import './json-view.css';

interface ViewModalProps {
  isOpen: boolean;
  onClose: () => void;
  keyName: string;
  value: string;
}

export default function ViewModal({ isOpen, onClose, keyName, value }: ViewModalProps) {
  const [copied, setCopied] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'tree' | 'raw'>('tree');

  if (!isOpen) return null;

  // Recursively parse nested JSON strings
  const deepParseJSON = (obj: any): any => {
    if (typeof obj === 'string') {
      // Try to parse string as JSON
      try {
        const parsed = JSON.parse(obj);
        // Recursively parse the result
        return deepParseJSON(parsed);
      } catch {
        return obj;
      }
    } else if (Array.isArray(obj)) {
      return obj.map(item => deepParseJSON(item));
    } else if (obj !== null && typeof obj === 'object') {
      const result: any = {};
      for (const [key, val] of Object.entries(obj)) {
        result[key] = deepParseJSON(val);
      }
      return result;
    }
    return obj;
  };

  // Try to parse as JSON
  let parsedJSON: any = null;
  let isJSON = false;
  try {
    const initialParse = JSON.parse(value);
    parsedJSON = deepParseJSON(initialParse);
    isJSON = true;
  } catch {
    // Not JSON, use as-is
  }

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Filter JSON based on search
  const filteredJSON = useMemo(() => {
    if (!isJSON || !searchQuery || !parsedJSON) return parsedJSON;

    const filterObject = (obj: any, query: string): any => {
      if (typeof obj !== 'object' || obj === null) {
        return String(obj).toLowerCase().includes(query.toLowerCase()) ? obj : null;
      }

      if (Array.isArray(obj)) {
        const filtered = obj.map(item => filterObject(item, query)).filter(item => item !== null);
        return filtered.length > 0 ? filtered : null;
      }

      const filtered: any = {};
      let hasMatch = false;

      for (const [key, val] of Object.entries(obj)) {
        // Check if key matches
        if (key.toLowerCase().includes(query.toLowerCase())) {
          filtered[key] = val;
          hasMatch = true;
        } else {
          // Check if value matches
          const filteredVal = filterObject(val, query);
          if (filteredVal !== null) {
            filtered[key] = filteredVal;
            hasMatch = true;
          }
        }
      }

      return hasMatch ? filtered : null;
    };

    return filterObject(parsedJSON, searchQuery) || parsedJSON;
  }, [parsedJSON, searchQuery, isJSON]);

  const formattedRawValue = isJSON ? JSON.stringify(parsedJSON, null, 2) : value;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black bg-opacity-50 transition-opacity"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="flex min-h-full items-center justify-center p-4">
        <div className="relative bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] flex flex-col">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b">
            <div className="flex items-center gap-4">
              <h3 className="text-lg font-semibold text-gray-900">
                View Entry
              </h3>
              {isJSON && (
                <div className="flex gap-2">
                  <button
                    onClick={() => setViewMode('tree')}
                    className={`px-3 py-1 text-xs font-medium rounded ${
                      viewMode === 'tree'
                        ? 'bg-blue-100 text-blue-700'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                    }`}
                  >
                    Tree View
                  </button>
                  <button
                    onClick={() => setViewMode('raw')}
                    className={`px-3 py-1 text-xs font-medium rounded ${
                      viewMode === 'raw'
                        ? 'bg-blue-100 text-blue-700'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                    }`}
                  >
                    Raw JSON
                  </button>
                </div>
              )}
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 transition-colors"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-4">
            {/* Key Section */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm font-medium text-gray-700">Key</label>
                <button
                  onClick={() => handleCopy(keyName)}
                  className="text-xs text-blue-600 hover:text-blue-700"
                >
                  {copied ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <div className="bg-gray-50 border border-gray-200 rounded-lg p-3">
                <code className="text-sm font-mono text-gray-900 break-all">
                  {keyName}
                </code>
              </div>
            </div>

            {/* Search (only for JSON) */}
            {isJSON && viewMode === 'tree' && (
              <div>
                <label className="text-sm font-medium text-gray-700 mb-2 block">
                  Search in JSON
                </label>
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search keys or values..."
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                {searchQuery && (
                  <p className="text-xs text-gray-500 mt-1">
                    Filtering JSON tree by: "{searchQuery}"
                  </p>
                )}
              </div>
            )}

            {/* Value Section */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm font-medium text-gray-700">
                  Value {isJSON && (
                    <span className="text-xs text-gray-500">
                      (JSON - nested strings auto-parsed)
                    </span>
                  )}
                </label>
                <button
                  onClick={() => handleCopy(value)}
                  className="text-xs text-blue-600 hover:text-blue-700"
                >
                  {copied ? 'Copied!' : 'Copy Original'}
                </button>
              </div>

              {isJSON && viewMode === 'tree' ? (
                <div>
                  <div className="bg-blue-50 border border-blue-200 rounded-lg px-3 py-2 mb-2">
                    <p className="text-xs text-blue-700">
                      Nested JSON strings have been automatically parsed and expanded for better readability.
                    </p>
                  </div>
                  <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 overflow-x-auto">
                    <JsonView
                      data={filteredJSON}
                      shouldExpandNode={allExpanded}
                      style={{
                        ...defaultStyles,
                        container: 'json-view-container',
                        basicChildStyle: 'json-view-item',
                        label: 'json-view-label',
                        nullValue: 'json-view-null',
                        undefinedValue: 'json-view-undefined',
                        numberValue: 'json-view-number',
                        stringValue: 'json-view-string',
                        booleanValue: 'json-view-boolean',
                        otherValue: 'json-view-other',
                        punctuation: 'json-view-punctuation',
                      }}
                    />
                  </div>
                </div>
              ) : (
                <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 overflow-x-auto">
                  <pre className="text-sm font-mono text-gray-900 whitespace-pre-wrap break-words">
                    {formattedRawValue}
                  </pre>
                </div>
              )}
            </div>

            {/* Metadata */}
            <div className="grid grid-cols-2 gap-4 pt-4 border-t">
              <div>
                <label className="text-xs font-medium text-gray-500">Key Length</label>
                <p className="text-sm text-gray-900 mt-1">{keyName.length} characters</p>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500">Value Length</label>
                <p className="text-sm text-gray-900 mt-1">{value.length} characters</p>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500">Value Type</label>
                <p className="text-sm text-gray-900 mt-1">{isJSON ? 'JSON Object' : 'String'}</p>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500">Size</label>
                <p className="text-sm text-gray-900 mt-1">
                  {new Blob([value]).size} bytes
                </p>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="flex justify-end gap-3 p-6 border-t">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
