import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Интерфейс для конфигурации корабля (соответствует backend ships.json)
export interface ShipConfig {
  id: string;
  name: string;
  type: string;
  side: 'german' | 'allied';
  maxFuel: number;
  baseEvasion: number;
  radarLevel: number;
  hullBoxes: number;
  basePrimaryArmamentBow: number;
  basePrimaryArmamentStern: number;
  baseSecondaryArmament: number;
  maxTorpedos: number;
  speedType: 'F' | 'M' | 'S' | 'VS';
  setupHex?: string;
  notes?: string;
  specialRules?: Array<{
    type: string;
    description: string;
    isActive: boolean;
  }>;
}

// Интерфейс для ответа API
export interface ShipsResponse {
  success: boolean;
  data: ShipConfig[];
  error?: string;
}

export interface ShipResponse {
  success: boolean;
  data: ShipConfig;
  error?: string;
}

// API клиент для работы с конфигурацией кораблей
export const shipsAPI = {
  /**
   * Получить все корабли
   */
  getAllShips: async (): Promise<ShipConfig[]> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/ships/all`);
      if (response.data.success) {
        return response.data.data || response.data;
      }
      console.error('Failed to fetch all ships:', response.data.error);
      return [];
    } catch (error: any) {
      console.error('Error fetching all ships:', error);
      return [];
    }
  },

  /**
   * Получить корабли по стороне
   */
  getShipsBySide: async (side: 'german' | 'allied'): Promise<ShipConfig[]> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/ships/side/${side}`);
      if (response.data.success) {
        return response.data.data || response.data;
      }
      console.error('Failed to fetch ships by side:', response.data.error);
      return [];
    } catch (error: any) {
      console.error('Error fetching ships by side:', error);
      return [];
    }
  },

  /**
   * Получить корабли по типу
   */
  getShipsByType: async (type: string): Promise<ShipConfig[]> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/ships/type/${type}`);
      if (response.data.success) {
        return response.data.data || response.data;
      }
      console.error('Failed to fetch ships by type:', response.data.error);
      return [];
    } catch (error: any) {
      console.error('Error fetching ships by type:', error);
      return [];
    }
  },

  /**
   * Получить конфигурацию конкретного корабля по ID
   */
  getShipConfig: async (shipId: string): Promise<ShipConfig | null> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/ships/config/${shipId}`);
      if (response.data.success) {
        return response.data.data || response.data;
      }
      console.error('Failed to fetch ship config:', response.data.error);
      return null;
    } catch (error: any) {
      console.error('Error fetching ship config:', error);
      return null;
    }
  },

  /**
   * Получить конфигурацию корабля по типу и стороне
   * Полезно когда нужно найти конфигурацию для юнита
   */
  getShipConfigByTypeAndSide: async (
    type: string,
    side: 'german' | 'allied'
  ): Promise<ShipConfig | null> => {
    try {
      const ships = await shipsAPI.getShipsByType(type);
      const ship = ships.find(s => s.side === side);
      return ship || null;
    } catch (error: any) {
      console.error('Error fetching ship config by type and side:', error);
      return null;
    }
  },

  /**
   * Получить статистику конфигурации
   */
  getConfigStats: async (): Promise<any> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/ships/stats`);
      if (response.data.success) {
        return response.data.data || response.data;
      }
      console.error('Failed to fetch config stats:', response.data.error);
      return null;
    } catch (error: any) {
      console.error('Error fetching config stats:', error);
      return null;
    }
  },
};

export default shipsAPI;
