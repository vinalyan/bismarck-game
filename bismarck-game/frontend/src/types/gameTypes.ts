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
  InProgress = 'active',
  Completed = 'completed',
  Cancelled = 'cancelled'
}

// Настройки игры
export interface GameSettings {
  use_optional_units: boolean;
  enable_crew_exhaustion: boolean;
  victory_conditions: VictoryConfig;
  time_limit_minutes: number;
  private_lobby: boolean;
  password?: string;
  max_turn_time: number;
  allow_spectators: boolean;
  auto_save: boolean;
  difficulty: string;
  // maxPlayers убран - всегда 2 игрока
}

// Конфигурация условий победы
export interface VictoryConfig {
  bismarck_sunk_vp: number;
  bismarck_france_vp: number;
  bismarck_norway_vp: number;
  bismarck_end_game_vp: number;
  bismarck_no_fuel_vp: number;
  ship_vp_values: any; // TODO: определить точную структуру
  convoy_vp: any; // TODO: определить точную структуру
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

// ===== ТИПЫ ЮНИТОВ =====

// Тип юнита
export enum UnitType {
  // Морские юниты (корабли)
  Battleship = 'BB',        // Линейный корабль
  Battlecruiser = 'BC',     // Линейный крейсер
  AircraftCarrier = 'CV',   // Авианосец
  HeavyCruiser = 'CA',      // Тяжелый крейсер
  LightCruiser = 'CL',      // Легкий крейсер
  Destroyer = 'DD',         // Флотилия эсминцев
  CoastGuard = 'CG',        // Береговая охрана
  Tanker = 'TK',            // Танкер
  
  // Воздушные юниты (самолеты)
  CombatAircraft = 'B',     // Боевой самолет
  ReconAircraft = 'R'       // Самолет-разведчик
}

// Класс скорости корабля
export enum SpeedType {
  Fast = 'F',        // Быстрый
  Medium = 'M',      // Средний
  Slow = 'S',        // Медленный
  VerySlow = 'VS'    // Очень медленный
}

// Статус морского юнита
export enum UnitStatus {
  Active = 'active',
  Damaged = 'damaged',
  Sunk = 'sunk',
  Repairing = 'repairing',
  Refueling = 'refueling',
  Hidden = 'hidden'
}

// Статус воздушного юнита
export enum AirUnitStatus {
  Landing = 'landing',     // Посадка
  Refit = 'refit',         // Перевооружение
  Operational = 'operational', // Операционный
  OnRaid = 'on_raid'       // На рейде
}

// Уровень обнаружения
export enum DetectionLevel {
  None = 'none',
  Sighted = 'sighted',
  Shadowed = 'shadowed',
  Lost = 'lost'
}


// Повреждение
export interface Damage {
  type: 'hull' | 'gun' | 'engine' | 'fire';
  severity: number; // 1-3
  location: 'bow' | 'stern' | 'port' | 'starboard' | 'center';
  description: string; // описание
  turnApplied: number; // ход, когда нанесено
  createdAt: string;
}

// Морской юнит
export interface NavalUnit {
  id: string;
  gameId: string;
  name: string;
  type: UnitType;
  class: string;
  owner: string;
  nationality: string;
  position: string; // Hex coordinate
  evasion: number; // Скорость в узлах
  baseEvasion: number;
  speedRating: SpeedType; // F, M, S, VS
  fuel: number;
  maxFuel: number;
  hullBoxes: number;
  currentHull: number;
  
  // Вооружение (простые числовые характеристики)
  primaryArmamentBow: number;    // Основное вооружение (нос) - текущее
  primaryArmamentStern: number;  // Основное вооружение (корма) - текущее
  secondaryArmament: number;     // Вспомогательное вооружение - текущее
  
