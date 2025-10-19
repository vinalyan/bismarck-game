// Типы для системы фаз игры

export type GamePhase = 
  | 'setup'
  | 'visibility'
  | 'pursuit'
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
}

export interface PhaseConfig {
  phase: GamePhase;
  name: string;
  description: string;
  duration: number; // в секундах, 0 = без ограничений
  skip_on_turn_1: boolean; // пропускать в первом ходу
  required: boolean; // обязательная фаза
}

// Конфигурации всех фаз
export const PHASE_CONFIGS: Record<GamePhase, PhaseConfig> = {
  setup: {
    phase: 'setup',
    name: 'Подготовка',
    description: 'Расстановка юнитов на карте. Немецкий игрок расставляет танкеры.',
    duration: 0,
    skip_on_turn_1: false,
    required: true,
  },
  visibility: {
    phase: 'visibility',
    name: 'Фаза видимости',
    description: 'Определение погоды и уровня видимости.',
    duration: 60,
    skip_on_turn_1: true,
    required: true,
  },
  pursuit: {
    phase: 'pursuit',
    name: 'Фаза преследования',
    description: 'Попытки преследования обнаруженных кораблей.',
    duration: 120,
    skip_on_turn_1: true,
    required: true,
  },
  movement: {
    phase: 'movement',
    name: 'Фаза движения',
    description: 'Движение морских и воздушных юнитов.',
    duration: 300,
    skip_on_turn_1: false,
    required: true,
  },
  search: {
    phase: 'search',
    name: 'Фаза поиска',
    description: 'Поиск и обнаружение юнитов противника.',
    duration: 180,
    skip_on_turn_1: false,
    required: true,
  },
  air_attack: {
    phase: 'air_attack',
    name: 'Фаза воздушного боя',
    description: 'Воздушные атаки и бои.',
    duration: 120,
    skip_on_turn_1: false,
    required: true,
  },
  naval_combat: {
    phase: 'naval_combat',
    name: 'Фаза морского боя',
    description: 'Морские сражения между кораблями.',
    duration: 600,
    skip_on_turn_1: false,
    required: true,
  },
  chance: {
    phase: 'chance',
    name: 'Фаза случайных событий',
    description: 'Случайные события: контакт с подлодкой, охота на конвои.',
    duration: 60,
    skip_on_turn_1: false,
    required: true,
  },
  admin: {
    phase: 'admin',
    name: 'Админская фаза',
    description: 'Административные действия: подсчет очков, проверка условий победы.',
    duration: 30,
    skip_on_turn_1: false,
    required: true,
  },
};

// Последовательность фаз для хода
export const getPhaseSequence = (turn: number): GamePhase[] => {
  const phases: GamePhase[] = [
    'setup',
    'visibility',
    'pursuit',
    'movement',
    'search',
    'air_attack',
    'naval_combat',
    'chance',
    'admin',
  ];
  
  // В первом ходу пропускаем фазы видимости и преследования
  if (turn === 1) {
    return phases.filter(phase => phase !== 'visibility' && phase !== 'pursuit');
  }
  
  return phases;
};

// Получить конфигурацию фазы
export const getPhaseConfig = (phase: GamePhase): PhaseConfig => {
  return PHASE_CONFIGS[phase];
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
