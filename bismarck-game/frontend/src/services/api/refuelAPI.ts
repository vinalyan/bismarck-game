import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface RefuelAllRequest {
  game_id: string;
  fuel_amount: number;
}

export interface RefuelAllResponse {
  success: boolean;
  data: {
    message: string;
    refueled_count: number;
    total_units: number;
    fuel_amount: number;
  };
}

export const refuelAPI = {
  refuelAll: async (request: RefuelAllRequest): Promise<RefuelAllResponse> => {
    const response = await axios.post(`${API_URL}/api/refuel/all`, request);
    return response.data;
  }
};

