import axios from 'axios';
import { GameTurn, PhaseRecord } from '../../types/phaseTypes';

// Экспортируем GameTurn для использования в других файлах
export type { GameTurn } from '../../types/phaseTypes';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Создаем экземпляр axios с базовой конфигурацией
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Интерфейсы для запросов
export interface StartPhaseRequest {
  game_id: string;
  turn: number;
  phase: string;
}

export interface CompletePhaseRequest {
  game_id: string;
  turn: number;
  phase: string;
}

export interface NextPhaseRequest {
  game_id: string;
}

export interface StartTurnRequest {
  game_id: string;
}

// API функции для управления фазами
export const phaseAPI = {
  // Получить текущую фазу игры
  getCurrentPhase: async (gameId: string): Promise<GameTurn | null> => {
    try {
      const response = await apiClient.get(`/api/phases/current?game_id=${gameId}`);
      console.log('📋 phaseAPI.getCurrentPhase response:', JSON.stringify(response.data, null, 2));
      
      // Обрабатываем вложенную структуру ответа
      let turnData = response.data;
      if (turnData.data) {
        turnData = turnData.data;
      }
      // Если data.data существует, используем его
      if (turnData.data) {
        turnData = turnData.data;
      }
      
      console.log('📋 Extracted turn data:', turnData);
      return turnData;
    } catch (error: any) {
      if (error.response?.status === 404) {
        // Нет активного хода
        return null;
      }
      throw error;
    }
  },

  // Получить записи о фазах для хода
  getPhaseRecords: async (gameId: string, turn: number): Promise<PhaseRecord[]> => {
    const response = await apiClient.get(`/api/phases/records?game_id=${gameId}&turn=${turn}`);
    return response.data.data;
  },

  // Начать фазу
  startPhase: async (request: StartPhaseRequest): Promise<void> => {
    await apiClient.post('/api/phases/start', request);
  },

  // Завершить фазу
  completePhase: async (request: CompletePhaseRequest): Promise<void> => {
    await apiClient.post('/api/phases/complete', request);
  },

  // Перейти к следующей фазе
  nextPhase: async (request: NextPhaseRequest): Promise<void> => {
    await apiClient.post('/api/phases/next', request);
  },

  // Начать новый ход
  startTurn: async (request: StartTurnRequest): Promise<GameTurn> => {
    const response = await apiClient.post('/api/phases/turn/start', request);
    return response.data.data;
  },

};

// Добавляем интерцептор для автоматического добавления токена авторизации
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Добавляем интерцептор для обработки ошибок
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Токен истек или недействителен
      localStorage.removeItem('authToken');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
