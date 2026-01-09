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
  is_emergency_fuel: boolean;
  emergency_turn: number;
  is_patrolling: boolean;
  is_activated?: boolean;
  available_actions?: string[];
  visibility?: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  created_at: string;
  updated_at: string;
}

// Интерфейс для Task Force
export interface TaskForce {
  id: string;
  name: string;
  nationality: string;
  position: string;
  units: string[];
  speed: number;
  detection_level: string;
  last_move_turn: number;
  is_activated: boolean;
  is_patrolling: boolean;
  available_actions?: string[];
  visibility?: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  created_at: string;
  updated_at: string;
}

// Интерфейс для ответа API
export interface EnemyContact {
  hex_id: string;
  visibility: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  ship_count: number;
  class_summary: string;
  task_force: string;
  task_force_list: string[];
  enemy_nationality: 'german' | 'allied';
  searching_side: 'german' | 'allied';
  turn: number;
  phase: string;
  last_seen_at: string;
}

export interface UnitsResponse {
  success: boolean;
  data: {
    units: GameUnit[];
    task_forces?: TaskForce[];
    enemy_contacts?: EnemyContact[];
    current_turn?: {
      turn: number;
      phase: string;
    };
    events?: any[]; // События из GameModel
    search?: {
      search_hexes?: { [hexId: string]: SearchHexData };
    };
    air_attack?: {
      german?: { [hexId: string]: number };
      allied?: { [hexId: string]: number };
    };
    version?: number; // Версия модели для проверки изменений
    visibility_level?: number;
    is_fog?: boolean;
    weather_track?: number;
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

// Интерфейс для данных поиска гекса
interface SearchHexData {
  factor: number;
  ships: number;
  patrol: number;
  air_search: number;
  intrinsic: number;
}

// Интерфейс для GameModel (из бэкенда)
interface GameModel {
  game_id: string;
  version: number;
  last_updated: string;
  current_turn: {
    turn: number;
    phase: string;
  };
  units: { [key: string]: UnitModel };
  task_forces: { [key: string]: TaskForceModel };
  enemy_contacts: EnemyContactModel[];
  search?: {
    search_hexes?: { [hexId: string]: SearchHexData };
  };
  air_attack?: {
    german?: { [hexId: string]: number };
    allied?: { [hexId: string]: number };
  };
  events: any[];
  intrinsic_search_hexes?: { [key: string]: number };
  visibility_level?: number;
  is_fog?: boolean;
  weather_track?: number;
}

interface UnitModel {
  id: string;
  game_id: string;
  name: string;
  type: string;
  category: 'naval' | 'air';
  owner: string;
  nationality: string;
  position: string;
  status: string;
  visibility?: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  naval_data?: {
    class: string;
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
    detection_level: string;
    last_known_pos: string | null;
    task_force_id: string | null;
    damage: any[];
    previous_turn_moved_hexes: number;
    last_move_turn: number;
    no_movement_turns_left: number;
    is_activated: boolean;
    is_emergency_fuel: boolean;
    emergency_turn: number;
    is_patrolling: boolean;
    available_actions?: string[];
  };
  air_data?: {
    base_position: string;
    max_speed: number;
    endurance: number;
    flight_path_search_hexes: string[];
  };
  created_at: string;
  updated_at: string;
}

interface TaskForceModel {
  id: string;
  game_id: string;
  name: string;
  owner: string;
  nationality: string;
  position: string;
  speed: number;
  units: string[];
  is_visible: boolean;
  visibility?: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  detection_level: string;
  last_move_turn: number;
  is_activated: boolean;
  is_patrolling: boolean;
  available_actions?: string[];
  created_at: string;
  updated_at: string;
}

interface EnemyContactModel {
  hex_id: string;
  visibility: 'unknown' | 'lost' | 'sighted' | 'shadowed';
  ship_count: number;
  class_summary: string;
  task_force: string;
  task_force_list: string[];
  enemy_nationality: 'german' | 'allied';
  searching_side: 'german' | 'allied';
  turn: number;
  phase: string;
  last_seen_at: string;
}

// Функция преобразования UnitModel в GameUnit
function convertUnitModelToGameUnit(unitModel: UnitModel): GameUnit {
  const navalData = unitModel.naval_data;
  if (!navalData) {
    // Для воздушных юнитов создаем минимальный объект
    return {
      id: unitModel.id,
      game_id: unitModel.game_id,
      name: unitModel.name,
      type: unitModel.type,
      class: '',
      owner: unitModel.owner,
      nationality: unitModel.nationality || '',
      position: unitModel.position,
      setup_hex: unitModel.air_data?.base_position || '',
      evasion: 0,
      base_evasion: 0,
      speed_rating: '',
      fuel: 0,
      max_fuel: 0,
      hull_boxes: 0,
      current_hull: 0,
      primary_armament_bow: 0,
      primary_armament_stern: 0,
      secondary_armament: 0,
      base_primary_armament_bow: 0,
      base_primary_armament_stern: 0,
      base_secondary_armament: 0,
      torpedoes: 0,
      max_torpedoes: 0,
      radar_level: 0,
      status: unitModel.status,
      detection_level: '',
      last_known_pos: null,
      task_force_id: null,
      damage: [],
      tactical_position: null,
      tactical_facing: null,
      tactical_speed: null,
      evasion_effects: null,
      tactical_damage_taken: null,
      has_fired: false,
      target_acquired: null,
      torpedoes_used: 0,
      movement_used: 0,
      previous_turn_moved_hexes: 0,
      last_move_turn: 0,
      no_movement_turns_left: 0,
      is_emergency_fuel: false,
      emergency_turn: 0,
      is_patrolling: false,
      is_activated: false,
      available_actions: [],
      visibility: unitModel.visibility,
      created_at: unitModel.created_at,
      updated_at: unitModel.updated_at,
    };
  }

  return {
    id: unitModel.id,
    game_id: unitModel.game_id,
    name: unitModel.name,
    type: unitModel.type,
    class: navalData.class,
    owner: unitModel.owner,
    nationality: unitModel.nationality,
    position: unitModel.position,
    setup_hex: navalData.setup_hex,
    evasion: navalData.evasion,
    base_evasion: navalData.base_evasion,
    speed_rating: navalData.speed_rating,
    fuel: navalData.fuel,
    max_fuel: navalData.max_fuel,
    hull_boxes: navalData.hull_boxes,
    current_hull: navalData.current_hull,
    primary_armament_bow: navalData.primary_armament_bow,
    primary_armament_stern: navalData.primary_armament_stern,
    secondary_armament: navalData.secondary_armament,
    base_primary_armament_bow: navalData.base_primary_armament_bow,
    base_primary_armament_stern: navalData.base_primary_armament_stern,
    base_secondary_armament: navalData.base_secondary_armament,
    torpedoes: navalData.torpedoes,
    max_torpedoes: navalData.max_torpedoes,
    radar_level: navalData.radar_level,
    status: unitModel.status,
    detection_level: navalData.detection_level,
    last_known_pos: navalData.last_known_pos,
    task_force_id: navalData.task_force_id,
    damage: navalData.damage || [],
    tactical_position: null,
    tactical_facing: null,
    tactical_speed: null,
    evasion_effects: null,
    tactical_damage_taken: null,
    has_fired: false,
    target_acquired: null,
    torpedoes_used: 0,
    movement_used: 0,
    previous_turn_moved_hexes: navalData.previous_turn_moved_hexes,
    last_move_turn: navalData.last_move_turn,
    no_movement_turns_left: navalData.no_movement_turns_left,
    is_emergency_fuel: navalData.is_emergency_fuel,
    emergency_turn: navalData.emergency_turn,
    is_patrolling: navalData.is_patrolling,
    is_activated: navalData.is_activated || false,
    available_actions: navalData.available_actions || [],
    visibility: unitModel.visibility,
    created_at: unitModel.created_at,
    updated_at: unitModel.updated_at,
  };
}

// Функция преобразования TaskForceModel в TaskForce
function convertTaskForceModelToTaskForce(tfModel: TaskForceModel): TaskForce {
  return {
    id: tfModel.id,
    name: tfModel.name,
    nationality: tfModel.nationality,
    position: tfModel.position,
    units: tfModel.units,
    speed: tfModel.speed,
    detection_level: tfModel.detection_level,
    last_move_turn: tfModel.last_move_turn,
    is_activated: tfModel.is_activated,
    is_patrolling: tfModel.is_patrolling,
    available_actions: tfModel.available_actions || [],
    visibility: tfModel.visibility,
    created_at: tfModel.created_at,
    updated_at: tfModel.updated_at,
  };
}

// Функция преобразования GameModel в UnitsResponse
function convertGameModelToUnitsResponse(gameModel: GameModel): UnitsResponse {
  // Преобразуем units из map в массив
  const units: GameUnit[] = Object.values(gameModel.units || {})
    .map(convertUnitModelToGameUnit);

  // Преобразуем task_forces из map в массив
  const taskForces: TaskForce[] = Object.values(gameModel.task_forces || {})
    .map(convertTaskForceModelToTaskForce);

  // Enemy contacts уже массив
  const enemyContacts: EnemyContact[] = (gameModel.enemy_contacts || []).map(contact => ({
    hex_id: contact.hex_id,
    visibility: contact.visibility,
    ship_count: contact.ship_count,
    class_summary: contact.class_summary,
    task_force: contact.task_force,
    task_force_list: contact.task_force_list,
    enemy_nationality: contact.enemy_nationality,
    searching_side: contact.searching_side,
    turn: contact.turn,
    phase: contact.phase,
    last_seen_at: contact.last_seen_at,
  }));

  return {
    success: true,
    data: {
      units,
      task_forces: taskForces,
      enemy_contacts: enemyContacts,
      // Добавляем информацию о текущей фазе из GameModel
      current_turn: gameModel.current_turn,
      // Добавляем события из GameModel
      events: gameModel.events || [],
      // Добавляем данные поиска из GameModel
      search: gameModel.search,
      // Добавляем данные воздушной атаки из GameModel
      air_attack: gameModel.air_attack,
      // Добавляем версию модели для проверки изменений
      version: gameModel.version,
      // Добавляем поля видимости и погоды из GameModel
      visibility_level: gameModel.visibility_level,
      is_fog: gameModel.is_fog,
      weather_track: gameModel.weather_track,
    },
  };
}

// API клиент для работы с юнитами
export const unitsAPI = {
  // Получить юниты игры через новый единый эндпоинт GameModel
  getGameUnits: async (gameId: string, token: string): Promise<UnitsResponse> => {
    try {
      const response = await axios.get(`${API_BASE_URL}/api/games/${gameId}/model`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      });
      
      const raw = response.data;
      
      // Проверяем формат ответа
      if (raw && raw.success && raw.data) {
        // Преобразуем GameModel в UnitsResponse
        return convertGameModelToUnitsResponse(raw.data);
      }
      
      // Если формат не соответствует ожидаемому, возвращаем ошибку
      return {
        success: false,
        data: { units: [], task_forces: [], enemy_contacts: [] },
        error: 'Unexpected response format'
      };
    } catch (error: any) {
      console.error('Error fetching game model:', error);
      return {
        success: false,
        data: { units: [], task_forces: [], enemy_contacts: [] },
        error: error.response?.data?.error || 'Failed to fetch game model'
      };
    }
  },

  // Установить патруль для морского юнита
  setPatrol: async (gameId: string, unitId: string, isPatrolling: boolean, token: string): Promise<{ success: boolean; data?: any; error?: string }> => {
    try {
      const response = await axios.put(
        `${API_BASE_URL}/api/games/${gameId}/units/${unitId}/patrol`,
        { is_patrolling: isPatrolling },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error setting patrol:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to set patrol'
      };
    }
  },

  // Установить патруль для Task Force
  setTaskForcePatrol: async (gameId: string, taskForceId: string, isPatrolling: boolean, token: string): Promise<{ success: boolean; data?: any; error?: string }> => {
    try {
      const response = await axios.put(
        `${API_BASE_URL}/api/games/${gameId}/task-forces/${taskForceId}/actions/patrol`,
        { is_patrolling: isPatrolling },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error setting task force patrol:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to set task force patrol'
      };
    }
  },

  // Выполнить ремонт в море
  repairAtSea: async (gameId: string, unitId: string, token: string): Promise<{ success: boolean; data?: any; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/units/${unitId}/actions/repair`,
        {},
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error repairing at sea:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to repair at sea'
      };
    }
  },

  // Заправить в порту
  refuelAtPort: async (gameId: string, unitId: string, token: string): Promise<{ success: boolean; data?: any; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/units/${unitId}/actions/refuel-port`,
        {},
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error refueling at port:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to refuel at port'
      };
    }
  },

  // Заправить в море
  refuelAtSea: async (gameId: string, unitId: string, token: string): Promise<{ success: boolean; data?: any; error?: string }> => {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/games/${gameId}/units/${unitId}/actions/refuel-sea`,
        {},
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          }
        }
      );
      return response.data;
    } catch (error: any) {
      console.error('Error refueling at sea:', error);
      return {
        success: false,
        error: error.response?.data?.error || 'Failed to refuel at sea'
      };
    }
  },

};

export default unitsAPI;
