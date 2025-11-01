import { useEffect, useState } from 'react';
import { useDbStore } from '@/stores/dbStore';
import { listColumnFamilies, scanData, searchData } from '@/api/database';
import type { ScanResult, SearchRequest, SearchResponse } from '@/types/api';
import ViewModal from '@/components/shared/ViewModal';
import SearchPanel from '@/components/shared/SearchPanel';
import ExportModal from '@/components/shared/ExportModal';

export default function Dashboard() {
  const {currentCF, columnFamilies, setCurrentCF, setColumnFamilies} = useDbStore();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [searchResult, setSearchResult] = useState<SearchResponse | null>(null);
  const [loadingData, setLoadingData] = useState(false);
  const [viewModal, setViewModal] = useState<{ key: string; value: string } | null>(null);
  const [showSearch, setShowSearch] = useState(false);
  const [isSearchMode, setIsSearchMode] = useState(false);
  const [showExport, setShowExport] = useState(false);

  useEffect(() => {
    loadColumnFamilies();
  }, []);

  useEffect(() => {
    if (currentCF) {
      loadData();
    }
  }, [currentCF]);

  const loadColumnFamilies = async () => {
    try {
      setLoading(true);
      const data = await listColumnFamilies();
      setColumnFamilies(data.column_families);
      if (data.column_families.length > 0 && !currentCF) {
        setCurrentCF(data.column_families[0]);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const loadData = async (cursor?: string) => {
    if (!currentCF) return;

    try {
      setLoadingData(true);
      const result = await scanData(currentCF, {
        limit: 50,
        after: cursor
      });

      // If loading next page, append to existing results
      if (cursor && scanResult) {
        // Use results_v2 if available, fallback to old format
        const newResultsV2 = result.results_v2 || [];
        const existingResultsV2 = scanResult.results_v2 || [];

        setScanResult({
          ...result,
          results: { ...scanResult.results, ...result.results },
          results_v2: [...existingResultsV2, ...newResultsV2],
          count: scanResult.count + result.count
        });
      } else {
        setScanResult(result);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoadingData(false);
    }
  };

  const loadMore = () => {
    if (scanResult?.next_cursor) {
      loadData(scanResult.next_cursor);
    }
  };

  const handleSearch = async (params: SearchRequest) => {
    if (!currentCF) return;

    try {
      setLoadingData(true);
      setIsSearchMode(true);
      const result = await searchData(currentCF, params);
      setSearchResult(result);
      setScanResult(null); // Clear scan results
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoadingData(false);
    }
  };

  const handleBackToScan = () => {
    setIsSearchMode(false);
    setSearchResult(null);
    setShowSearch(false);
    loadData(); // Reload scan data
  };

  // Helper function to convert hex string to binary
  const hexToBytes = (hexStr: string): string => {
    const hex = hexStr.replace(/\s/g, ''); // Remove spaces
    const bytes = [];
    for (let i = 0; i < hex.length; i += 2) {
      bytes.push(String.fromCharCode(parseInt(hex.substr(i, 2), 16)));
    }
    return bytes.join('');
  };

  const getExportData = () => {
    if (isSearchMode && searchResult) {
      // Convert SearchResult[] to Record<string, string>
      // Decode hex values for export
      const data: Record<string, string> = {};
      searchResult.results.forEach(item => {
        const key = item.key_is_binary ? hexToBytes(item.key) : item.key;
        const value = item.value_is_binary ? hexToBytes(item.value) : item.value;
        data[key] = value;
      });
      return data;
    } else if (scanResult) {
      // Use results_v2 if available and decode hex
      if (scanResult.results_v2) {
        const data: Record<string, string> = {};
        scanResult.results_v2.forEach(item => {
          const key = item.key_is_binary ? hexToBytes(item.key) : item.key;
          const value = item.value_is_binary ? hexToBytes(item.value) : item.value;
          data[key] = value;
        });
        return data;
      }
      return scanResult.results;
    }
    return {};
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
          <p className="text-gray-600">Loading database...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 max-w-md">
          <h3 className="text-red-800 font-semibold mb-2">Error</h3>
          <p className="text-red-600">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="px-6 py-4">
          <h1 className="text-2xl font-bold text-gray-900">
            RocksDB Web UI
          </h1>
          <p className="text-sm text-gray-600">
            Database Management Interface
          </p>
        </div>
      </header>

      <div className="flex h-[calc(100vh-73px)]">
        {/* Sidebar - Column Families */}
        <aside className="w-64 bg-white border-r overflow-y-auto">
          <div className="p-4 border-b">
            <h2 className="text-sm font-semibold text-gray-700 mb-3">
              Column Families ({columnFamilies.length})
            </h2>
            <div className="space-y-1">
              {columnFamilies.map((cf) => (
                <button
                  key={cf}
                  onClick={() => setCurrentCF(cf)}
                  className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                    currentCF === cf
                      ? 'bg-blue-50 text-blue-700 font-medium'
                      : 'text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  {cf}
                </button>
              ))}
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-hidden flex flex-col">
          <div className="p-6 border-b bg-white">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold text-gray-900">
                  {currentCF || 'Select a Column Family'}
                </h2>
                <p className="text-sm text-gray-600 mt-1">
                  {isSearchMode && searchResult
                    ? `${searchResult.count} search results (${searchResult.query_time})`
                    : scanResult && `${scanResult.count} entries shown`}
                </p>
              </div>
              <div className="flex gap-2">
                {isSearchMode && (
                  <button
                    onClick={handleBackToScan}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    Back to Browse
                  </button>
                )}
                <button
                  onClick={() => setShowSearch(!showSearch)}
                  className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                    showSearch
                      ? 'bg-blue-600 text-white hover:bg-blue-700'
                      : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
                  }`}
                >
                  {showSearch ? 'Hide Search' : 'Search'}
                </button>
                {((scanResult && scanResult.count > 0) || (searchResult && searchResult.count > 0)) && (
                  <button
                    onClick={() => setShowExport(true)}
                    className="px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 transition-colors"
                  >
                    Export
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* Search Panel */}
          {showSearch && currentCF && (
            <SearchPanel onSearch={handleSearch} isLoading={loadingData} />
          )}

          <div className="flex-1 overflow-auto p-6">
            {loadingData ? (
              <div className="flex items-center justify-center h-full">
                <div className="text-center">
                  <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-2"></div>
                  <p className="text-gray-600">Loading data...</p>
                </div>
              </div>
            ) : isSearchMode && searchResult && searchResult.count > 0 ? (
              <div className="space-y-4">
                <div className="bg-white rounded-lg shadow overflow-hidden">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Key
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Value
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Matched Fields
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {searchResult.results.map((item) => (
                        <tr key={item.key} className="hover:bg-gray-50">
                          <td className="px-6 py-4 text-sm font-mono text-gray-900">
                            <div className="flex items-center gap-2">
                              <span className={item.key_is_binary ? 'text-purple-600' : ''}>
                                {item.key}
                              </span>
                              {item.key_is_binary && (
                                <span className="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 rounded">
                                  HEX
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-600">
                            <div className="flex items-center gap-2">
                              <div className={`max-w-md truncate font-mono ${item.value_is_binary ? 'text-purple-600' : ''}`}>
                                {item.value}
                              </div>
                              {item.value_is_binary && (
                                <span className="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 rounded flex-shrink-0">
                                  HEX
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm">
                            <div className="flex gap-1">
                              {item.matched_fields.map((field) => (
                                <span
                                  key={field}
                                  className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded"
                                >
                                  {field}
                                </span>
                              ))}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm">
                            <button
                              onClick={() => setViewModal({ key: item.key, value: item.value })}
                              className="text-blue-600 hover:text-blue-800 font-medium"
                            >
                              View
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : scanResult && scanResult.count > 0 ? (
              <div className="space-y-4">
                <div className="bg-white rounded-lg shadow overflow-hidden">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Key
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Value
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {(scanResult.results_v2 || Object.entries(scanResult.results).map(([key, value]) => ({
                        key,
                        value,
                        key_is_binary: false,
                        value_is_binary: false
                      }))).map((item) => (
                        <tr key={item.key} className="hover:bg-gray-50">
                          <td className="px-6 py-4 text-sm font-mono text-gray-900">
                            <div className="flex items-center gap-2">
                              <span className={item.key_is_binary ? 'text-purple-600' : ''}>
                                {item.key}
                              </span>
                              {item.key_is_binary && (
                                <span className="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 rounded">
                                  HEX
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-600">
                            <div className="flex items-center gap-2">
                              <div className={`max-w-md truncate font-mono ${item.value_is_binary ? 'text-purple-600' : ''}`}>
                                {item.value}
                              </div>
                              {item.value_is_binary && (
                                <span className="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 rounded flex-shrink-0">
                                  HEX
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm">
                            <button
                              onClick={() => setViewModal({ key: item.key, value: item.value })}
                              className="text-blue-600 hover:text-blue-800 font-medium"
                            >
                              View
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                {scanResult.has_more && (
                  <div className="flex justify-center">
                    <button
                      onClick={loadMore}
                      disabled={loadingData}
                      className="px-6 py-2 bg-white border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      {loadingData ? 'Loading...' : 'Load More'}
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-full">
                <div className="text-center text-gray-500">
                  <p className="text-lg mb-2">No data found</p>
                  <p className="text-sm">This column family is empty</p>
                </div>
              </div>
            )}
          </div>
        </main>
      </div>

      {/* View Modal */}
      {viewModal && (
        <ViewModal
          isOpen={true}
          onClose={() => setViewModal(null)}
          keyName={viewModal.key}
          value={viewModal.value}
        />
      )}

      {/* Export Modal */}
      {showExport && currentCF && (
        <ExportModal
          isOpen={true}
          onClose={() => setShowExport(false)}
          data={getExportData()}
          columnFamily={currentCF}
        />
      )}
    </div>
  );
}
