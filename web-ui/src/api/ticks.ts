import axios from 'axios';

const API_BASE = '/api/v1/tools/ticks';

export interface DateTimeToTicksRequest {
  datetime: string;
}

export interface DateTimeToTicksResponse {
  datetime: string;
  ticks: string; // Changed to string to preserve precision
}

export interface TicksToDateTimeRequest {
  ticks: string; // Changed to string to preserve precision
}

export interface TicksToDateTimeResponse {
  ticks: string; // Changed to string to preserve precision
  datetime: string;
}

export const ticksAPI = {
  /**
   * Convert a datetime string to .NET ticks
   */
  async convertDateTimeToTicks(datetime: string): Promise<DateTimeToTicksResponse> {
    const response = await axios.post(`${API_BASE}/from-datetime`, { datetime });
    return response.data.data;
  },

  /**
   * Convert .NET ticks to a datetime string
   */
  async convertTicksToDateTime(ticks: string): Promise<TicksToDateTimeResponse> {
    const response = await axios.post(`${API_BASE}/to-datetime`, { ticks });
    return response.data.data;
  },
};

