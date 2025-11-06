// Базовый URL API
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export interface MapStructure {
  landAreas: LandArea[];
  nonGameHexes: NonGameHex[];
  restrictedDD?: RestrictedDD;
  fogAreas?: FogArea[];
}

export interface LandArea {
  type: string;
  hexIds: string[];
  name: string;
}

export interface NonGameHex {
  type: string;
  hexIds: string[];
  name: string;
}

export interface RestrictedDD {
  type: string;
  hexIds: string[];
}

export interface FogArea {
  type: string;
  hexIds: string[];
  name: string;
}

export const mapService = {
  async getMapStructures(): Promise<MapStructure> {
    const response = await fetch(`${API_BASE_URL}/map/structures`);
    if (!response.ok) {
      throw new Error(`Failed to fetch map structures: ${response.statusText}`);
    }
    const result = await response.json();
    // API возвращает {success: true, data: {mapStructures: {...}}}
    if (result.success && result.data && result.data.mapStructures) {
      return result.data.mapStructures;
    }
    throw new Error('Invalid map structures response format');
  }
};
