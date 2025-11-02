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
};

export default searchAPI;

