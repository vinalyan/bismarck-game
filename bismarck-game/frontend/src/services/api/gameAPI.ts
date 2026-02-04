// API клиент для связи с бекендом

import axios, { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig, AxiosError } from 'axios';
import {
  APIResponse,
  RegisterRequest,
  LoginRequest,
  LoginResponse,
  User,
  Game,
  GameResponse,
  CreateGameRequest,
  JoinGameRequest,
  SurrenderGameRequest,
  UpdateProfileRequest,
  ChangePasswordRequest,
  TaskForce,
  TaskForceDetails,
  ViewType,
  ScenarioMetadata
} from '../../types/gameTypes';

// Базовый URL API
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

// Создаем экземпляр axios
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptor для добавления токена авторизации
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

// Интерцептор для обработки ответов
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    return response;
  },
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Токен истек или недействителен
      localStorage.removeItem('authToken');
      localStorage.removeItem('user');
      // Используем store для правильной навигации в SPA
      // Импортируем store динамически, чтобы избежать циклических зависимостей
      import('../../stores/gameStore').then(({ useGameStore }) => {
        const store = useGameStore.getState();
        store.setAuthenticated(false);
        store.setUser(null);
        store.setAuthToken(null);
        store.setCurrentView(ViewType.Login);
      });
    }
    return Promise.reject(error);
  }
);

// API методы для аутентификации
export const authAPI = {
  // Регистрация
  register: async (data: RegisterRequest): Promise<APIResponse<User>> => {
    const response = await apiClient.post('/auth/register', data);
    return response.data;
  },

  // Логин
  login: async (data: LoginRequest): Promise<APIResponse<LoginResponse>> => {
    const response = await apiClient.post('/auth/login', data);
    return response.data;
  },

  // Валидация токена
  validateToken: async (): Promise<APIResponse<User>> => {
    const response = await apiClient.get('/auth/validate');
    return response.data;
  },

  // Логаут
  logout: async (): Promise<APIResponse> => {
    const response = await apiClient.post('/auth/logout');
    return response.data;
  },

  // Получение профиля
  getProfile: async (): Promise<APIResponse<User>> => {
    const response = await apiClient.get('/auth/profile');
    return response.data;
  },

  // Обновление профиля
  updateProfile: async (data: UpdateProfileRequest): Promise<APIResponse<User>> => {
    const response = await apiClient.put('/auth/profile', data);
    return response.data;
  },

  // Смена пароля
  changePassword: async (data: ChangePasswordRequest): Promise<APIResponse> => {
    const response = await apiClient.post('/auth/change-password', data);
    return response.data;
  },
};

// API методы для игр
export const gameAPI = {
  // Список доступных сценариев (начальные условия игры)
  getScenarios: async (): Promise<{ scenarios: ScenarioMetadata[] }> => {
    const response = await apiClient.get<{ scenarios: ScenarioMetadata[] }>('/games/scenarios');
    return response.data;
  },

  // Создание игры
  createGame: async (data: CreateGameRequest): Promise<APIResponse<Game>> => {
    const response = await apiClient.post('/games', data);
    return response.data;
  },

  // Получение списка игр
  getGames: async (params?: {
    page?: number;
    perPage?: number;
    status?: string;
  }): Promise<APIResponse<GameResponse[]>> => {
    const response = await apiClient.get('/games', { params });
    return response.data;
  },

  // Получение игры по ID
  getGame: async (gameId: string): Promise<APIResponse<GameResponse>> => {
    const response = await apiClient.get(`/games/${gameId}`);
    return response.data;
  },

  // Присоединение к игре
  joinGame: async (data: JoinGameRequest): Promise<APIResponse<GameResponse>> => {
    const response = await apiClient.post(`/games/${data.gameId}/join`, {
      side: data.side,
      password: data.password || ''
    });
    return response.data;
  },

  // Сдача в игре
  surrenderGame: async (data: SurrenderGameRequest): Promise<APIResponse> => {
    const response = await apiClient.post(`/games/${data.gameId}/surrender`, {
      reason: data.reason
    });
    return response.data;
  },

  // Удаление игры
  deleteGame: async (gameId: string): Promise<APIResponse> => {
    const response = await apiClient.delete(`/games/${gameId}`);
    return response.data;
  },

  // Task Force API endpoints
  getTaskForces: async (gameId: string): Promise<APIResponse<TaskForce[]>> => {
    const response = await apiClient.get(`/games/${gameId}/task-forces`);
    return response.data;
  },

  getTaskForce: async (gameId: string, taskForceId: string): Promise<APIResponse<TaskForceDetails>> => {
    const response = await apiClient.get(`/games/${gameId}/task-forces/${taskForceId}`);
    return response.data;
  },

  createTaskForce: async (gameId: string, data: {
    name?: string;
    unitIds: string[];
    formation: string;
    nationality?: string; // Добавляем nationality для правильного именования
    existingTaskForces?: any[]; // Передаем существующие TF для правильной нумерации
  }): Promise<APIResponse<TaskForce>> => {
    // Генерируем правильное имя по правилам: TF-1, TF-2... для союзников, KG-1, KG-2... для немцев
    const generateTFName = (nationality: string, existingTFs: any[] = []) => {
      const prefix = nationality === 'german' ? 'KG' : 'TF';
      const existingNumbers = existingTFs
        .filter(tf => tf.name && tf.name.startsWith(prefix))
        .map(tf => {
          const match = tf.name.match(new RegExp(`${prefix}-(\\d+)`));
          return match ? parseInt(match[1]) : 0;
        })
        .filter(num => !isNaN(num));
      
      let nextNumber = 1;
      while (existingNumbers.includes(nextNumber)) {
        nextNumber++;
      }
      
      return `${prefix}-${nextNumber}`;
    };

    const requestData = {
      name: data.name || generateTFName(data.nationality || 'allied', data.existingTaskForces || []),
      unit_ids: data.unitIds,
      formation: data.formation,
    };
    console.log('📡 Sending createTaskForce request:', requestData);
    const response = await apiClient.post(`/games/${gameId}/task-forces`, requestData);
    return response.data;
  },

  addUnitToTaskForce: async (gameId: string, data: {
    taskForceId: string;
    unitId: string;
  }): Promise<APIResponse> => {
    const response = await apiClient.post(`/games/${gameId}/task-forces/add-unit`, {
      task_force_id: data.taskForceId,
      unit_id: data.unitId,
    });
    return response.data;
  },

  removeUnitFromTaskForce: async (gameId: string, data: {
    taskForceId: string;
    unitId: string;
  }): Promise<APIResponse> => {
    const response = await apiClient.post(`/games/${gameId}/task-forces/remove-unit`, {
      task_force_id: data.taskForceId,
      unit_id: data.unitId,
    });
    return response.data;
  },
};

// API методы для системных функций
export const systemAPI = {
  // Проверка здоровья сервера
  healthCheck: async (): Promise<APIResponse> => {
    const response = await apiClient.get('/health');
    return response.data;
  },
};

// Экспорт основного клиента для кастомных запросов
export default apiClient;
