import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Типы для маркеров
export interface HexMarkers {
  flight_path_search?: number;
  air_attack?: number;
  [key: string]: number | undefined; // для расширяемости
}

// Расширенный интерфейс для ответа API факторов поиска
export interface SearchFactorsResponse {
  hex_factors: Record<string, number>; // hexId -> factors
  hex_markers: Record<string, HexMarkers>; // hexId -> маркеры
  error?: string;
}

// Старый интерфейс для обратной совместимости (deprecated)
export interface LegacySearchFactorsResponse {
  success: boolean;
  data?: {
    hex_factors: Record<string, number>;
    hex_markers?: Record<string, HexMarkers>;
  };
  error?: string;
}

// API клиент для работы с поиском
export const searchAPI = {
  // Получить факторы поиска для указанных гексов (возвращает факторы и маркеры)
  getSearchFactors: async (
    gameId: string,
    hexIds: string[],
    playerSide: 'german' | 'allied',
    token: string
  ): Promise<SearchFactorsResponse> => {
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
      
      if (response.data.success && response.data.data) {
        return {
          hex_factors: response.data.data.hex_factors || {},
          hex_markers: response.data.data.hex_markers || {},
        };
      }
      
      return { hex_factors: {}, hex_markers: {} };
    } catch (error: any) {
      console.error('Error fetching search factors:', error);
      return { hex_factors: {}, hex_markers: {}, error: error.response?.data?.error || 'Failed to fetch search factors' };
    }
  },

  // Универсальные методы для работы с маркерами
  
  // Добавить маркер указанного типа в гекс
  addHexMarker: async (
    gameId: string,
    hexId: string,
    markerType: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/hex-markers`,
        { hex_id: hexId, marker_type: markerType },
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
      console.error('Error adding hex marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to add hex marker'
      };
    }
  },

  // Удалить маркер указанного типа из гекса
  removeHexMarker: async (
    gameId: string,
    hexId: string,
    markerType: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.delete(
        `${API_BASE_URL}/api/games/${gameId}/hex-markers/${hexId}?type=${markerType}`,
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
      console.error('Error removing hex marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to remove hex marker'
      };
    }
  },

  // Получить все маркеры указанного типа для текущего игрока
  getHexMarkers: async (
    gameId: string,
    markerType: string,
    token: string
  ): Promise<string[]> => {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/api/games/${gameId}/hex-markers?type=${markerType}`,
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
      console.error('Error fetching hex markers:', error);
      return [];
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

