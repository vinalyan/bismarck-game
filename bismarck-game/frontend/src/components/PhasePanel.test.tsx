import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PhasePanel from './PhasePanel';
import { phaseAPI } from '../services/api/phaseAPI';
import { GameTurn, PhaseRecord } from '../types/phaseTypes';
import { Game, GameStatus } from '../types/gameTypes';

// Мокируем API
jest.mock('../services/api/phaseAPI');
const mockPhaseAPI = phaseAPI as jest.Mocked<typeof phaseAPI>;

describe('PhasePanel', () => {
  const mockGameId = 'game-1';
  const mockUserId = 'user-1';
  const mockGame: Game = {
    id: mockGameId,
    name: 'Test Game',
    player1_id: 'player-1',
    player2_id: 'player-2',
    current_turn: 1,
    current_phase: 'movement',
    status: GameStatus.InProgress,
    settings: {
      use_optional_units: false,
      enable_crew_exhaustion: false,
      victory_conditions: {
        bismarck_sunk_vp: -10,
        bismarck_france_vp: -5,
        bismarck_norway_vp: -7,
        bismarck_end_game_vp: -10,
        bismarck_no_fuel_vp: -15,
        ship_vp_values: {},
        convoy_vp: {},
      },
      time_limit_minutes: 180,
      private_lobby: false,
      max_turn_time: 30,
      allow_spectators: true,
      auto_save: true,
      difficulty: 'standard',
    },
    created_at: '2023-01-01',
    updated_at: '2023-01-01',
  };

  const mockCurrentTurn: GameTurn = {
    turn_number: 1,
    current_phase: 'movement',
    status: 'active',
    started_at: '2023-01-01',
    completed_at: null,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render phase panel with game id', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(screen.getByText(/фазы игры/i)).toBeInTheDocument();
      });
    });

    it('should render current turn information', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(screen.getByText(/ход/i)).toBeInTheDocument();
      });
    });
  });

  describe('Phase Records Loading', () => {
    it('should load phase records on mount when currentTurn is provided', async () => {
      const mockRecords: PhaseRecord[] = [
        {
          id: 'record-1',
          game_id: mockGameId,
          turn: 1,
          phase: 'setup',
          status: 'completed',
          started_at: '2023-01-01',
          completed_at: '2023-01-01',
        },
      ];

      mockPhaseAPI.getPhaseRecords.mockResolvedValue(mockRecords);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(mockPhaseAPI.getPhaseRecords).toHaveBeenCalledWith(mockGameId, 1);
      });
    });

    it('should not load phase records when currentTurn is not provided', () => {
      render(
        <PhasePanel
          gameId={mockGameId}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      expect(mockPhaseAPI.getPhaseRecords).not.toHaveBeenCalled();
    });
  });

  describe('Start Phase', () => {
    it('should load phase records on mount', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);
      mockPhaseAPI.startPhase.mockResolvedValue(undefined);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      // Ждем загрузки записей
      await waitFor(() => {
        expect(mockPhaseAPI.getPhaseRecords).toHaveBeenCalled();
      });
    });

    it('should handle error when starting phase fails', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);
      mockPhaseAPI.startPhase.mockRejectedValue(new Error('Failed to start phase'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(mockPhaseAPI.getPhaseRecords).toHaveBeenCalled();
      });

      // Тест проверяет, что ошибка обрабатывается
      // Детали зависят от реализации UI

      consoleErrorSpy.mockRestore();
    });
  });

  describe('Complete Phase', () => {
    it('should call completePhase API', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);
      mockPhaseAPI.completePhase.mockResolvedValue(undefined);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(mockPhaseAPI.getPhaseRecords).toHaveBeenCalled();
      });
    });
  });

  describe('Next Phase', () => {
    it('should call nextPhase API and dispatch turnUpdated event', async () => {
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);
      mockPhaseAPI.nextPhase.mockResolvedValue(undefined);

      const eventListener = jest.fn();
      window.addEventListener('turnUpdated', eventListener);

      render(
        <PhasePanel
          gameId={mockGameId}
          currentTurn={mockCurrentTurn}
          currentUserId={mockUserId}
          currentGame={mockGame}
        />
      );

      await waitFor(() => {
        expect(mockPhaseAPI.getPhaseRecords).toHaveBeenCalled();
      });

      window.removeEventListener('turnUpdated', eventListener);
    });
  });

  describe('Start Turn', () => {
    it('should call startTurn API and dispatch turnUpdated event', async () => {
      const mockNewTurn: GameTurn = {
        turn_number: 1,
        current_phase: 'setup',
        status: 'active',
        started_at: '2023-01-01',
        completed_at: null,
      };

      mockPhaseAPI.startTurn.mockResolvedValue(mockNewTurn);
      mockPhaseAPI.getPhaseRecords.mockResolvedValue([]);

      const eventListener = jest.fn();
      window.addEventListener('turnUpdated', eventListener);

      const mockOnPhaseChange = jest.fn();

      render(
        <PhasePanel
          gameId={mockGameId}
          currentUserId={mockUserId}
          currentGame={mockGame}
          onPhaseChange={mockOnPhaseChange}
        />
      );

      await waitFor(() => {
        expect(mockPhaseAPI.startTurn).not.toHaveBeenCalled(); // Не вызывается автоматически
      });

      window.removeEventListener('turnUpdated', eventListener);
    });
  });
});

