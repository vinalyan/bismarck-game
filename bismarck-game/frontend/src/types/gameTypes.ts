// Базовые типы для игры "Погоня за Бисмарком"

// Пользователь
export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  stats: UserStats;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  lastLogin?: string;
}

// Роли пользователей
export enum UserRole {
  Player = 'player',
  Admin = 'admin',
  Moderator = 'moderator'
}

// Стороны игроков
export enum PlayerSide {
  German = 'german',
  Allied = 'allied'
}

// Статистика пользователя
export interface UserStats {
  gamesPlayed: number;
  gamesWon: number;
  gamesLost: number;
  totalScore: number;
  averageScore: number;
  winRate: number;
  favoriteFaction: string;
  totalPlayTime: number;
  lastGameDate?: string;
}

// Игра
export interface Game {
  id: string;
  name: string;
  player1_id: string;
  player2_id?: string;
  current_turn: number;
  current_phase: GamePhase;
  status: GameStatus;
  settings: GameSettings;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

// Фазы игры
export enum GamePhase {
  Waiting = 'waiting',
  Setup = 'setup',
  Movement = 'movement',
  Search = 'search',
  Combat = 'combat',
  End = 'end'
}

// Статус игры
export enum GameStatus {
  Waiting = 'waiting',
  InProgress = 'in_progress',
  Completed = 'completed',
  Cancelled = 'cancelled'
}

// Настройки игры
export interface GameSettings {
  turnDuration: number;
  gameMode: GameMode;
  difficulty: Difficulty;
  weatherEnabled: boolean;
  fogOfWar: boolean;
  randomEvents: boolean;
  victoryConditions: VictoryCondition[];
  // maxPlayers убран - всегда 2 игрока
}

// Режимы игры
export enum GameMode {
  Classic = 'classic',
  Quick = 'quick',
  Campaign = 'campaign'
}

// Сложность
export enum Difficulty {
  Easy = 'easy',
  Normal = 'normal',
  Hard = 'hard',
  Expert = 'expert'
}

// Условия победы
export enum VictoryCondition {
  Operational = 'operational',
  Strategic = 'strategic',
  TimeLimit = 'time_limit'
}

// Ответ API для игры
export interface GameResponse {
  id: string;
  name: string;
  player1_id: string;
  player2_id?: string;
  player1_username?: string;
  player2_username?: string;
  player1_side: PlayerSide;
  player2_side: PlayerSide;
  player1?: User;
  player2?: User;
  current_turn: number;
  current_phase: GamePhase;
  status: GameStatus;
  settings: GameSettings;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

// Запрос на создание игры
export interface CreateGameRequest {
  name: string;
  side: PlayerSide;
  settings: Partial<GameSettings>;
  password?: string;
}

// Запрос на присоединение к игре
export interface JoinGameRequest {
  gameId: string;
  password?: string;
}

// Запрос на сдачу
export interface SurrenderGameRequest {
  gameId: string;
  reason?: string;
}

// API ответ
export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
  meta?: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
  };
}

// Запрос на регистрацию
export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

// Запрос на логин
export interface LoginRequest {
  username: string;
  password: string;
}

// Ответ на логин
export interface LoginResponse {
  user: User;
  token: string;
  expiresAt: string;
}

// Запрос на обновление профиля
export interface UpdateProfileRequest {
  username?: string;
  email?: string;
}

// Запрос на смену пароля
export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

// Состояние UI
export interface UIState {
  isLoading: boolean;
  error: string | null;
  currentView: ViewType;
  selectedGame?: string;
  notifications: Notification[];
}

// Типы представлений
export enum ViewType {
  Login = 'login',
  Register = 'register',
  Lobby = 'lobby',
  Game = 'game',
  Profile = 'profile'
}

// Уведомление
export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
}

// Типы уведомлений
export enum NotificationType {
  Info = 'info',
  Success = 'success',
  Warning = 'warning',
  Error = 'error'
}

// WebSocket сообщения
export interface WSMessage {
  type: WSMessageType;
  data: any;
  timestamp: number;
  gameId?: string;
  userId?: string;
}

// Типы WebSocket сообщений
export enum WSMessageType {
  // Системные
  Ping = 'ping',
  Pong = 'pong',
  Error = 'error',
  
  // Игровые
  GameUpdate = 'game_update',
  PlayerJoined = 'player_joined',
  PlayerLeft = 'player_left',
  GameStarted = 'game_started',
  GameEnded = 'game_ended',
  
  // Чат
  ChatMessage = 'chat_message',
  
  // Действия
  ActionSubmitted = 'action_submitted',
  ActionProcessed = 'action_processed',
  
  // Уведомления
  Notification = 'notification'
}

// Чат сообщение
export interface ChatMessage {
  id: string;
  userId: string;
  username: string;
  message: string;
  timestamp: string;
  gameId?: string;
  type: ChatMessageType;
}

// Типы сообщений чата
export enum ChatMessageType {
  System = 'system',
  Player = 'player',
  Game = 'game'
}
