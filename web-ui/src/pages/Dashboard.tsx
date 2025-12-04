import { useEffect, useState } from 'react';
import { useDbStore } from '@/stores/dbStore';
import { listColumnFamilies, scanData, searchData } from '@/api/database';
import { aiAPI } from '@/api/ai';
import { disconnectDatabase, connectDatabase, getCurrentDatabase } from '@/api/dbManager';
import type { ScanResult, SearchRequest, SearchResponse } from '@/types/api';
import ViewModal from '@/components/shared/ViewModal';
import SearchPanel from '@/components/shared/SearchPanel';
import ExportModal from '@/components/shared/ExportModal';
import { AIAssistant } from '@/components/shared/AIAssistant';
import { ToastContainer } from '@/components/shared/Toast';
import { useToast } from '@/hooks/useToast';
import { dbHistory, type FavoriteDatabase } from '@/utils/dbHistory';

export default function Dashboard() {
  const {currentCF, columnFamilies, currentDatabase, setCurrentCF, setColumnFamilies, setCurrentDatabase, disconnect} = useDbStore();
  const [loading, setLoading] = useState(true);
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [searchResult, setSearchResult] = useState<SearchResponse | null>(null);
  const [loadingData, setLoadingData] = useState(false);
  const [viewModal, setViewModal] = useState<{ key: string; value: string } | null>(null);
  const [showSearch, setShowSearch] = useState(false);
  const [isSearchMode, setIsSearchMode] = useState(false);
  const [showExport, setShowExport] = useState(false);
  const [showAI, setShowAI] = useState(false);
  const [aiEnabled, setAIEnabled] = useState(false);
  const [lastSearchParams, setLastSearchParams] = useState<SearchRequest | null>(null);
  
  // Toast notifications
  const { toasts, showError, closeToast } = useToast();

  // Timestamp column state
  const [showTimestampColumn, setShowTimestampColumn] = useState(() => {
    const saved = localStorage.getItem('showTimestampColumn');
    return saved === 'true';
  });

  // Database switcher state
  const [showDatabaseMenu, setShowDatabaseMenu] = useState(false);
  const [switching, setSwitching] = useState(false);

  // Custom path state
  const [customPath, setCustomPath] = useState('');
  const [validatingCustom, setValidatingCustom] = useState(false);
  const [customPathError, setCustomPathError] = useState('');

  // Favorite databases state
  const [favoriteDatabases, setFavoriteDatabases] = useState<FavoriteDatabase[]>([]);

  // Save timestamp column preference to localStorage
  useEffect(() => {
    localStorage.setItem('showTimestampColumn', showTimestampColumn.toString());
  }, [showTimestampColumn]);

  useEffect(() => {
    const init = async () => {
      // Load favorites first
      setFavoriteDatabases(dbHistory.getFavorites());

      // Check AI availability
      checkAI();

      // Load current database info first
      try {
        const currentDbInfo = await getCurrentDatabase();
        if (currentDbInfo) {
          setCurrentDatabase(currentDbInfo);
        }
      } catch (err) {
        console.log('[Dashboard] Failed to get current database info:', err);
      }

      // Try to load column families (check if database is connected)
      try {
        await loadColumnFamilies();
      } catch (err: any) {
        // If loading fails, show database selection modal
        console.log('[Dashboard] No database connected, showing switch DB modal');
        setShowDatabaseMenu(true);
      }
    };

    init();
  }, []);

  const checkAI = async () => {
    try {
      const health = await aiAPI.checkHealth();
      setAIEnabled(health.ai_enabled);
    } catch {
      setAIEnabled(false);
    }
  };

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
      showError(err.message || 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  const loadData = async (cursor?: string) => {
    if (!currentCF) return;

    try {
      setLoadingData(true);
      
      // Save scroll position before loading
      const scrollContainer = document.querySelector('.overflow-auto');
      const scrollHeight = scrollContainer?.scrollHeight || 0;
      
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
        
        // Restore scroll position after a short delay to allow DOM update
        setTimeout(() => {
          if (scrollContainer) {
            const newScrollHeight = scrollContainer.scrollHeight;
            scrollContainer.scrollTop = newScrollHeight - scrollHeight + scrollContainer.scrollTop;
          }
        }, 50);
      } else {
        setScanResult(result);
      }
    } catch (err: any) {
      showError(err.message || 'An error occurred');
    } finally {
      setLoadingData(false);
    }
  };

  const loadMore = () => {
    if (scanResult?.next_cursor) {
      loadData(scanResult.next_cursor);
    }
  };

  const loadMoreSearch = async () => {
    if (!currentCF || !searchResult?.next_cursor || !lastSearchParams) return;

    try {
      setLoadingData(true);
      
      // Save scroll position before loading
      const scrollContainer = document.querySelector('.overflow-auto');
      const scrollHeight = scrollContainer?.scrollHeight || 0;
      
      // Use stored search params with new cursor
      const nextParams: SearchRequest = {
        ...lastSearchParams,
        after: searchResult.next_cursor
      };
      
      const result = await searchData(currentCF, nextParams);
      
      // Append new results to existing results
      setSearchResult({
        ...result,
        results: [...searchResult.results, ...result.results],
        count: searchResult.count + result.count
      });
      
      // Restore scroll position after a short delay to allow DOM update
      setTimeout(() => {
        if (scrollContainer) {
          const newScrollHeight = scrollContainer.scrollHeight;
          scrollContainer.scrollTop = newScrollHeight - scrollHeight + scrollContainer.scrollTop;
        }
      }, 50);
    } catch (err: any) {
      showError(err.message || 'An error occurred');
    } finally {
      setLoadingData(false);
    }
  };

  const handleSearch = async (params: SearchRequest) => {
    if (!currentCF) return;

    try {
      setLoadingData(true);
      setIsSearchMode(true);
      setLastSearchParams(params); // Store search params for pagination
      const result = await searchData(currentCF, params);
      setSearchResult(result);
      setScanResult(null); // Clear scan results
    } catch (err: any) {
      showError(err.message || 'An error occurred');
    } finally {
      setLoadingData(false);
    }
  };

  const handleBackToScan = () => {
    setIsSearchMode(false);
    setSearchResult(null);
    setLastSearchParams(null); // Clear stored search params
    setShowSearch(false);
    loadData(); // Reload scan data
  };

  const loadAvailableDatabases = () => {
    try {
      // Reload favorite databases
      setFavoriteDatabases(dbHistory.getFavorites());
    } catch (err) {
      console.error('Failed to load databases:', err);
    }
  };

  const handleSwitchDatabase = async (dbPath: string) => {
    try {
      setSwitching(true);
      setShowDatabaseMenu(false);

      console.log('[handleSwitchDatabase] Disconnecting from current database...');
      // Disconnect from current database
      await disconnectDatabase();

      console.log('[handleSwitchDatabase] Connecting to new database:', dbPath);
      // Connect to new database
      const response = await connectDatabase({ path: dbPath, read_only: true });

      console.log('[handleSwitchDatabase] Response:', response);

      // Backend returns {message: '...', data: {...}} format
      // Check if connection was successful by checking if data exists
      if (response.data) {
        console.log('[handleSwitchDatabase] Connection successful, adding to favorites...');

        // Add to favorites BEFORE setting state
        dbHistory.addFavorite(dbPath);
        console.log('[handleSwitchDatabase] Added to favorites');

        // Update state - use 'data' field from backend response
        setCurrentDatabase(response.data);

        // Reload column families and data
        setScanResult(null);
        setSearchResult(null);
        setCurrentCF(null);

        console.log('[handleSwitchDatabase] Reloading page...');
        // Auto-refresh the page to ensure clean state
        window.location.reload();
      } else {
        console.error('[handleSwitchDatabase] Connection failed:', response);
        showError(response.message || 'Failed to connect to database');
        setSwitching(false);
      }
    } catch (err: any) {
      console.error('[handleSwitchDatabase] Error:', err);
      showError(err.response?.data?.error || err.message || 'Failed to switch database');
      setSwitching(false);
    }
  };

  const handleDisconnect = async () => {
    try {
      await disconnectDatabase();
      disconnect();
      // Reload the page to reset state and show the database selection modal
      window.location.reload();
    } catch (err: any) {
      showError(err.message || 'Failed to disconnect');
    }
  };

  const handleConnectCustomPath = async () => {
    if (!customPath.trim()) {
      setCustomPathError('Please enter a database path');
      return;
    }

    try {
      setValidatingCustom(true);
      setCustomPathError('');
      await handleSwitchDatabase(customPath.trim());
      setCustomPath('');
      setShowDatabaseMenu(false);
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to connect';
      setCustomPathError(errorMsg);
      showError(errorMsg);
    } finally {
      setValidatingCustom(false);
    }
  };

  const handleRemoveFavorite = (path: string) => {
    dbHistory.removeFavorite(path);
    setFavoriteDatabases(dbHistory.getFavorites());
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

  return (
    <>
      <ToastContainer toasts={toasts} onClose={closeToast} />
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="px-6 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              RocksDB Web UI
            </h1>
            <p className="text-sm text-gray-600">
              Database Management Interface
            </p>
          </div>

          {/* Database Info and Switcher */}
          <div className="flex items-center gap-3">
            {currentDatabase && (
              <div className="text-right">
                <div className="text-sm font-medium text-gray-900">
                  {currentDatabase.path.split('/').pop() || 'Database'}
                </div>
                <div className="text-xs text-gray-500 font-mono">
                  {currentDatabase.path}
                </div>
                <div className="text-xs text-gray-500">
                  {currentDatabase.column_family_count} column families
                </div>
              </div>
            )}

            <button
              onClick={() => {
                setShowDatabaseMenu(true);
                loadAvailableDatabases();
              }}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
              disabled={switching}
            >
              {switching ? 'Switching...' : 'Switch DB'}
            </button>
          </div>
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
                {aiEnabled && (
                  <button
                    onClick={() => setShowAI(true)}
                    className="px-4 py-2 text-sm font-medium text-white bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors flex items-center gap-2"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                    </svg>
                    AI Assistant
                  </button>
                )}
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
                  <>
                    <button
                      onClick={() => setShowTimestampColumn(!showTimestampColumn)}
                      className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                        showTimestampColumn
                          ? 'bg-indigo-600 text-white hover:bg-indigo-700'
                          : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
                      }`}
                      title="Show/hide timestamp column"
                    >
                      <svg className="w-4 h-4 inline-block mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      {showTimestampColumn ? 'Hide Timestamp' : 'Show Timestamp'}
                    </button>
                    <button
                      onClick={() => setShowExport(true)}
                      className="px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 transition-colors"
                    >
                      Export
                    </button>
                  </>
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
            ) : isSearchMode && searchResult ? (
              searchResult.results && searchResult.results.length > 0 ? (
                <div className="space-y-4">
                  <div className="bg-white rounded-lg shadow overflow-hidden">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Key
                          </th>
                          {showTimestampColumn && (
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                              Timestamp
                            </th>
                          )}
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
                        {(searchResult.results || []).map((item) => (
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
                          {showTimestampColumn && (
                            <td className="px-6 py-4 text-sm text-gray-700">
                              {item.timestamp ? (
                                <span className="font-mono text-xs">{item.timestamp}</span>
                              ) : (
                                <span className="text-gray-400">-</span>
                              )}
                            </td>
                          )}
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
                              {(item.matched_fields || []).map((field) => (
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

                {/* Pagination for search results */}
                {searchResult.has_more && (
                  <div className="flex justify-center">
                    <button
                      onClick={loadMoreSearch}
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
                    <p className="text-lg mb-2">No search results found</p>
                    <p className="text-sm mb-4">Try adjusting your search criteria or range</p>
                    <button
                      onClick={handleBackToScan}
                      className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors"
                    >
                      Back to Scan
                    </button>
                  </div>
                </div>
              )
            ) : scanResult && ((scanResult.results_v2 && scanResult.results_v2.length > 0) || scanResult.count > 0) ? (
              <div className="space-y-4">
                <div className="bg-white rounded-lg shadow overflow-hidden">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Key
                        </th>
                        {showTimestampColumn && (
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Timestamp
                          </th>
                        )}
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
                        value_is_binary: false,
                        timestamp: undefined
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
                          {showTimestampColumn && (
                            <td className="px-6 py-4 text-sm text-gray-700">
                              {item.timestamp ? (
                                <span className="font-mono text-xs">{item.timestamp}</span>
                              ) : (
                                <span className="text-gray-400">-</span>
                              )}
                            </td>
                          )}
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

      {/* AI Assistant */}
      <AIAssistant
        isOpen={showAI}
        onClose={() => setShowAI(false)}
      />

      {/* Switch Database Modal */}
      {showDatabaseMenu && (
        <div className="fixed inset-0 bg-gray-900 bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-lg p-8 max-w-4xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-bold text-gray-900">Switch Database</h2>
              <button
                onClick={() => setShowDatabaseMenu(false)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Current Database Section */}
            <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg">
              <h3 className="text-sm font-semibold text-green-800 mb-2">Current Database</h3>
              <div className="text-sm text-green-900 font-mono">{currentDatabase?.path}</div>
              <div className="text-xs text-green-700 mt-1">
                {currentDatabase?.column_family_count} column families
              </div>
            </div>

            {/* Favorite Databases */}
            <div className="mb-6">
              <h3 className="text-lg font-medium text-gray-800 mb-3">Favorite Databases</h3>
              {favoriteDatabases.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {favoriteDatabases.map((db) => {
                    const isCurrent = db.path === currentDatabase?.path;
                    return (
                      <div
                        key={db.path}
                        className={`border rounded-md p-3 transition-all ${
                          isCurrent
                            ? 'border-green-400 bg-green-50'
                            : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
                        }`}
                      >
                        <div className="flex justify-between items-start mb-2">
                          <div className="flex-1 min-w-0">
                            <h4 className="text-sm font-medium text-gray-900 truncate">{db.name}</h4>
                            <p className="text-xs text-gray-500 font-mono mt-0.5 truncate">{db.path}</p>
                            {db.lastConnected && (
                              <p className="text-xs text-gray-400 mt-0.5">
                                {new Date(db.lastConnected).toLocaleString()}
                              </p>
                            )}
                          </div>
                          {isCurrent && (
                            <span className="ml-2 text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full font-medium flex-shrink-0">
                              Current
                            </span>
                          )}
                        </div>

                        <div className="flex gap-2 mt-2">
                          <button
                            onClick={() => handleSwitchDatabase(db.path)}
                            disabled={isCurrent || switching}
                            className={`flex-1 px-3 py-1.5 rounded text-sm font-medium transition-all ${
                              isCurrent || switching
                                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                                : 'bg-blue-500 text-white hover:bg-blue-600 hover:shadow-sm active:scale-95'
                            }`}
                          >
                            {isCurrent ? 'Connected' : switching ? 'Opening...' : 'Open'}
                          </button>
                          {!isCurrent && (
                            <button
                              onClick={() => handleRemoveFavorite(db.path)}
                              disabled={switching}
                              className="px-3 py-1.5 border border-red-300 text-red-600 rounded text-sm font-medium hover:bg-red-50 hover:border-red-400 transition-all disabled:opacity-50 active:scale-95"
                            >
                              Delete
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="text-center py-8 text-gray-400 text-sm border border-dashed border-gray-300 rounded">
                  No favorite databases yet. Connect to a database to add it to favorites.
                </div>
              )}
            </div>

            {/* Custom Path Input */}
            <div className="border-t pt-4 mb-4">
              <h3 className="text-lg font-medium text-gray-800 mb-3">Or Enter Custom Path</h3>
              <div className="space-y-2">
                <input
                  type="text"
                  value={customPath}
                  onChange={(e) => {
                    setCustomPath(e.target.value);
                    setCustomPathError('');
                  }}
                  onKeyPress={(e) => {
                    if (e.key === 'Enter' && !validatingCustom) {
                      handleConnectCustomPath();
                    }
                  }}
                  placeholder="/path/to/rocksdb"
                  className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:border-blue-400 focus:ring-1 focus:ring-blue-400 transition-colors"
                  disabled={validatingCustom || switching}
                />

                {customPathError && (
                  <p className="text-xs text-red-500">{customPathError}</p>
                )}

                <button
                  onClick={handleConnectCustomPath}
                  disabled={!customPath.trim() || validatingCustom || switching}
                  className={`w-full px-3 py-2 rounded text-sm font-medium transition-all ${
                    customPath.trim() && !validatingCustom && !switching
                      ? 'bg-blue-500 text-white hover:bg-blue-600 hover:shadow-sm active:scale-95'
                      : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                  }`}
                >
                  {validatingCustom ? 'Connecting...' : 'Connect'}
                </button>
              </div>
            </div>

            {/* Disconnect Button */}
            <div className="border-t pt-3">
              <button
                onClick={() => {
                  setShowDatabaseMenu(false);
                  handleDisconnect();
                }}
                className="w-full px-3 py-2 border border-red-300 text-red-600 rounded text-sm font-medium hover:bg-red-50 hover:border-red-400 transition-all active:scale-95"
              >
                Disconnect from Database
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
    </>
  );
}
