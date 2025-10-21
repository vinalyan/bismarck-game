import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Интерфейс для юнита
export interface GameUnit {
  id: string;
  game_id: string;
  name: string;
  type: string;
  class: string;
  owner: string;
  nationality: string;
  position: string;
  setup_hex: string;
  evasion: number;
  base_evasion: number;
  speed_rating: string;
  fuel: number;
  max_fuel: number;
  hull_boxes: number;
  current_hull: number;
  primary_armament_bow: number;
  primary_armament_stern: number;
  secondary_armament: number;
  base_primary_armament_bow: number;
  base_primary_armament_stern: number;
  base_secondary_armament: number;
  torpedoes: number;
  max_torpedoes: number;
  radar_level: number;
  status: string;
  detection_level: string;
  last_known_pos: string | null;
  task_force_id: string | null;
  damage: any[];
  tactical_position: any;
  tactical_facing: any;
  tactical_speed: any;
  evasion_effects: any;
  tactical_damage_taken: any;
  has_fired: boolean;
  target_acquired: any;
  torpedoes_used: number;
  movement_used: number;
  previous_turn_moved_hexes: number;
  last_move_turn: number;
  no_movement_turns_left: number;
  created_at: string;
  updated_at: string;
}

// Интерфейс для ответа API
export interface UnitsResponse {
  success: boolean;
  data: {
    units: GameUnit[];
  };
  error?: string;
}

// Интерфейс для запроса обновления позиции
export interface UpdatePositionRequest {
  position: string;
  fuel?: number;
  hexesMoved?: number; // Количество гексов, на которое переместился юнит
}

// Интерфейс для ответа обновления позиции
export interface UpdatePositionResponse {
  success: boolean;
  data?: {
    message: string;
    unitId: string;
    position: string;
  };
  error?: string;
}

// API клиент для работы с юнитами
export const unitsAPI = {
  // Получить юниты игры, видимые для текущего игрока
  getGameUnits: async (gameId: string, token: string): Promise<UnitsResponse> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/games/${gameId}/units`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      return response.data;
    } catch (error: any) {
      console.error('Error fetching game units:', error);
      return {
        success: false,
        data: { units: [] },
        error: error.response?.data?.error || 'Failed to fetch game units'
      };
    }
  },

  // Обновить позицию юнита
  updateUnitPosition: async (
    gameId: string, 
    unitId: string, 
    request: UpdatePositionRequest, 
    token: string
  ): Promise<UpdatePositionResponse> => {
    try {
      const response = await axios.put(
        `${API_BASE_URL}/api/games/${gameId}/units/${unitId}/position`,
        request,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error updating unit position:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to update unit position'
      };
    }
  }
};

export default unitsAPI;
