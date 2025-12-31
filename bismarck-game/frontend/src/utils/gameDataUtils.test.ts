import {
  extractSearchDataFromModel,
  createGameTurnFromModel,
  updateGameDataFromModel
} from './gameDataUtils';
import { UnitsResponse } from '../services/api/unitsAPI';
import { GameResponse } from '../types/gameTypes';
import { GameTurn } from '../types/phaseTypes';
import { HexMarkers } from '../services/api/searchAPI';

describe('gameDataUtils', () => {
  describe('extractSearchDataFromModel()', () => {
    it('should extract search data from response with search_hexes', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: 5,
                air_search: 0
              },
              'B2': {
                factor: 10,
                air_search: 3
              }
            }
          }
        }
      };

      const result = extractSearchDataFromModel(response, 'german');
      
      expect(result.factorsMap.size).toBe(2);
      expect(result.factorsMap.get('A1')).toBe(5);
      expect(result.factorsMap.get('B2')).toBe(10);
      expect(result.markersMap['B2']).toBeDefined();
      expect(result.markersMap['B2']?.flight_path_search).toBe(3);
    });

    it('should create markers map only for hexes with air_search > 0', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: 5,
                air_search: 0
              },
              'B2': {
                factor: 10,
                air_search: 5
              }
            }
          }
        }
      };

      const result = extractSearchDataFromModel(response, 'german');
      
      expect(result.markersMap['A1']).toBeUndefined();
      expect(result.markersMap['B2']).toBeDefined();
      expect(result.markersMap['B2']?.flight_path_search).toBe(5);
    });

    it('should return empty maps when search_hexes is missing', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: []
        }
      };

      const result = extractSearchDataFromModel(response, 'german');
      
      expect(result.factorsMap.size).toBe(0);
      expect(Object.keys(result.markersMap).length).toBe(0);
    });

    it('should return empty maps when search is missing', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: []
        }
      };

      const result = extractSearchDataFromModel(response, 'german');
      
      expect(result.factorsMap.size).toBe(0);
      expect(Object.keys(result.markersMap).length).toBe(0);
    });

    it('should handle missing factor values', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: undefined as any,
                air_search: 0
              }
            }
          }
        }
      };

      const result = extractSearchDataFromModel(response, 'german');
      
      expect(result.factorsMap.get('A1')).toBe(0);
    });

    it('should work for both german and allied sides', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: 5,
                air_search: 0
              }
            }
          }
        }
      };

      const resultGerman = extractSearchDataFromModel(response, 'german');
      const resultAllied = extractSearchDataFromModel(response, 'allied');
      
      expect(resultGerman.factorsMap.size).toBe(1);
      expect(resultAllied.factorsMap.size).toBe(1);
    });
  });

  describe('createGameTurnFromModel()', () => {
    const mockGame: GameResponse = {
      id: 'test-game',
      name: 'Test Game',
      status: 'active',
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z',
      visibility_level: 3,
      is_fog: false
    };

    it('should create GameTurn from valid turn data', () => {
      const currentTurnData = {
        turn: 5,
        phase: 'movement'
      };

      const result = createGameTurnFromModel(currentTurnData, mockGame);
      
      expect(result).not.toBeNull();
      expect(result?.turn_number).toBe(5);
      expect(result?.current_phase).toBe('movement');
      expect(result?.game_id).toBe('test-game');
      expect(result?.visibility_level).toBe(3);
    });

    it('should return null for turn 0', () => {
      const currentTurnData = {
        turn: 0,
        phase: 'setup'
      };

      const result = createGameTurnFromModel(currentTurnData, mockGame);
      expect(result).toBeNull();
    });

    it('should return null for undefined turn data', () => {
      const result = createGameTurnFromModel(undefined, mockGame);
      expect(result).toBeNull();
    });

    it('should include game properties', () => {
      const currentTurnData = {
        turn: 3,
        phase: 'search'
      };

      const result = createGameTurnFromModel(currentTurnData, mockGame);
      
      expect(result?.visibility_level).toBe(mockGame.visibility_level);
      expect(result?.is_fog).toBe(mockGame.is_fog);
    });

    it('should handle null game', () => {
      const currentTurnData = {
        turn: 2,
        phase: 'movement'
      };

      const result = createGameTurnFromModel(currentTurnData, null);
      
      expect(result).not.toBeNull();
      expect(result?.game_id).toBe('');
    });
  });

  describe('updateGameDataFromModel()', () => {
    const mockGame: GameResponse = {
      id: 'test-game',
      name: 'Test Game',
      status: 'active',
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z'
    };

    const mockUpdateGame = jest.fn();

    beforeEach(() => {
      mockUpdateGame.mockClear();
    });

    it('should extract units and task forces', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [
            {
              id: 'unit-1',
              game_id: 'test-game',
              name: 'Unit 1',
              type: 'BB',
              class: 'Battleship',
              owner: 'german',
              nationality: 'german',
              position: 'A1',
              setup_hex: 'A1',
              evasion: 5,
              base_evasion: 5,
              speed_rating: 'fast',
              fuel: 100,
              max_fuel: 100,
              hull_boxes: 10,
              current_hull: 10,
              primary_armament_bow: 15,
              primary_armament_stern: 15,
              secondary_armament: 8,
              base_primary_armament_bow: 15,
              base_primary_armament_stern: 15,
              base_secondary_armament: 8,
              torpedoes: 0,
              max_torpedoes: 0,
              radar_level: 2,
              status: 'active',
              detection_level: 'sighted',
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
              created_at: '2023-01-01T00:00:00Z',
              updated_at: '2023-01-01T00:00:00Z'
            }
          ],
          task_forces: [
            {
              id: 'tf-1',
              name: 'Task Force 1',
              nationality: 'german',
              position: 'B2',
              units: ['unit-1'],
              speed: 5,
              detection_level: 'sighted',
              last_move_turn: 0,
              is_activated: true,
              is_patrolling: false,
              created_at: '2023-01-01T00:00:00Z',
              updated_at: '2023-01-01T00:00:00Z'
            }
          ],
          enemy_contacts: []
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(result.units.length).toBe(1);
      expect(result.taskForces.length).toBe(1);
      expect(result.enemyContacts.length).toBe(0);
    });

    it('should extract search data for german side', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: 5,
                air_search: 0
              }
            }
          }
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(result.searchFactorHexes.size).toBe(1);
      expect(result.searchFactorHexes.get('A1')).toBe(5);
    });

    it('should extract search data for allied side', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'B2': {
                factor: 10,
                air_search: 3
              }
            }
          }
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'allied', mockUpdateGame);
      
      expect(result.searchFactorHexes.size).toBe(1);
      expect(result.searchFactorHexes.get('B2')).toBe(10);
      expect(result.hexMarkers['B2']?.flight_path_search).toBe(3);
    });

    it('should not extract search data for unknown side', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          search: {
            search_hexes: {
              'A1': {
                factor: 5,
                air_search: 0
              }
            }
          }
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'unknown', mockUpdateGame);
      
      expect(result.searchFactorHexes.size).toBe(0);
      expect(Object.keys(result.hexMarkers).length).toBe(0);
    });

    it('should create GameTurn from current_turn data', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          current_turn: {
            turn: 3,
            phase: 'movement'
          }
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(result.currentTurn).not.toBeNull();
      expect(result.currentTurn?.turn_number).toBe(3);
      expect(result.currentTurn?.current_phase).toBe('movement');
    });

    it('should update game in store when current_turn exists', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: [],
          current_turn: {
            turn: 5,
            phase: 'search'
          }
        }
      };

      updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(mockUpdateGame).toHaveBeenCalledWith('test-game', {
        current_turn: 5,
        current_phase: 'search'
      });
    });

    it('should update game to setup when current_turn is missing', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: []
        }
      };

      updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(mockUpdateGame).toHaveBeenCalledWith('test-game', {
        current_turn: 0,
        current_phase: 'setup'
      });
    });

    it('should return empty arrays when data is missing', () => {
      const response: UnitsResponse = {
        success: true,
        data: {
          units: []
        }
      };

      const result = updateGameDataFromModel(response, mockGame, 'german', mockUpdateGame);
      
      expect(result.units).toEqual([]);
      expect(result.taskForces).toEqual([]);
      expect(result.enemyContacts).toEqual([]);
    });
  });
});

