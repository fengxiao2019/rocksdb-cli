# Dynamic Database Selection Implementation Guide

This document contains the complete implementation for adding dynamic database selection to the Web UI.

## Already Completed

✅ 1. Database Manager Service (`internal/service/db_manager.go`)
✅ 2. Database Manager Handler (`internal/api/handlers/db_manager_handler.go`)

## Remaining Implementation Steps

### 3. Modify Router to Support Dynamic Database

**File**: `internal/api/router.go`

Add these changes to the `SetupRouterWithUI` function:

```go
// Add parameter for DBManager
func SetupRouterWithUI(dbManager *service.DBManager) *gin.Engine {
    router := gin.Default()

    // ... existing CORS and middleware setup ...

    // Create DB Manager handler
    dbManagerHandler := handlers.NewDBManagerHandler(dbManager)

    // Database management routes (no auth required - accessible before DB connection)
    dbRoutes := router.Group("/api/v1/databases")
    {
        dbRoutes.GET("/current", dbManagerHandler.GetCurrent)
        dbRoutes.GET("/list", dbManagerHandler.ListAvailable)
        dbRoutes.GET("/status", dbManagerHandler.GetStatus)
        dbRoutes.POST("/connect", dbManagerHandler.Connect)
        dbRoutes.POST("/disconnect", dbManagerHandler.Disconnect)
        dbRoutes.POST("/validate", dbManagerHandler.Validate)
    }

    // Middleware to check DB connection for data routes
    dbCheckMiddleware := func(c *gin.Context) {
        db, err := dbManager.GetCurrentDB()
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "error": "No database connected. Please connect to a database first.",
            })
            c.Abort()
            return
        }
        // Store DB in context for handlers to use
        c.Set("db", db)
        c.Next()
    }

    // Existing API routes - now require DB connection
    api := router.Group("/api/v1")
    api.Use(dbCheckMiddleware) // Add middleware
    {
        // Get DB from context in each handler
        // Example in handlers:
        // db := c.MustGet("db").(db.KeyValueDB)

        // ... existing routes ...
    }

    // ... rest of setup ...

    return router
}
```

### 4. Modify Existing Handlers to Use Dynamic DB

**Pattern for all handlers** in `internal/api/handlers/*.go`:

Change from:
```go
type DatabaseHandler struct {
    service *service.DatabaseService
}
```

To:
```go
type DatabaseHandler struct {
    getDB func() (db.KeyValueDB, error) // Function to get current DB
}

// In handler methods:
func (h *DatabaseHandler) SomeMethod(c *gin.Context) {
    db, err := h.getDB()
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
        return
    }
    // Use db...
}
```

Or use context approach (simpler):
```go
func (h *DatabaseHandler) SomeMethod(c *gin.Context) {
    db := c.MustGet("db").(db.KeyValueDB)
    // Use db...
}
```

### 5. Modify Web Server Startup

**File**: `cmd/web-server/main.go`

```go
func main() {
    // ... existing flag parsing ...

    // Create DB Manager
    dbManager := service.NewDBManager()

    // If initial DB path is provided, connect to it
    if *dbPath != "" {
        log.Printf("Connecting to initial database: %s\n", *dbPath)
        if _, err := dbManager.Connect(*dbPath); err != nil {
            log.Printf("Warning: Failed to connect to initial database: %v\n", err)
            log.Println("Starting without database connection. Use Web UI to connect.")
        } else {
            log.Println("Successfully connected to initial database")
        }
    } else {
        log.Println("No initial database specified. Use Web UI to connect.")
    }

    // Setup router with DB Manager
    router := api.SetupRouterWithUI(dbManager)

    // ... rest of server startup ...
}
```

### 6. Frontend API Client

**File**: `web-ui/src/api/database.ts`

