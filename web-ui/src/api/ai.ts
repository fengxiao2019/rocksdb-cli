const API_BASE_URL = '/api/v1';

export interface AIQueryRequest {
  query: string;
}

export interface AIQueryResponse {
  success: boolean;
  data?: any;
  error?: string;
  error_type?: string;
  explanation?: string;
  execution_time?: string;
  tools_used?: string[];
  intent_detected?: string;
}

export const aiAPI = {
  // Check if AI is enabled
  async checkHealth(): Promise<{ ai_enabled: boolean }> {
    const response = await fetch(`${API_BASE_URL}/health`);
    return response.json();
  },

  // Send query to AI assistant
  async query(queryText: string): Promise<AIQueryResponse> {
    const response = await fetch(`${API_BASE_URL}/ai/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ query: queryText }),
    });
    return response.json();
  },

  // Get AI capabilities
  async getCapabilities(): Promise<{ capabilities: string[] }> {
    const response = await fetch(`${API_BASE_URL}/ai/capabilities`);
    return response.json();
  },
};
