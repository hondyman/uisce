
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

export interface ClientConfig {
  baseURL: string;
  token?: string;
}


export interface ListPositionsParams {
  
  accountId: string;
  
  ticker: string;
  
}

export interface ListPositionsResponse {
  [key: string]: any;
}

export interface GetAccountParams {
  
  id: string;
  
}

export interface GetAccountResponse {
  [key: string]: any;
}


export class SemanticClient {
  private client: AxiosInstance;

  constructor(config: ClientConfig) {
    this.client = axios.create({
      baseURL: config.baseURL,
      headers: config.token ? { Authorization: "Bearer " + config.token } : {},
    });
  }

  
  /**
   * 
   */
  async listPositions(params: ListPositionsParams): Promise<ListPositionsResponse[]> {
    const resp = await this.client.get('/api/positions', { params });
    return resp.data;
  }
  
  /**
   * 
   */
  async getAccount(params: GetAccountParams): Promise<GetAccountResponse[]> {
    const resp = await this.client.get('/api/accounts/{id}', { params });
    return resp.data;
  }
  
}