```typescript
export interface DatabaseInfo {
  path: string;
  readOnly: boolean;
  connected: boolean;
  connectedAt?: string;
  cfCount: number;
  columnFamilies: string[];
}

export interface ConnectRequest {
  path: string;
}

export interface ValidateRequest {
  path: string;
}

export const databaseAPI = {
  // Get current database info
  async getCurrent(): Promise<{ connected: boolean; data?: DatabaseInfo }> {
    const response = await fetch('/api/v1/databases/current');
    return response.json();
  },

  // List available databases
  async list(): Promise<{ databases: string[]; mountPoints: string[] }> {
    const response = await fetch('/api/v1/databases/list');
    const result = await response.json();
    return result.data;
  },

  // Connect to a database
  async connect(path: string): Promise<DatabaseInfo> {
    const response = await fetch('/api/v1/databases/connect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });
    const result = await response.json();
    if (!response.ok) {
      throw new Error(result.error || 'Failed to connect');
    }
    return result.data;
  },

  // Disconnect from current database
  async disconnect(): Promise<void> {
    const response = await fetch('/api/v1/databases/disconnect', {
      method: 'POST',
    });
    if (!response.ok) {
      const result = await response.json();
      throw new Error(result.error || 'Failed to disconnect');
    }
  },

  // Validate a database path
  async validate(path: string): Promise<{ valid: boolean; error?: string }> {
    const response = await fetch('/api/v1/databases/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });
    return response.json();
  },

  // Get connection status
  async getStatus(): Promise<{ connected: boolean; database?: DatabaseInfo }> {
    const response = await fetch('/api/v1/databases/status');
    return response.json();
  },
};
```

### 7. Update Frontend State Management

**File**: `web-ui/src/stores/dbStore.ts`

```typescript
import { create } from 'zustand';
import { DatabaseInfo, databaseAPI } from '../api/database';

interface DbState {
  // Connection state
  connected: boolean;
  connecting: boolean;
  databasePath: string | null;
  databaseInfo: DatabaseInfo | null;

  // Column family state
  currentCF: string | null;
  columnFamilies: string[];

  // Available databases
  availableDatabases: string[];

  // Actions
  connect: (path: string) => Promise<void>;
  disconnect: () => Promise<void>;
  refreshCurrent: () => Promise<void>;
  loadAvailableDatabases: () => Promise<void>;
  setCurrentCF: (cf: string) => void;
}

export const useDbStore = create<DbState>((set, get) => ({
  connected: false,
  connecting: false,
  databasePath: null,
  databaseInfo: null,
  currentCF: null,
  columnFamilies: [],
  availableDatabases: [],

  connect: async (path: string) => {
    set({ connecting: true });
    try {
      const info = await databaseAPI.connect(path);
      set({
        connected: true,
        connecting: false,
        databasePath: info.path,
        databaseInfo: info,
        columnFamilies: info.columnFamilies,
        currentCF: info.columnFamilies[0] || null,
      });

      // Save to localStorage
      localStorage.setItem('lastDatabasePath', path);
    } catch (error) {
      set({ connecting: false });
      throw error;
    }
  },

  disconnect: async () => {
    await databaseAPI.disconnect();
    set({
      connected: false,
      databasePath: null,
      databaseInfo: null,
      currentCF: null,
      columnFamilies: [],
    });
  },

  refreshCurrent: async () => {
    const result = await databaseAPI.getCurrent();
    if (result.connected && result.data) {
      set({
        connected: true,
        databasePath: result.data.path,
        databaseInfo: result.data,
        columnFamilies: result.data.columnFamilies,
        currentCF: result.data.columnFamilies[0] || null,
      });
    } else {
      set({
        connected: false,
        databasePath: null,
        databaseInfo: null,
      });
    }
  },

  loadAvailableDatabases: async () => {
    const result = await databaseAPI.list();
    set({ availableDatabases: result.databases });
  },

  setCurrentCF: (cf: string) => set({ currentCF: cf }),
}));
```

### 8. Database Selector Component

