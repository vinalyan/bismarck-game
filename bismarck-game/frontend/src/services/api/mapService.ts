// Базовый URL API
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export interface MapStructure {
  landAreas: LandArea[];
  nonGameHexes: NonGameHex[];
  restrictedDD?: RestrictedDD;
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

export const mapService = {
  async getMapStructures(): Promise<MapStructure> {
    const response = await fetch(`${API_BASE_URL}/map/structures`);
    if (!response.ok) {
      throw new Error(`Failed to fetch map structures: ${response.statusText}`);
    }
    const data = await response.json();
    return data.mapStructures;
  }
};
