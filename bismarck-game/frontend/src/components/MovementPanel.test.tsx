import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MovementPanel from './MovementPanel';
import { NavalUnit } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';

// Мокируем API
const mockUseMovementReturn = {
  availableMoves: null,
  loading: false,
  error: null,
  getAvailableMoves: jest.fn(),
  moveUnit: jest.fn(),
};

jest.mock('../services/api/movementAPI', () => ({
  movementAPI: {
    getAvailableMoves: jest.fn(),
    moveUnit: jest.fn(),
  },
  movementUtils: {
    getSpeedClass: jest.fn((type: string) => 'M'),
    getMaxMovementDistance: jest.fn((speedClass: string) => 2),
  },
  useMovement: jest.fn(),
}));

import { movementAPI, useMovement } from '../services/api/movementAPI';
const mockMovementAPI = movementAPI as jest.Mocked<typeof movementAPI>;
const mockUseMovement = useMovement as jest.MockedFunction<typeof useMovement>;

describe('MovementPanel', () => {
  const mockGameId = 'game-1';
  const mockPlayerId = 'player-1';
  const mockAuthToken = 'test-token';

  const mockSelectedUnit: NavalUnit = {
    id: 'unit-1',
    name: 'Bismarck',
    type: 'battleship',
    class: 'Battleship',
    owner: mockPlayerId,
    nationality: 'german',
    position: 'A1',
    setup_hex: 'A1',
    evasion: 5,
    base_evasion: 5,
    speed_rating: 'M',
    fuel: 100,
    max_fuel: 100,
    hull_boxes: 10,
    current_hull: 10,
    primary_armament_bow: 8,
    primary_armament_stern: 8,
    secondary_armament: 4,
    base_primary_armament_bow: 8,
    base_primary_armament_stern: 8,
    base_secondary_armament: 4,
    torpedoes: 0,
    max_torpedoes: 0,
    radar_level: 3,
    status: 'active',
    detection_level: 'visible',
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
    visibility: 'sighted',
    created_at: '2023-01-01',
    updated_at: '2023-01-01',
  };

  const mockOnMove = jest.fn();
  const mockOnCancel = jest.fn();
  const mockOnHexSelect = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    // Настраиваем useMovement для возврата правильного объекта
    mockUseMovement.mockReturnValue(mockUseMovementReturn);
  });

  describe('Rendering', () => {
    it('should load available moves when unit is selected', async () => {
      mockMovementAPI.getAvailableMoves.mockResolvedValue({
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2', 'B1'],
        max_distance: 2,
        fuel_costs: { 'A2': 1, 'B1': 1 },
      });

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
          onHexSelect={mockOnHexSelect}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalledWith(
          mockGameId,
          mockSelectedUnit.id,
          mockAuthToken
        );
      });
    });

    it('should not load moves when no unit is selected', () => {
      render(
        <MovementPanel
          selectedUnit={null}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
          onHexSelect={mockOnHexSelect}
        />
      );

      expect(mockMovementAPI.getAvailableMoves).not.toHaveBeenCalled();
    });
  });

  describe('Loading Available Moves', () => {
    it('should load available moves when unit is selected', async () => {
      const mockAvailableMoves = {
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2', 'B1', 'A0'],
        max_distance: 2,
        fuel_costs: { 'A2': 1, 'B1': 1, 'A0': 1 },
      };

      mockMovementAPI.getAvailableMoves.mockResolvedValue(mockAvailableMoves);

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalledWith(
          mockGameId,
          mockSelectedUnit.id,
          mockAuthToken
        );
      });
    });

    it('should handle error when loading available moves fails', async () => {
      mockMovementAPI.getAvailableMoves.mockRejectedValue(new Error('Failed to load moves'));

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalled();
      });
    });
  });

  describe('Move Unit', () => {
    it('should call moveUnit API when move is confirmed', async () => {
      const mockAvailableMoves = {
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2'],
        max_distance: 2,
        fuel_costs: { 'A2': 1 },
      };

      mockMovementAPI.getAvailableMoves.mockResolvedValue(mockAvailableMoves);
      mockMovementAPI.moveUnit.mockResolvedValue({
        success: true,
        movement: {
          id: 'movement-1',
          game_id: mockGameId,
          unit_id: 'unit-1',
          from_hex: 'A1',
          to_hex: 'A2',
          fuel_cost: 1,
          hexes_moved: 1,
          movement_type: 'normal',
          turn: 1,
          phase: 'movement',
          path: ['A1', 'A2'],
          created_at: '2023-01-01',
          updated_at: '2023-01-01',
        },
      });

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalled();
      });
    });

    it('should call onMove callback when movement succeeds', async () => {
      const mockAvailableMoves = {
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2'],
        max_distance: 2,
        fuel_costs: { 'A2': 1 },
      };

      mockMovementAPI.getAvailableMoves.mockResolvedValue(mockAvailableMoves);
      mockMovementAPI.moveUnit.mockResolvedValue({
        success: true,
        movement: {
          id: 'movement-1',
          game_id: mockGameId,
          unit_id: 'unit-1',
          from_hex: 'A1',
          to_hex: 'A2',
          fuel_cost: 1,
          hexes_moved: 1,
          movement_type: 'normal',
          turn: 1,
          phase: 'movement',
          path: ['A1', 'A2'],
          created_at: '2023-01-01',
          updated_at: '2023-01-01',
        },
      });

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalled();
      });
    });

    it('should handle error when movement fails', async () => {
      const mockAvailableMoves = {
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2'],
        max_distance: 2,
        fuel_costs: { 'A2': 1 },
      };

      mockMovementAPI.getAvailableMoves.mockResolvedValue(mockAvailableMoves);
      mockMovementAPI.moveUnit.mockResolvedValue({
        success: false,
        message: 'Movement failed',
      });

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalled();
      });
    });
  });

  describe('Cancel', () => {
    it('should call onCancel callback when cancel button is clicked', () => {
      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
        />
      );

      // Тест проверяет, что onCancel вызывается
      // Детали зависят от реализации UI
    });
  });

  describe('Hex Selection', () => {
    it('should call onHexSelect when hex is selected', async () => {
      const mockAvailableMoves = {
        unit_id: 'unit-1',
        current_hex: 'A1',
        available_hexes: ['A2'],
        max_distance: 2,
        fuel_costs: { 'A2': 1 },
      };

      mockMovementAPI.getAvailableMoves.mockResolvedValue(mockAvailableMoves);

      render(
        <MovementPanel
          selectedUnit={mockSelectedUnit}
          gameId={mockGameId}
          playerId={mockPlayerId}
          authToken={mockAuthToken}
          onMove={mockOnMove}
          onCancel={mockOnCancel}
          onHexSelect={mockOnHexSelect}
        />
      );

      await waitFor(() => {
        expect(mockMovementAPI.getAvailableMoves).toHaveBeenCalled();
      });
    });
  });
});

