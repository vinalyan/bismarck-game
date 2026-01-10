import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Интерфейс для маркеров воздушной атаки
export interface AirAttackMarkers {
  [hexId: string]: number; // hexId -> количество маркеров
}

// Интерфейс для информации о цели воздушной атаки
export interface AirAttackTarget {
  unit_id?: string;
  unit_name?: string;
  class: string;
  type: string;
  task_force_id?: string;
  task_force_name?: string;
  visibility: string;
  current_hull: number;
  max_hull: number;
}

// Интерфейс для ответа с целями
export interface AirAttackTargetsResponse {
  hex_id: string;
  targets: AirAttackTarget[];
}

// Интерфейс для результата выполнения атаки
export interface AirAttackExecuteResponse {
  message: string;
  hex_id: string;
  target_id: string;
  target_name?: string;
  new_hull: number;
  sunk: boolean;
}

// API клиент для работы с воздушной атакой
export const airAttackAPI = {
  // Добавить маркер воздушной атаки в гекс
  addMarker: async (
    gameId: string,
    hexId: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/air-attack/marker`,
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

      return { success: false, error: response.data.error || 'Failed to add air attack marker' };
    } catch (error: any) {
      console.error('Error adding air attack marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to add air attack marker'
      };
    }
  },

  // Удалить маркер воздушной атаки из гекса
  removeMarker: async (
    gameId: string,
    hexId: string,
    token: string
  ): Promise<{ success: boolean; error?: string }> => {
    try {
      const response = await axios.delete(
        `${API_BASE_URL}/api/games/${gameId}/air-attack/marker`,
        {
          data: { hex_id: hexId },
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data.success) {
        return { success: true };
      }

      return { success: false, error: response.data.error || 'Failed to remove air attack marker' };
    } catch (error: any) {
      console.error('Error removing air attack marker:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to remove air attack marker'
      };
    }
  },

  // Получить все маркеры воздушной атаки для текущего игрока
  getMarkers: async (
    gameId: string,
    token: string
  ): Promise<AirAttackMarkers> => {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/api/games/${gameId}/air-attack/markers`,
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

      return {};
    } catch (error: any) {
      console.error('Error fetching air attack markers:', error);
      return {};
    }
  },

  // Получить список целей (вражеских кораблей) в гексе для воздушной атаки
  getTargets: async (
    gameId: string,
    hexId: string,
    token: string
  ): Promise<AirAttackTargetsResponse | null> => {
    try {
      const response = await axios.get(
        `${API_BASE_URL}/api/games/${gameId}/air-attack/targets?hex_id=${hexId}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data.success && response.data.data) {
        return response.data.data as AirAttackTargetsResponse;
      }

      return null;
    } catch (error: any) {
      console.error('Error fetching air attack targets:', error);
      return null;
    }
  },

  // Выполнить воздушную атаку на цель
  executeAttack: async (
    gameId: string,
    hexId: string,
    targetId: string,
    targetClass?: string,
    token?: string
  ): Promise<{ success: boolean; data?: AirAttackExecuteResponse; error?: string }> => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'airAttackAPI.ts:171',
        message: 'executeAttack API call started',
        data: { gameId, hexId, targetId, targetClass, url: `${API_BASE_URL}/api/games/${gameId}/air-attack/execute` },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H5'
      })
    }).catch(() => {});
    // #endregion
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/air-attack/execute`,
        {
          hex_id: hexId,
          target_id: targetId,
          target_class: targetClass
        },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );

      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'airAttackAPI.ts:196',
          message: 'executeAttack API response received',
          data: { gameId, hexId, targetId, success: response.data.success, status: response.status, data: response.data.data, error: response.data.error },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H5'
        })
      }).catch(() => {});
      // #endregion

      if (response.data.success) {
        return {
          success: true,
          data: response.data.data as AirAttackExecuteResponse
        };
      }

      return {
        success: false,
        error: response.data.error || 'Failed to execute air attack'
      };
    } catch (error: any) {
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'airAttackAPI.ts:216',
          message: 'executeAttack API error',
          data: { gameId, hexId, targetId, error: error.message || String(error), responseError: error.response?.data?.error, status: error.response?.status },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H5'
        })
      }).catch(() => {});
      // #endregion
      console.error('Error executing air attack:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to execute air attack'
      };
    }
  },
};

export default airAttackAPI;
