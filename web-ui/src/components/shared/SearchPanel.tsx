import { useState } from 'react';
import type { SearchRequest } from '@/types/api';

interface SearchPanelProps {
  onSearch: (params: SearchRequest) => void;
  isLoading: boolean;
}

export default function SearchPanel({ onSearch, isLoading }: SearchPanelProps) {
  const [keyPattern, setKeyPattern] = useState('');
  const [valuePattern, setValuePattern] = useState('');
  const [useRegex, setUseRegex] = useState(false);
  const [caseSensitive, setCaseSensitive] = useState(false);

  const handleSearch = () => {
    if (!keyPattern && !valuePattern) {
      return;
    }

    onSearch({
      key_pattern: keyPattern || undefined,
      value_pattern: valuePattern || undefined,
      use_regex: useRegex,
      case_sensitive: caseSensitive,
      limit: 100,
    });
  };

  const handleClear = () => {
    setKeyPattern('');
    setValuePattern('');
    setUseRegex(false);
    setCaseSensitive(false);
  };

  return (
    <div className="bg-white border-b p-4 space-y-4">
      <div className="flex items-center gap-4">
        <h3 className="text-sm font-semibold text-gray-900">Search</h3>
        <div className="flex gap-4">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={useRegex}
              onChange={(e) => setUseRegex(e.target.checked)}
              className="rounded border-gray-300"
            />
            <span className="text-gray-700">Use Regex</span>
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={caseSensitive}
              onChange={(e) => setCaseSensitive(e.target.checked)}
              className="rounded border-gray-300"
            />
            <span className="text-gray-700">Case Sensitive</span>
          </label>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Key Pattern
          </label>
          <input
            type="text"
            value={keyPattern}
            onChange={(e) => setKeyPattern(e.target.value)}
            placeholder="Search by key..."
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Value Pattern
          </label>
          <input
            type="text"
            value={valuePattern}
            onChange={(e) => setValuePattern(e.target.value)}
            placeholder="Search by value..."
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
        </div>
      </div>

      <div className="flex gap-2">
        <button
          onClick={handleSearch}
          disabled={isLoading || (!keyPattern && !valuePattern)}
          className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isLoading ? 'Searching...' : 'Search'}
        </button>
        <button
          onClick={handleClear}
          className="px-4 py-2 bg-white border border-gray-300 text-gray-700 text-sm font-medium rounded-lg hover:bg-gray-50 transition-colors"
        >
          Clear
        </button>
      </div>
    </div>
  );
}
