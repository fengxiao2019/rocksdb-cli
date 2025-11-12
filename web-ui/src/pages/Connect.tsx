import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { healthCheck } from '@/api/database';
import { listDatabases, connectDatabase, getCurrentDatabase, validatePath } from '@/api/dbManager';
import { useDbStore } from '@/stores/dbStore';
import type { AvailableDatabase } from '@/types/api';

export default function Connect() {
  const navigate = useNavigate();
  const { setCurrentDatabase, setAvailableDatabases } = useDbStore();

  const [loading, setLoading] = useState(true);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState('');
  const [serverError, setServerError] = useState('');

  const [databases, setDatabases] = useState<AvailableDatabase[]>([]);
  const [customPath, setCustomPath] = useState('');
  const [validating, setValidating] = useState(false);
  const [validationError, setValidationError] = useState('');

  useEffect(() => {
    // Check if already connected, then load available databases
    checkInitialState();
  }, []);

  const checkInitialState = async () => {
    try {
      setLoading(true);
      setServerError('');

      // Check server health
      await healthCheck();

      // Check if already connected to a database
      const currentDb = await getCurrentDatabase();
      if (currentDb) {
        // Already connected, go to dashboard
        setCurrentDatabase(currentDb);
        navigate('/dashboard');
        return;
      }

      // Load available databases
      const dbList = await listDatabases();
      setDatabases(dbList.databases || []);
      setAvailableDatabases(dbList.databases || []);

    } catch (err: any) {
      setServerError(err.message || 'Cannot connect to database server');
    } finally {
      setLoading(false);
    }
  };

  const handleConnectToDatabase = async (path: string) => {
    try {
      setConnecting(true);
      setError('');

      const response = await connectDatabase({ path, read_only: true });

      if (response.success && response.database) {
        setCurrentDatabase(response.database);
        navigate('/dashboard');
      } else {
        setError(response.message || 'Failed to connect');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Connection failed');
    } finally {
      setConnecting(false);
    }
  };

  const handleValidateCustomPath = async () => {
    if (!customPath.trim()) {
      setValidationError('Please enter a database path');
      return;
    }

    try {
      setValidating(true);
      setValidationError('');

      const result = await validatePath(customPath);

      if (result.valid) {
        // Path is valid, connect to it
        await handleConnectToDatabase(customPath);
      } else {
        setValidationError(result.error || 'Invalid database path');
      }
    } catch (err: any) {
      setValidationError(err.response?.data?.error || 'Failed to validate path');
    } finally {
      setValidating(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg p-8 max-w-2xl w-full">
          <div className="text-center py-8">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
            <p className="text-gray-600">Connecting to server...</p>
          </div>
        </div>
      </div>
    );
  }

  if (serverError) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
          <div className="text-center mb-6">
            <h1 className="text-3xl font-bold text-gray-900 mb-2">RocksDB Web UI</h1>
            <p className="text-gray-600">Web interface for RocksDB database management</p>
          </div>

          <div className="space-y-4">
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <h3 className="text-red-800 font-semibold mb-2">Server Connection Failed</h3>
              <p className="text-red-600 text-sm">{serverError}</p>
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <h4 className="text-blue-800 font-semibold mb-2">Make sure the server is running:</h4>
              <code className="text-sm bg-blue-100 px-2 py-1 rounded block">
                ./rocksdb-web-server --port 8080
              </code>
            </div>

            <button
              onClick={checkInitialState}
              className="w-full bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              Retry Connection
            </button>
          </div>

          <div className="mt-6 text-center text-sm text-gray-500">
            Server: <span className="font-mono">localhost:8080</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg p-8 max-w-4xl w-full">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Select Database</h1>
          <p className="text-gray-600">Choose a database to connect or enter a custom path</p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <p className="text-red-600 text-sm">{error}</p>
          </div>
        )}

        {/* Available Databases */}
        {databases.length > 0 && (
          <div className="mb-8">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Available Databases</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {databases.map((db) => (
                <div
                  key={db.path}
                  className="border border-gray-200 rounded-lg p-4 hover:border-blue-500 transition-colors"
                >
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex-1">
                      <h3 className="font-semibold text-gray-900">{db.name}</h3>
                      <p className="text-sm text-gray-500 font-mono mt-1">{db.path}</p>
                    </div>
                    {db.is_valid ? (
                      <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded">Valid</span>
                    ) : (
                      <span className="text-xs bg-red-100 text-red-800 px-2 py-1 rounded">Invalid</span>
                    )}
                  </div>

                  {db.column_families && db.column_families.length > 0 && (
                    <p className="text-sm text-gray-600 mb-3">
                      Column Families: {db.column_families.length}
                    </p>
                  )}

                  {db.error && (
                    <p className="text-sm text-red-600 mb-3">{db.error}</p>
                  )}

                  <button
                    onClick={() => handleConnectToDatabase(db.path)}
                    disabled={!db.is_valid || connecting}
                    className={`w-full px-4 py-2 rounded-lg font-medium transition-colors ${
                      db.is_valid && !connecting
                        ? 'bg-blue-600 text-white hover:bg-blue-700'
                        : 'bg-gray-300 text-gray-500 cursor-not-allowed'
                    }`}
                  >
                    {connecting ? 'Connecting...' : 'Connect'}
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Custom Path Input */}
        <div>
          <h2 className="text-xl font-semibold text-gray-900 mb-4">Or Enter Custom Path</h2>
          <div className="space-y-3">
            <input
              type="text"
              value={customPath}
              onChange={(e) => {
                setCustomPath(e.target.value);
                setValidationError('');
              }}
              placeholder="/path/to/rocksdb"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={validating || connecting}
            />

            {validationError && (
              <p className="text-sm text-red-600">{validationError}</p>
            )}

            <button
              onClick={handleValidateCustomPath}
              disabled={!customPath.trim() || validating || connecting}
              className={`w-full px-4 py-2 rounded-lg font-medium transition-colors ${
                customPath.trim() && !validating && !connecting
                  ? 'bg-blue-600 text-white hover:bg-blue-700'
                  : 'bg-gray-300 text-gray-500 cursor-not-allowed'
              }`}
            >
              {validating ? 'Validating...' : connecting ? 'Connecting...' : 'Connect to Custom Path'}
            </button>
          </div>
        </div>

        <div className="mt-8 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <h4 className="text-blue-800 font-semibold mb-2">Configure Database Mount Points</h4>
          <p className="text-sm text-blue-700 mb-2">
            Set the DB_MOUNT_POINTS environment variable to automatically discover databases:
          </p>
          <code className="text-sm bg-blue-100 px-2 py-1 rounded block">
            export DB_MOUNT_POINTS=/db1,/db2,/db3
          </code>
        </div>
      </div>
    </div>
  );
}
