import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { healthCheck } from '@/api/database';
import { useDbStore } from '@/stores/dbStore';

export default function Connect() {
  const navigate = useNavigate();
  const setConnected = useDbStore((state) => state.setConnected);
  const [checking, setChecking] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    // Check if backend is accessible
    checkConnection();
  }, []);

  const checkConnection = async () => {
    try {
      setChecking(true);
      setError('');
      await healthCheck();
      // Backend is accessible, mark as connected
      setConnected(true);
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Cannot connect to database server');
      setConnected(false);
    } finally {
      setChecking(false);
    }
  };

  const handleRetry = () => {
    checkConnection();
  };

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
        <div className="text-center mb-6">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">
            RocksDB Web UI
          </h1>
          <p className="text-gray-600">
            Web interface for RocksDB database management
          </p>
        </div>

        {checking ? (
          <div className="text-center py-8">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
            <p className="text-gray-600">Connecting to database server...</p>
          </div>
        ) : error ? (
          <div className="space-y-4">
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <h3 className="text-red-800 font-semibold mb-2">
                Connection Failed
              </h3>
              <p className="text-red-600 text-sm">{error}</p>
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <h4 className="text-blue-800 font-semibold mb-2">
                Make sure the server is running:
              </h4>
              <code className="text-sm bg-blue-100 px-2 py-1 rounded block">
                ./rocksdb-web-server --db ./testdb --port 8080
              </code>
            </div>

            <button
              onClick={handleRetry}
              className="w-full bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              Retry Connection
            </button>
          </div>
        ) : null}

        <div className="mt-6 text-center text-sm text-gray-500">
          Server: <span className="font-mono">localhost:8080</span>
        </div>
      </div>
    </div>
  );
}