**File**: `web-ui/src/components/DatabaseSelector.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { useDbStore } from '../stores/dbStore';
import { databaseAPI } from '../api/database';

export const DatabaseSelector: React.FC = () => {
  const {
    connected,
    connecting,
    databasePath,
    databaseInfo,
    availableDatabases,
    connect,
    disconnect,
    loadAvailableDatabases,
  } = useDbStore();

  const [selectedPath, setSelectedPath] = useState('');
  const [customPath, setCustomPath] = useState('');
  const [useCustom, setUseCustom] = useState(false);
  const [error, setError] = useState('');
  const [showDisconnectConfirm, setShowDisconnectConfirm] = useState(false);

  // Load available databases on mount
  useEffect(() => {
    loadAvailableDatabases();

    // Try to restore last database
    const lastPath = localStorage.getItem('lastDatabasePath');
    if (lastPath) {
      setSelectedPath(lastPath);
    }
  }, [loadAvailableDatabases]);

  const handleConnect = async () => {
    const path = useCustom ? customPath : selectedPath;
    if (!path) {
      setError('Please select or enter a database path');
      return;
    }

    setError('');

    // Validate first
    try {
      const validation = await databaseAPI.validate(path);
      if (!validation.valid) {
        setError(validation.error || 'Invalid database path');
        return;
      }
    } catch (err) {
      setError('Failed to validate path: ' + err.message);
      return;
    }

    // Show confirmation if already connected
    if (connected) {
      if (!window.confirm(`Disconnect from current database (${databasePath}) and connect to ${path}?`)) {
        return;
      }
    }

    // Connect
    try {
      await connect(path);
    } catch (err) {
      setError('Failed to connect: ' + err.message);
    }
  };

  const handleDisconnect = async () => {
    if (!window.confirm('Are you sure you want to disconnect from the current database?')) {
      return;
    }

    try {
      await disconnect();
      setError('');
    } catch (err) {
      setError('Failed to disconnect: ' + err.message);
    }
  };

  return (
    <div className="database-selector">
      <h3>Database Connection</h3>

      {connected ? (
        <div className="connected-info">
          <div className="status success">
            ✓ Connected to: {databasePath}
          </div>
          <div className="db-details">
            <span className="badge">Read-Only</span>
            <span>{databaseInfo?.cfCount} Column Families</span>
            <span>Connected: {new Date(databaseInfo?.connectedAt).toLocaleString()}</span>
          </div>
          <button onClick={handleDisconnect} className="btn-disconnect">
            Disconnect
          </button>
        </div>
      ) : (
        <div className="connect-form">
          <div className="form-group">
            <label>
              <input
                type="radio"
                checked={!useCustom}
                onChange={() => setUseCustom(false)}
              />
              Select from available databases
            </label>

            {!useCustom && (
              <select
                value={selectedPath}
                onChange={(e) => setSelectedPath(e.target.value)}
                disabled={connecting}
              >
                <option value="">-- Select a database --</option>
                {availableDatabases.map((db) => (
                  <option key={db} value={db}>{db}</option>
                ))}
              </select>
            )}
          </div>

          <div className="form-group">
            <label>
              <input
                type="radio"
                checked={useCustom}
                onChange={() => setUseCustom(true)}
              />
              Enter custom path
            </label>

            {useCustom && (
              <input
                type="text"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                placeholder="/path/to/database"
                disabled={connecting}
              />
            )}
          </div>

          {error && <div className="error">{error}</div>}

          <button
            onClick={handleConnect}
            disabled={connecting || (!selectedPath && !customPath)}
            className="btn-connect"
          >
            {connecting ? 'Connecting...' : 'Connect (Read-Only)'}
          </button>
        </div>
      )}
    </div>
  );
};
```

### 9. Update Connect Page

**File**: `web-ui/src/pages/Connect.tsx`

