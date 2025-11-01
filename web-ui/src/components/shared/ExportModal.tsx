import { useState } from 'react';

interface ExportModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: Record<string, string>;
  columnFamily: string;
}

export default function ExportModal({ isOpen, onClose, data, columnFamily }: ExportModalProps) {
  const [format, setFormat] = useState<'csv' | 'json'>('json');
  const [exporting, setExporting] = useState(false);

  if (!isOpen) return null;

  const handleExport = async () => {
    setExporting(true);

    try {
      let content: string;
      let filename: string;
      let mimeType: string;

      if (format === 'csv') {
        // CSV export
        const rows = [['Key', 'Value']];
        Object.entries(data).forEach(([key, value]) => {
          // Escape double quotes and wrap in quotes if contains comma
          const escapeCSV = (str: string) => {
            if (str.includes(',') || str.includes('"') || str.includes('\n')) {
              return `"${str.replace(/"/g, '""')}"`;
            }
            return str;
          };
          rows.push([escapeCSV(key), escapeCSV(value)]);
        });
        content = rows.map(row => row.join(',')).join('\n');
        filename = `${columnFamily}_export_${new Date().toISOString().split('T')[0]}.csv`;
        mimeType = 'text/csv;charset=utf-8;';
      } else {
        // JSON export
        const jsonData = Object.entries(data).map(([key, value]) => {
          // Try to parse value as JSON
          let parsedValue = value;
          try {
            parsedValue = JSON.parse(value);
          } catch {
            // Keep as string if not JSON
          }
          return { key, value: parsedValue };
        });
        content = JSON.stringify(jsonData, null, 2);
        filename = `${columnFamily}_export_${new Date().toISOString().split('T')[0]}.json`;
        mimeType = 'application/json;charset=utf-8;';
      }

      // Create download
      const blob = new Blob([content], { type: mimeType });
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = filename;
      link.click();
      URL.revokeObjectURL(link.href);

      // Close modal after short delay
      setTimeout(() => {
        onClose();
      }, 500);
    } catch (error) {
      console.error('Export error:', error);
    } finally {
      setExporting(false);
    }
  };

  const entryCount = Object.keys(data).length;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black bg-opacity-50 transition-opacity"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="flex min-h-full items-center justify-center p-4">
        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b">
            <h3 className="text-lg font-semibold text-gray-900">
              Export Data
            </h3>
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
          <div className="p-6 space-y-4">
            <div>
              <p className="text-sm text-gray-600">
                Export <span className="font-semibold">{entryCount}</span> entries from{' '}
                <span className="font-semibold">{columnFamily}</span>
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Export Format
              </label>
              <div className="space-y-2">
                <label className="flex items-center p-3 border rounded-lg cursor-pointer hover:bg-gray-50">
                  <input
                    type="radio"
                    value="json"
                    checked={format === 'json'}
                    onChange={() => setFormat('json')}
                    className="mr-3"
                  />
                  <div>
                    <div className="font-medium text-gray-900">JSON</div>
                    <div className="text-xs text-gray-500">
                      Structured JSON array with key-value pairs
                    </div>
                  </div>
                </label>
                <label className="flex items-center p-3 border rounded-lg cursor-pointer hover:bg-gray-50">
                  <input
                    type="radio"
                    value="csv"
                    checked={format === 'csv'}
                    onChange={() => setFormat('csv')}
                    className="mr-3"
                  />
                  <div>
                    <div className="font-medium text-gray-900">CSV</div>
                    <div className="text-xs text-gray-500">
                      Comma-separated values (Key, Value)
                    </div>
                  </div>
                </label>
              </div>
            </div>

            {/* Preview */}
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-3">
              <p className="text-xs font-medium text-gray-700 mb-2">
                Preview ({format.toUpperCase()})
              </p>
              <pre className="text-xs text-gray-600 font-mono overflow-x-auto">
                {format === 'json'
                  ? `[\n  {\n    "key": "...",\n    "value": {...}\n  },\n  ...\n]`
                  : `Key,Value\n"key1","value1"\n"key2","value2"\n...`}
              </pre>
            </div>
          </div>

          {/* Footer */}
          <div className="flex justify-end gap-3 p-6 border-t">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleExport}
              disabled={exporting}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {exporting ? 'Exporting...' : 'Export'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
