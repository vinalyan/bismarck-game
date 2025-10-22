import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Lobby from '../Lobby';

// Mock the gameAPI
jest.mock('../../services/api/gameAPI', () => ({
  gameAPI: {
    getGames: jest.fn(),
    createGame: jest.fn(),
    joinGame: jest.fn()
  }
}));

// Mock the gameStore
const mockGameStore = {
  user: {
    id: 'test-user-id',
    username: 'testuser'
  },
  games: [],
  joinGame: jest.fn(),
  ui: {
    currentView: 'lobby'
  }
};

jest.mock('../../stores/gameStore', () => ({
  useGameStore: () => mockGameStore
}));

import { gameAPI } from '../../services/api/gameAPI';
import { useGameStore } from '../../stores/gameStore';

const mockGameAPI = gameAPI as jest.Mocked<typeof gameAPI>;
// Remove the mock function since we're using a direct mock object

describe('Lobby', () => {
  const mockJoinGame = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockGameStore.joinGame = mockJoinGame;
  });

  it('should render lobby with create game button', () => {
    render(<Lobby />);
    
    expect(screen.getByText(/create new game/i)).toBeInTheDocument();
    expect(screen.getByText(/available games/i)).toBeInTheDocument();
  });

  it('should display games list', async () => {
    const mockGames = [
      {
        id: 'game-1',
        name: 'Game 1',
        status: 'waiting',
        player1_username: 'player1',
        player2_username: null,
        current_turn: 1,
        current_phase: 'waiting'
      },
      {
        id: 'game-2',
        name: 'Game 2',
        status: 'active',
        player1_username: 'player1',
        player2_username: 'player2',
        current_turn: 2,
        current_phase: 'movement'
      }
    ];

    mockGameAPI.getGames.mockResolvedValueOnce(mockGames);
    mockGameStore.user = {
      id: 'test-user-id',
      username: 'testuser'
    };
    mockGameStore.games = mockGames;
    mockGameStore.joinGame = mockJoinGame;

    render(<Lobby />);

    expect(screen.getByText('Game 1')).toBeInTheDocument();
    expect(screen.getByText('Game 2')).toBeInTheDocument();
  });

  it('should handle create game button click', async () => {
    const user = userEvent;
    const mockGame = {
      id: 'new-game-id',
      name: 'New Game',
      status: 'waiting',
      player1_id: 'test-user-id',
      player2_id: null
    };

    mockGameAPI.createGame.mockResolvedValueOnce(mockGame);

    render(<Lobby />);

    await user.click(screen.getByText(/create new game/i));

    // Should open create game modal
    expect(screen.getByText(/create game/i)).toBeInTheDocument();
  });

  it('should handle join game button click', async () => {
    const user = userEvent;
    const mockGame = {
      id: 'game-1',
      name: 'Game 1',
      status: 'waiting',
      player1_username: 'player1',
      player2_username: null
    };

    mockGameAPI.joinGame.mockResolvedValueOnce(mockGame);

    mockUseGameStore.mockReturnValue({
      user: {
        id: 'test-user-id',
        username: 'testuser'
      },
      games: [mockGame],
      joinGame: mockJoinGame,
      ui: {
        currentView: 'lobby'
      }
    } as any);

    render(<Lobby />);

    await user.click(screen.getByText(/join/i));

    await waitFor(() => {
      expect(mockGameAPI.joinGame).toHaveBeenCalledWith('game-1', {
        side: 'allied',
        password: ''
      });
    });

    expect(mockJoinGame).toHaveBeenCalledWith('game-1');
  });

  it('should handle game search', async () => {
    const user = userEvent;
    const mockGames = [
      {
        id: 'game-1',
        name: 'Test Game',
        status: 'waiting'
      }
    ];

    mockGameAPI.getGames.mockResolvedValueOnce(mockGames);

    render(<Lobby />);

    await user.type(screen.getByPlaceholderText(/search games/i), 'test');

    await waitFor(() => {
      expect(mockGameAPI.getGames).toHaveBeenCalledWith('test');
    });
  });

  it('should display game status correctly', () => {
    const mockGames = [
      {
        id: 'game-1',
        name: 'Waiting Game',
        status: 'waiting',
        player1_username: 'player1',
        player2_username: null
      },
      {
        id: 'game-2',
        name: 'Active Game',
        status: 'active',
        player1_username: 'player1',
        player2_username: 'player2'
      }
    ];

    mockGameStore.user = {
      id: 'test-user-id',
      username: 'testuser'
    };
    mockGameStore.games = mockGames;
    mockGameStore.joinGame = mockJoinGame;

    render(<Lobby />);

    expect(screen.getByText(/waiting/i)).toBeInTheDocument();
    expect(screen.getByText(/active/i)).toBeInTheDocument();
  });

  it('should show loading state while fetching games', async () => {
    mockGameAPI.getGames.mockImplementationOnce(
      () => new Promise(resolve => setTimeout(() => resolve([]), 100))
    );

    render(<Lobby />);

    expect(screen.getByText(/loading games/i)).toBeInTheDocument();
  });

  it('should handle game creation form submission', async () => {
    const user = userEvent;
    const mockGame = {
      id: 'new-game-id',
      name: 'My New Game',
      status: 'waiting',
      player1_id: 'test-user-id',
      player2_id: null
    };

    mockGameAPI.createGame.mockResolvedValueOnce(mockGame);

    mockUseGameStore.mockReturnValue({
      user: {
        id: 'test-user-id',
        username: 'testuser'
      },
      games: [mockGame],
      joinGame: mockJoinGame,
      ui: {
        currentView: 'lobby'
      }
    } as any);

    render(<Lobby />);

    // Open create game modal
    await user.click(screen.getByText(/create new game/i));

    // Fill form
    await user.type(screen.getByLabelText(/game name/i), 'My New Game');
    await user.selectOptions(screen.getByLabelText(/side/i), 'german');
    await user.click(screen.getByRole('button', { name: /create/i }));

    await waitFor(() => {
      expect(mockGameAPI.createGame).toHaveBeenCalledWith({
        name: 'My New Game',
        side: 'german',
        password: '',
        settings: {}
      });
    });
  });

  it('should handle join game error', async () => {
    const user = userEvent;
    const errorMessage = 'Game not found';

    mockGameAPI.joinGame.mockRejectedValueOnce(new Error(errorMessage));

    mockUseGameStore.mockReturnValue({
      user: {
        id: 'test-user-id',
        username: 'testuser'
      },
      games: [{
        id: 'game-1',
        name: 'Game 1',
        status: 'waiting',
        player1_username: 'player1',
        player2_username: null
      }],
      joinGame: mockJoinGame,
      ui: {
        currentView: 'lobby'
      }
    } as any);

    render(<Lobby />);

    await user.click(screen.getByText(/join/i));

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });
  });
});
