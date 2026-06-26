import axios from 'axios';
import { WorkflowDefinition } from '../../../bp-backend/pkg/workflow/workflow';

class BusinessProcessService {
  private baseURL = '/api/v1';

  // Create a new business process
  async createBusinessProcess(process: WorkflowDefinition): Promise<{ id: string }> {
    try {
      const response = await axios.post(`${this.baseURL}/workflow_versions`, process);
      return response.data;
    } catch (error: any) {
      throw new Error(error.response?.data?.message || 'Failed to create business process');
    }
  }
}

export const businessProcessService = new BusinessProcessService();