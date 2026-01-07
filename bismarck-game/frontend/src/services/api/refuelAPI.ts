import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Запрос на заправку всех юнитов (устаревший, для обратной совместимости)
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

// Запрос на заправку в порту
export interface RefuelAtPortRequest {
  game_id: string;
  unit_id: string;
}

// Запрос на заправку в море
export interface RefuelAtSeaRequest {
  game_id: string;
  unit_id: string;
  tanker_id: string;
}

// Результат заправки
export interface RefuelResult {
  success: boolean;
  message: string;
  fuel_added: number;
  new_fuel_level: number;
  refuel_type: 'port' | 'sea';
}

// Ответ API для заправки
export interface RefuelResponse {
  success: boolean;
  data?: RefuelResult;
  error?: string;
}

// Ответ API для доступных гексов заправки
export interface AvailableRefuelHexesResponse {
  success: boolean;
  data?: {
    hexes: string[];
  };
  error?: string;
}

// Информация о танкере
export interface TankerInfo {
  id: string;
  name: string;
  position: string;
  fuel: number;
  max_fuel: number;
  tanker_used_this_turn: boolean;
}

// Ответ API для танкеров в гексе
export interface TankersInHexResponse {
  success: boolean;
  data?: {
    tankers: TankerInfo[];
  };
  error?: string;
}

export const refuelAPI = {
  /**
   * Заправка всех юнитов (устаревший метод, для обратной совместимости)
   * @param request Данные для заправки
   */
  refuelAll: async (request: RefuelAllRequest): Promise<RefuelAllResponse> => {
    try {
      const response = await axios.post(`${API_URL}/api/refuel/all`, request);
      return response.data;
    } catch (error: any) {
      console.error('Error refueling all units:', error);
      return {
        success: false,
        data: {
          message: error.response?.data?.error || error.message || 'Unknown error',
          refueled_count: 0,
          total_units: 0,
          fuel_amount: 0
        }
      };
    }
  },

  /**
   * Заправка юнита в порту
   * @param request Данные для заправки
   * @param token Токен авторизации
   */
  refuelAtPort: async (request: RefuelAtPortRequest, token?: string): Promise<RefuelResponse> => {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await axios.post(`${API_URL}/api/refuel/port`, request, { headers });
      return {
        success: true,
        data: response.data.data
      };
    } catch (error: any) {
      console.error('Error refueling at port:', error);
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Unknown error'
      };
    }
  },

  /**
   * Заправка юнита в море от танкера
   * @param request Данные для заправки
   * @param token Токен авторизации
   */
  refuelAtSea: async (request: RefuelAtSeaRequest, token?: string): Promise<RefuelResponse> => {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await axios.post(`${API_URL}/api/refuel/sea`, request, { headers });
      return {
        success: true,
        data: response.data.data
      };
    } catch (error: any) {
      console.error('Error refueling at sea:', error);
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Unknown error'
      };
    }
  },

  /**
   * Получить доступные гексы для заправки юнита
   * @param gameId ID игры
   * @param unitId ID юнита
   * @param token Токен авторизации
   */
  getAvailableRefuelHexes: async (
    gameId: string, 
    unitId: string, 
    token?: string
  ): Promise<AvailableRefuelHexesResponse> => {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await axios.get(
        `${API_URL}/api/refuel/available-hexes/${gameId}/${unitId}`, 
        { headers }
      );
      return {
        success: true,
        data: {
          hexes: response.data.data?.hexes || []
        }
      };
    } catch (error: any) {
      console.error('Error getting available refuel hexes:', error);
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Unknown error'
      };
    }
  },

  /**
   * Получить список танкеров в указанном гексе
   * @param gameId ID игры
   * @param hexId ID гекса
   * @param token Токен авторизации
   */
  getTankersInHex: async (
    gameId: string, 
    hexId: string, 
    token?: string
  ): Promise<TankersInHexResponse> => {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await axios.get(
        `${API_URL}/api/refuel/tankers/${gameId}/${hexId}`, 
        { headers }
      );
      return {
        success: true,
        data: {
          tankers: response.data.data?.tankers || []
        }
      };
    } catch (error: any) {
      console.error('Error getting tankers in hex:', error);
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Unknown error'
      };
    }
  }
};
