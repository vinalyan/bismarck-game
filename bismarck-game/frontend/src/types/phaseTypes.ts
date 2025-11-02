// Типы для системы фаз игры

export type GamePhase = 
  | 'setup'
  | 'visibility'
  | 'shadow'  // было: 'pursuit'
  | 'movement'
  | 'search'
  | 'air_attack'
  | 'naval_combat'
  | 'chance'
  | 'admin';

export type PhaseStatus = 
  | 'pending'
  | 'active'
  | 'completed'
  | 'skipped';

export interface PhaseRecord {
  phase: GamePhase;
  turn: number;
  status: PhaseStatus;
  start_time?: string;
  end_time?: string;
  duration: number;
  data: string;
}

export interface GameTurn {
  id: string;
  game_id: string;
  turn_number: number;
  current_phase: GamePhase;
  status: string;
  start_time: string;
  end_time?: string;
  created_at: string;
  updated_at: string;
  visibility_level?: number;
  is_fog?: boolean;
  weather_track?: number;
}

// Тип для ответа API, который может содержать обернутые данные
export interface GameTurnResponse {
  data?: GameTurn;
  success?: boolean;
}

// Последовательность фаз (фиксированная)
export const PHASE_SEQUENCE_TURN_1: GamePhase[] = [
  'movement',
  'search',
  'air_attack',
  'naval_combat',
  'chance',
  'admin',
];

export const PHASE_SEQUENCE_DEFAULT: GamePhase[] = [
  'visibility',
  'shadow',
  'movement',
  'search',
  'air_attack',
  'naval_combat',
  'chance',
  'admin',
];

export const getPhaseSequence = (turn: number): GamePhase[] => {
  return turn === 1 ? PHASE_SEQUENCE_TURN_1 : PHASE_SEQUENCE_DEFAULT;
};

// Названия фаз (UI-константы)
export const PHASE_NAMES: Record<GamePhase, string> = {
  setup: 'Подготовка',
  visibility: 'Фаза видимости',
  shadow: 'Фаза слежения',
  movement: 'Фаза движения',
  search: 'Фаза поиска',
  air_attack: 'Фаза воздушного боя',
  naval_combat: 'Фаза морского боя',
  chance: 'Фаза случайных событий',
  admin: 'Админская фаза',
};

export const PHASE_DESCRIPTIONS: Record<GamePhase, string> = {
  setup: 'Расстановка юнитов на карте. Немецкий игрок расставляет танкеры.',
  visibility: 'Определение погоды и уровня видимости.',
  shadow: 'Попытки слежения за обнаруженными кораблями.',
  movement: 'Движение морских и воздушных юнитов.',
  search: 'Поиск и обнаружение юнитов противника.',
  air_attack: 'Воздушные атаки и бои.',
  naval_combat: 'Морские сражения между кораблями.',
  chance: 'Случайные события: контакт с подлодкой, охота на конвои.',
  admin: 'Административные действия: подсчет очков, проверка условий победы.',
};

// Получить статус фазы в виде строки
export const getPhaseStatusText = (status: PhaseStatus): string => {
  switch (status) {
    case 'pending':
      return 'Ожидает';
    case 'active':
      return 'Активна';
    case 'completed':
      return 'Завершена';
    case 'skipped':
      return 'Пропущена';
    default:
      return 'Неизвестно';
  }
};

// Получить цвет для статуса фазы
export const getPhaseStatusColor = (status: PhaseStatus): string => {
  switch (status) {
    case 'pending':
      return '#6b7280'; // серый
    case 'active':
      return '#3b82f6'; // синий
    case 'completed':
      return '#10b981'; // зеленый
    case 'skipped':
      return '#f59e0b'; // желтый
    default:
      return '#6b7280';
  }
};