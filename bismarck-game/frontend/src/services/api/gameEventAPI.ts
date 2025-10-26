import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface GameEvent {
  id: string;
  game_id: string;
  turn: number;
  phase: string;
  event_type: string;
  actor_name: string;
  description: string;
  data: any;
  created_at: string;
}

export interface GameEventsResponse {
  events: GameEvent[];
  count: number;
}

export const gameEventAPI = {
  async getGameEvents(gameId: string, limit?: number): Promise<{
    success: boolean;
    data?: GameEventsResponse;
    error?: string;
  }> {
    try {
      const params = new URLSearchParams({
        game_id: gameId,
      });
      
      if (limit) {
        params.append('limit', limit.toString());
      }

      const response = await axios.get(`${API_BASE_URL}/api/game-events?${params}`);
      return {
        success: true,
        data: response.data.data, // response.data содержит {success: true, data: {...}}
      };
    } catch (error: any) {
      return {
        success: false,
        error: error.response?.data?.message || 'Failed to get game events',
      };
    }
  },
};
