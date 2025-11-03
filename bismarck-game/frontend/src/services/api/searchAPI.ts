import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Интерфейс для ответа API факторов поиска
export interface SearchFactorsResponse {
  success: boolean;
  data?: {
    hex_factors: Record<string, number>; // hexId -> factors
  };
  error?: string;
}

// API клиент для работы с поиском
export const searchAPI = {
  // Получить факторы поиска для указанных гексов
  getSearchFactors: async (
    gameId: string,
    hexIds: string[],
    playerSide: 'german' | 'allied',
    token: string
  ): Promise<Record<string, number>> => {
    try {
      const hexIdsParam = hexIds.join(',');
      const response = await axios.get(
        `${API_BASE_URL}/api/games/${gameId}/search/factors?hex_ids=${hexIdsParam}&player_side=${playerSide}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      
      if (response.data.success && response.data.data?.hex_factors) {
        return response.data.data.hex_factors;
      }
      
      return {};
    } catch (error: any) {
      console.error('Error fetching search factors:', error);
      return {};
    }
  },

  // Добавить маркер пути полета поиска в гекс
  addFlightPathSearchMarker: async (
    gameId: string,
    hexId: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/flight-path-search/markers`,
        { hex_id: hexId },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      
      if (response.data.success) {
        return { success: true };
      }
      
      return { success: false, error: response.data.error || 'Failed to add marker' };
    } catch (error: any) {
      console.error('Error adding flight path search marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to add flight path search marker'
      };
    }
  },

  // Удалить маркер пути полета поиска из гекса
  removeFlightPathSearchMarker: async (
    gameId: string,
    hexId: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.delete(
        `${API_BASE_URL}/api/games/${gameId}/flight-path-search/markers/${hexId}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      
      if (response.data.success) {
        return { success: true };
      }
      
      return { success: false, error: response.data.error || 'Failed to remove marker' };
    } catch (error: any) {
      console.error('Error removing flight path search marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to remove flight path search marker'
      };
    }
  },

  // Получить все маркеры пути полета поиска для текущего игрока
  getFlightPathSearchMarkers: async (
    gameId: string,
    token: string
  ): Promise<string[]> => {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/api/games/${gameId}/flight-path-search/markers`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      
      if (response.data.success && response.data.data?.markers) {
        return response.data.data.markers;
      }
      
      return [];
    } catch (error: any) {
      console.error('Error fetching flight path search markers:', error);
      return [];
    }
  },
};

export default searchAPI;

