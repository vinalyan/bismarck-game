// Типы структур для редактора карты

export type StructureType = 
  | 'port'
  | 'canal'
  | 'convoy_route'
  | 'air_sector'
  | 'english_channel'
  | 'restricted_dd';

export interface PortStructure {
  type: 'port';
  hexIds: string[];
  portType: 'france' | 'norway';
  name: string;
  canRefuel: boolean;
  canReloadTorpedoes: boolean;
}

export interface CanalStructure {
  type: 'canal';
  fromHex: string;
  toHex: string;
  allowedSide: 'german' | 'allied' | 'both';
  name: string;
}

export interface ConvoyRouteStructure {
  type: 'convoy_route';
  hexIds: string[];
  direction: 'ew' | 'ns';
  name: string;
}

export interface AirSectorStructure {
  type: 'air_sector';
  hexIds: string[];
  name: string;
}

export interface EnglishChannelStructure {
  type: 'english_channel';
  hexIds: string[];
}

export interface RestrictedDDStructure {
  type: 'restricted_dd';
  hexIds: string[];
}

export type Structure = 
  | PortStructure
  | CanalStructure
  | ConvoyRouteStructure
  | AirSectorStructure
  | EnglishChannelStructure
  | RestrictedDDStructure;

export interface MapStructures {
  ports: PortStructure[];
  canals: CanalStructure[];
  convoyRoutes: ConvoyRouteStructure[];
  airSectors: AirSectorStructure[];
  englishChannel?: EnglishChannelStructure;
  restrictedDD?: RestrictedDDStructure;
}

// Цветовая схема для типов структур
export const STRUCTURE_COLORS: Record<StructureType, string> = {
  port: '#2196f3',
  canal: '#4caf50',
  convoy_route: '#ff9800',
  air_sector: '#00bcd4',
  english_channel: '#9c27b0',
  restricted_dd: '#f44336',
};

// Читаемые названия типов
export const STRUCTURE_LABELS: Record<StructureType, string> = {
  port: 'Порт',
  canal: 'Канал',
  convoy_route: 'Маршрут конвоя',
  air_sector: 'Воздушный сектор',
  english_channel: 'Ла-Манш',
  restricted_dd: 'Ограничение эсминцев',
};