```typescript
import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { DatabaseSelector } from '../components/DatabaseSelector';
import { useDbStore } from '../stores/dbStore';

export const Connect: React.FC = () => {
  const navigate = useNavigate();
  const { connected, refreshCurrent } = useDbStore();

  useEffect(() => {
    // Check if already connected
    refreshCurrent();
  }, [refreshCurrent]);

  useEffect(() => {
    // Redirect to dashboard if connected
    if (connected) {
      navigate('/dashboard');
    }
  }, [connected, navigate]);

  return (
    <div className="connect-page">
      <div className="connect-container">
        <h1>RocksDB CLI</h1>
        <p className="subtitle">Connect to a RocksDB database to get started</p>

        <DatabaseSelector />

        <div className="info-section">
          <h4>📖 About</h4>
          <ul>
            <li>All databases are opened in <strong>read-only mode</strong></li>
            <li>You can switch between databases without restarting the service</li>
            <li>Recent databases are remembered for quick access</li>
          </ul>
        </div>
      </div>
    </div>
  );
};
```

### 10. Update Dashboard to Show DB Status

**File**: `web-ui/src/pages/Dashboard.tsx`

Add a database status bar at the top:

```typescript
// Add to Dashboard component
const { databasePath, databaseInfo, disconnect } = useDbStore();

return (
  <div className="dashboard">
    <div className="db-status-bar">
      <div className="db-info">
        <span className="db-path">📁 {databasePath}</span>
        <span className="badge read-only">Read-Only</span>
        <span className="cf-count">{databaseInfo?.cfCount} Column Families</span>
      </div>
      <button onClick={() => {
        if (window.confirm('Switch to a different database?')) {
          disconnect();
        }
      }} className="btn-switch">
        Switch Database
      </button>
    </div>

    {/* Rest of dashboard */}
  </div>
);
```

### 11. Update Docker Configuration

**File**: `docker-compose.yml`

```yaml
version: '3.8'

services:
  rocksdb-cli-web:
    build:
      context: .
      dockerfile: Dockerfile.web
    image: rocksdb-cli-web:latest
    ports:
      - "${WEB_PORT:-8090}:8090"
    volumes:
      # Multiple database volumes
      - ${DB1_PATH:-./data/db1}:/db1
      - ${DB2_PATH:-./data/db2}:/db2
      - ${DB3_PATH:-./data/db3}:/db3
      # Or single data volume with subdirectories
      - ${DATA_PATH:-./data}:/data
    environment:
      # Database mount points (comma-separated)
      - DB_MOUNT_POINTS=/db1,/db2,/db3,/data

      # AI Configuration
      - GRAPHCHAIN_LLM_PROVIDER=${GRAPHCHAIN_LLM_PROVIDER:-openai}
      - GRAPHCHAIN_API_KEY=${GRAPHCHAIN_API_KEY}
      # ... other env vars ...
    restart: unless-stopped
```

**File**: `.env.example`

Add these lines:

```bash
# Database Mount Points (comma-separated)
DB_MOUNT_POINTS=/db1,/db2,/db3,/data

# Database Paths (for docker-compose volumes)
DB1_PATH=./data/db1
DB2_PATH=./data/db2
DB3_PATH=./data/db3
DATA_PATH=./data
```

## Testing

1. Start the service:
```bash
docker-compose up -d
```

2. Open web UI: `http://localhost:8090`

3. You should see the connection page with database selector

4. Select or enter a database path and connect

5. Navigate through the UI - all operations are read-only

6. Try switching databases using the "Switch Database" button

## Summary

This implementation provides:
- ✅ Dynamic database selection in Web UI
- ✅ Read-only mode enforced for all connections
- ✅ No service restart required for switching databases
- ✅ Support for multiple volume mounts
- ✅ Support for custom path input
- ✅ Database validation before connection
- ✅ Confirmation dialogs for destructive operations
- ✅ Thread-safe database switching
- ✅ Remember last used database
- ✅ List available databases from configured mount points
