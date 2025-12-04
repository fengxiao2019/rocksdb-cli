import { useState } from 'react';
import { ticksAPI } from '@/api/ticks';

interface TimeTicksConverterProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function TimeTicksConverter({ isOpen, onClose }: TimeTicksConverterProps) {
  const [dateTimeInput, setDateTimeInput] = useState('');
  const [ticksInput, setTicksInput] = useState('');
  const [dateTimeResult, setDateTimeResult] = useState('');
  const [ticksResult, setTicksResult] = useState('');
  const [error, setError] = useState('');
  const [converting, setConverting] = useState(false);

  // Convert datetime string to .NET ticks using backend API
  const convertToTicks = async () => {
    setError('');
    setTicksResult('');
    setConverting(true);
    
    try {
      if (!dateTimeInput.trim()) {
        setError('Please enter a date/time');
        return;
      }

      const result = await ticksAPI.convertDateTimeToTicks(dateTimeInput);
      setTicksResult(result.ticks); // Already a string from API
      setDateTimeResult(result.datetime + ' (' + new Date(result.datetime).toLocaleString() + ')');
    } catch (err: any) {
      const errorMsg = err.response?.data?.message || err.message || 'Conversion failed';
      setError(errorMsg);
    } finally {
      setConverting(false);
    }
  };

  // Convert .NET ticks to datetime using backend API
  const convertFromTicks = async () => {
    setError('');
    setDateTimeResult('');
    setConverting(true);
    
    try {
      if (!ticksInput.trim()) {
        setError('Please enter ticks value');
        return;
      }

      // Validate that it's a valid integer string
      if (!/^\d+$/.test(ticksInput.trim())) {
        setError('Invalid ticks value. Must be a valid integer');
        return;
      }

      const result = await ticksAPI.convertTicksToDateTime(ticksInput.trim());
      setDateTimeResult(result.datetime + ' (' + new Date(result.datetime).toLocaleString() + ')');
      setTicksResult(result.ticks); // Already a string from API
    } catch (err: any) {
      const errorMsg = err.response?.data?.message || err.message || 'Conversion failed';
      setError(errorMsg);
    } finally {
      setConverting(false);
    }
  };

  const setCurrentTime = () => {
    const now = new Date();
    setDateTimeInput(now.toISOString().slice(0, 19));
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-gray-900 bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg shadow-xl p-6 max-w-2xl w-full">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-gray-900">.NET Ticks Converter</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
            {error}
          </div>
        )}

        {/* DateTime to Ticks */}
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <h3 className="text-lg font-semibold text-gray-800 mb-3">DateTime → .NET Ticks</h3>
          
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Date/Time Input
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={dateTimeInput}
                  onChange={(e) => setDateTimeInput(e.target.value)}
                  placeholder="2024-12-04 15:30:00 or ISO format"
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  onKeyPress={(e) => e.key === 'Enter' && convertToTicks()}
                />
                <button
                  onClick={setCurrentTime}
                  className="px-3 py-2 bg-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-300 transition-colors"
                  title="Set current time"
                >
                  Now
                </button>
              </div>
              <p className="mt-1 text-xs text-gray-500">
                Examples: 2024-12-04, 2024-12-04 15:30:00, 2024-12-04T15:30:00Z
              </p>
            </div>

            <button
              onClick={convertToTicks}
              disabled={converting}
              className="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {converting ? 'Converting...' : 'Convert to Ticks'}
            </button>

            {ticksResult && (
              <div className="p-3 bg-white border border-gray-300 rounded-lg">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-medium text-gray-600">Result (Ticks):</span>
                  <button
                    onClick={() => copyToClipboard(ticksResult)}
                    className="text-xs text-blue-600 hover:text-blue-800"
                    title="Copy to clipboard"
                  >
                    Copy
                  </button>
                </div>
                <div className="font-mono text-sm text-gray-900 break-all">{ticksResult}</div>
                {dateTimeResult && (
                  <div className="mt-2 text-xs text-gray-500">{dateTimeResult}</div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Ticks to DateTime */}
        <div className="p-4 bg-gray-50 rounded-lg">
          <h3 className="text-lg font-semibold text-gray-800 mb-3">.NET Ticks → DateTime</h3>
          
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Ticks Input
              </label>
              <input
                type="text"
                value={ticksInput}
                onChange={(e) => setTicksInput(e.target.value)}
                placeholder="638389728000000000"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                onKeyPress={(e) => e.key === 'Enter' && convertFromTicks()}
              />
              <p className="mt-1 text-xs text-gray-500">
                Example: 638389728000000000 (100-nanosecond intervals since 0001-01-01 00:00:00 UTC)
              </p>
            </div>

            <button
              onClick={convertFromTicks}
              disabled={converting}
              className="w-full px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {converting ? 'Converting...' : 'Convert to DateTime'}
            </button>

            {dateTimeResult && (
              <div className="p-3 bg-white border border-gray-300 rounded-lg">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-medium text-gray-600">Result (DateTime):</span>
                  <button
                    onClick={() => copyToClipboard(dateTimeResult.split(' (')[0])}
                    className="text-xs text-blue-600 hover:text-blue-800"
                    title="Copy ISO format"
                  >
                    Copy
                  </button>
                </div>
                <div className="text-sm text-gray-900">{dateTimeResult}</div>
              </div>
            )}
          </div>
        </div>

        <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <p className="text-xs text-blue-800">
            <strong>Info:</strong> .NET ticks are 100-nanosecond intervals since January 1, 0001 00:00:00 UTC.
            1 tick = 100 nanoseconds, 10,000 ticks = 1 millisecond.
          </p>
          <p className="text-xs text-blue-700 mt-2">
            <strong>Backend API:</strong> This converter uses the Go backend API which supports full nanosecond precision,
            preserving all .NET ticks precision including sub-millisecond values.
          </p>
        </div>
      </div>
    </div>
  );
}

