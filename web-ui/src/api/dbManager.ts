import client from './client';
import type {
  DatabaseListResponse,
  ConnectRequest,
  ConnectResponse,
  DatabaseConnectionInfo,
  DatabaseStatus,
  ValidatePathRequest,
  ValidatePathResponse,
} from '../types/api';

/**
 * List all available databases from configured mount points
 */
export const listDatabases = async (): Promise<DatabaseListResponse> => {
  const response = await client.get<{ data: DatabaseListResponse }>('/databases/list');
  return response.data.data;
};

/**
 * Connect to a RocksDB database
 */
export const connectDatabase = async (request: ConnectRequest): Promise<ConnectResponse> => {
  const response = await client.post<ConnectResponse>('/databases/connect', request);
  return response.data;
};

/**
 * Disconnect from the current database
 */
export const disconnectDatabase = async (): Promise<{ message: string }> => {
  const response = await client.post<{ message: string }>('/databases/disconnect');
  return response.data;
};

/**
 * Get current database connection information
 */
export const getCurrentDatabase = async (): Promise<DatabaseConnectionInfo | null> => {
  try {
    const response = await client.get<{ data: DatabaseConnectionInfo }>('/databases/current');
    return response.data.data;
  } catch (error: any) {
    // Not connected
    if (error.response?.status === 404) {
      return null;
    }
    throw error;
  }
};

/**
 * Get database connection status
 */
export const getDatabaseStatus = async (): Promise<DatabaseStatus> => {
  const response = await client.get<DatabaseStatus>('/databases/status');
  return response.data;
};

/**
 * Validate if a path is a valid RocksDB database
 */
export const validatePath = async (path: string): Promise<ValidatePathResponse> => {
  const request: ValidatePathRequest = { path };
  const response = await client.post<{ data: ValidatePathResponse }>('/databases/validate', request);
  return response.data.data;
};
