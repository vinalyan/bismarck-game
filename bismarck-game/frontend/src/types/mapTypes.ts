// Типы для гексагональной карты

// Координаты гекса
export interface HexCoordinate {
  letter: string;  // A, B, C, ..., AH (34 буквы)
  number: number;  // 1, 2, 3, ..., 35
  q: number;       // Гексагональная координата q
  r: number;       // Гексагональная координата r
}

// Типы гексов
export type HexType = 'water' | 'land' | 'port' | 'ice' | 'fog';

// Стороны игроков
export type PlayerSide = 'german' | 'allied';

// Погодные условия
export type WeatherType = 'clear' | 'storm' | 'fog' | 'ice';

// Данные гекса
export interface HexData {
  coordinate: HexCoordinate;
  type: HexType;
  isVisible: boolean;        // Видим ли гекс игроку
  isHighlighted: boolean;    // Подсвечен ли гекс
  hasUnit: boolean;          // Есть ли юнит на гексе
  unitId?: string | null;    // ID юнита
  unitSide?: PlayerSide | null; // Сторона юнита
  weather: WeatherType;      // Погода на гексе
  fogLevel: number;          // Уровень тумана войны (0-100)
}

// Юнит на карте
export interface MapUnit {
  id: string;
  name: string;
  type: 'battleship' | 'cruiser' | 'destroyer' | 'aircraft_carrier' | 'submarine';
  side: PlayerSide;
  coordinate: HexCoordinate;
  speed: number;
  fuel: number;
  maxFuel: number;
  isVisible: boolean;
  isDetected: boolean;
  lastKnownPosition?: HexCoordinate;
}

// Оперативное соединение (Task Force)
export interface TaskForce {
  id: string;
  name: string;
  side: PlayerSide;
  units: string[]; // IDs юнитов
  coordinate: HexCoordinate;
  formation: 'line' | 'diamond' | 'wedge' | 'scattered';
  speed: number;
  isVisible: boolean;
}

// Движение юнита
export interface UnitMovement {
  unitId: string;
  from: HexCoordinate;
  to: HexCoordinate;
  speed: number;
  fuelCost: number;
  isShadowed: boolean; // Видно ли движение противнику
}

// Поиск и обнаружение
export interface SearchResult {
  coordinate: HexCoordinate;
  units: MapUnit[];
  isDetected: boolean;
  detectionLevel: number; // 0-100
  searchRange: number;
}

// Погодная зона
export interface WeatherZone {
  id: string;
  type: WeatherType;
  coordinates: HexCoordinate[];
  intensity: number; // 0-100
  duration: number; // в ходах
}

// Игровая карта
export interface GameMap {
  width: number;
  height: number;
  hexes: Map<string, HexData>;
  units: Map<string, MapUnit>;
  taskForces: Map<string, TaskForce>;
  weatherZones: WeatherZone[];
  currentTurn: number;
  currentPhase: string;
}

// События карты
export interface MapEvent {
  id: string;
  type: 'movement' | 'detection' | 'combat' | 'weather_change';
  coordinate: HexCoordinate;
  description: string;
  timestamp: number;
  isVisible: boolean;
}

// Настройки карты
export interface MapSettings {
  showCoordinates: boolean;
  showGrid: boolean;
  showWeather: boolean;
  showFogOfWar: boolean;
  showUnitInfo: boolean;
  zoomLevel: number;
  panOffset: { x: number; y: number };
}