  // Базовые значения вооружения (неизменяемые)
  basePrimaryArmamentBow: number;    // Базовое основное вооружение (нос)
  basePrimaryArmamentStern: number;  // Базовое основное вооружение (корма)
  baseSecondaryArmament: number;     // Базовое вспомогательное вооружение
  
  torpedoes: number;
  maxTorpedoes: number;
  radarLevel: number; // 0, 1, 2 (RADAR I, RADAR II, RADAR II*)
  status: UnitStatus;
  detectionLevel: DetectionLevel;
  lastKnownPos?: string;
  taskForceId?: string;
  damage: Damage[];
  
  // Поля для тактического боя (используются только во время боя)
  tacticalPosition?: string; // Movement Zone ID
  tacticalFacing?: 'closing' | 'opening' | 'breaking-off';
  tacticalSpeed?: number;
  evasionEffects: number[];
  tacticalDamageTaken: Damage[];
  hasFired: boolean;
  targetAcquired?: string;
  torpedoesUsed: number;
  movementUsed: number;
  
  createdAt: string;
  updatedAt: string;
}

// Воздушный юнит
export interface AirUnit {
  id: string;
  gameId: string;
  type: UnitType; // B (боевой) или R (разведывательный)
  owner: string;
  position: string; // Hex coordinate
  basePosition: string;
  maxSpeed: number; // Максимальная скорость
  endurance: number; // Дальность полета
  status: AirUnitStatus;
  createdAt: string;
  updatedAt: string;
}


// Оперативное соединение (Task Force)
export interface TaskForce {
  id: string;
  gameId: string;
  name: string;
  owner: string;
  position: string; // Hex coordinate
  speed: number;
  units: string[]; // IDs юнитов
  isVisible: boolean;
  createdAt: string;
  updatedAt: string;
}

// Движение юнита
export interface UnitMovement {
  id: string;
  gameId: string;
  unitId: string;
  from: string; // Hex coordinate
  to: string; // Hex coordinate
  path: string[]; // Path coordinates
  speed: number;
  fuelCost: number;
  isShadowed: boolean;
  turn: number;
  phase: GamePhase;
  createdAt: string;
}

// Поиск юнита
export interface UnitSearch {
  id: string;
  gameId: string;
  unitId: string;
  targetHex: string;
  searchType: 'air' | 'naval' | 'radar';
  searchFactors: number;
  result: 'no_contact' | 'contact' | 'detection';
  unitsFound: string[]; // IDs найденных юнитов
  turn: number;
  phase: GamePhase;
  createdAt: string;
}

// ===== API ЗАПРОСЫ И ОТВЕТЫ ДЛЯ ЮНИТОВ =====

// Запрос на движение юнита
export interface MoveUnitRequest {
  unitId: string;
  to: string; // Hex coordinate
  speed: number;
  path?: string[]; // Optional path
}

// Запрос на поиск
export interface SearchRequest {
  unitId: string;
  targetHex: string;
  searchType: 'air' | 'naval' | 'radar';
}

// Запрос на создание Task Force
export interface CreateTaskForceRequest {
  name: string;
  unitIds: string[];
  formation: 'line' | 'diamond' | 'wedge' | 'scattered';
}

// Запрос на добавление юнита в Task Force
export interface AddUnitToTaskForceRequest {
  taskForceId: string;
  unitId: string;
}

// Запрос на удаление юнита из Task Force
export interface RemoveUnitFromTaskForceRequest {
  taskForceId: string;
  unitId: string;
}

// Ответ с информацией о юните
export interface UnitResponse {
  unit: NavalUnit | AirUnit;
  canMove: boolean;
  canSearch: boolean;
  canFire: boolean;
  availableActions: string[];
  movementRange: number;
  searchRange: number;
}

// Ответ с информацией о Task Force
export interface TaskForceResponse {
  taskForce: TaskForce;
  units: (NavalUnit | AirUnit)[];
  effectiveSpeed: number;
  totalSearchFactors: number;
  canForm: boolean;
  canSplit: boolean;
}
