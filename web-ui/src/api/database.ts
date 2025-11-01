import client, { apiRequest } from './client';
import type {
  DatabaseInfo,
  KeyValuePair,
  ScanOptions,
  ScanResult,
  PrefixScanRequest,
  SearchRequest,
  SearchResponse,
  DatabaseStats,
  Stats,
} from '@/types/api';

// List all column families
export const listColumnFamilies = () => {
  return apiRequest<DatabaseInfo>(client.get('/cf'));
};

// Get value by key
export const getValue = (cf: string, key: string) => {
  return apiRequest<KeyValuePair>(client.get(`/cf/${cf}/get/${encodeURIComponent(key)}`));
};

// Put key-value pair
export const putValue = (cf: string, key: string, value: string) => {
  return apiRequest(client.post(`/cf/${cf}/put`, { key, value }));
};

// Delete key
export const deleteKey = (cf: string, key: string) => {
  return apiRequest(client.delete(`/cf/${cf}/delete/${encodeURIComponent(key)}`));
};

// Get last entry
export const getLastEntry = (cf: string) => {
  return apiRequest<KeyValuePair>(client.get(`/cf/${cf}/last`));
};

// Scan data
export const scanData = (cf: string, options: ScanOptions = {}) => {
  return apiRequest<ScanResult>(client.post(`/cf/${cf}/scan`, options));
};

// Prefix scan
export const prefixScan = (cf: string, request: PrefixScanRequest) => {
  return apiRequest<ScanResult>(client.post(`/cf/${cf}/prefix`, request));
};

// Advanced search
export const searchData = (cf: string, request: SearchRequest) => {
  return apiRequest<SearchResponse>(client.post(`/cf/${cf}/search`, request));
};

// JSON query
export const jsonQuery = (cf: string, field: string, value: string) => {
  return apiRequest(client.post(`/cf/${cf}/jsonquery`, { field, value }));
};

// Get database stats
export const getDatabaseStats = () => {
  return apiRequest<DatabaseStats>(client.get('/stats'));
};

// Get column family stats
export const getColumnFamilyStats = (cf: string) => {
  return apiRequest<{ cf: string; stats: Stats }>(client.get(`/cf/${cf}/stats`));
};

// Health check (doesn't use standard API response format)
export const healthCheck = async () => {
  const response = await client.get('/health');
  return response.data;
};
