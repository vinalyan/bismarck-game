import axios from 'axios';
import { GameTurn, PhaseRecord, PhaseConfig } from '../../types/phaseTypes';

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
  turn_number: number;
}

// API функции для управления фазами
export const phaseAPI = {
  // Получить текущую фазу игры
  getCurrentPhase: async (gameId: string): Promise<GameTurn> => {
    const response = await apiClient.get(`/api/phases/current?game_id=${gameId}`);
    return response.data.data;
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
  startTurn: async (request: StartTurnRequest): Promise<void> => {
    await apiClient.post('/api/phases/turn/start', request);
  },

  // Получить информацию о фазе
  getPhaseInfo: async (phase: string): Promise<PhaseConfig> => {
    const response = await apiClient.get(`/api/phases/info?phase=${phase}`);
    return response.data.data;
  },

  // Получить информацию о всех фазах
  getAllPhases: async (): Promise<PhaseConfig[]> => {
    const response = await apiClient.get('/api/phases/all');
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
