// API Response types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// Database types
export interface ColumnFamily {
  name: string;
}

export interface DatabaseInfo {
  column_families: string[];
  count: number;
}

// Data types
export interface KeyValuePair {
  key: string;
  value: string;
  key_is_binary?: boolean;
  value_is_binary?: boolean;
}

export interface ScanOptions {
  start_key?: string;
  end_key?: string;
  limit?: number;
  reverse?: boolean;
  keys_only?: boolean;
  after?: string;
}

export interface ScanResult {
  cf: string;
  results: Record<string, string>; // Deprecated, use results_v2
  results_v2?: KeyValuePair[]; // New format with binary support
  count: number;
  has_more: boolean;
  next_cursor: string;
}

export interface PrefixScanRequest {
  prefix: string;
  limit?: number;
}

export interface SearchRequest {
  key_pattern?: string;
  value_pattern?: string;
  use_regex?: boolean;
  case_sensitive?: boolean;
  keys_only?: boolean;
  limit?: number;
  after?: string;
}

export interface SearchResult {
  key: string;
  value: string;
  key_is_binary?: boolean;
  value_is_binary?: boolean;
  matched_fields: string[];
}

export interface SearchResponse {
  cf: string;
  results: SearchResult[];
  count: number;
  total: number;
  has_more: boolean;
  next_cursor: string;
  query_time: string;
}

export interface Stats {
  name: string;
  key_count: number;
  total_key_size: number;
  total_value_size: number;
  average_key_size: number;
  average_value_size: number;
  data_type_distribution: Record<string, number>;
  key_length_distribution?: Record<string, number>;
  value_length_distribution?: Record<string, number>;
  common_prefixes?: Record<string, number>;
  sample_keys: string[];
}

export interface DatabaseStats {
  column_families: Stats[];
  total_key_count: number;
  total_size: number;
  column_family_count: number;
}

// Database Management types
export interface DatabaseConnectionInfo {
  path: string;
  connected_at: string;
  read_only: boolean;
  column_families: string[];
  column_family_count: number;
}

export interface AvailableDatabase {
  path: string;
  name: string;
  is_valid: boolean;
  error?: string;
  column_families?: string[];
}

export interface DatabaseListResponse {
  databases: AvailableDatabase[];
  mount_points: string[];
}

export interface ConnectRequest {
  path: string;
  read_only?: boolean;
}

export interface ConnectResponse {
  message: string;
  data?: DatabaseConnectionInfo;
}

export interface DatabaseStatus {
  connected: boolean;
  database?: DatabaseConnectionInfo;
  error?: string;
}

export interface ValidatePathRequest {
  path: string;
}

export interface ValidatePathResponse {
  valid: boolean;
  error?: string;
  column_families?: string[];
}
