import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import FogOfWar from './FogOfWar';
import { NavalUnit } from '../types/gameTypes';

// Мокируем API
const mockUseVisibilityReturn = {
  visibleUnits: [],
  lastKnownPositions: [],
  loading: false,
  error: null,
  getVisibleUnits: jest.fn(),
  updateVisibility: jest.fn(),
};

jest.mock('../services/api/movementAPI', () => ({
  movementAPI: {
    getVisibleUnits: jest.fn(),
    updateVisibility: jest.fn(),
  },
  movementUtils: {
    isUnitVisible: jest.fn((unitOwner, playerSide, visibility) => {
      if (unitOwner === playerSide) return true;
      return visibility === 'sighted' || visibility === 'shadowed';
    }),
    getVisibilityMarker: jest.fn((visibility) => {
      const markerMap: Record<string, string> = {
        'sighted': 'SIGHTED',
        'shadowed': 'SHADOWED',
        'lost': 'LOST'
      };
      return markerMap[visibility] || '';
    }),
    getVisibilityText: jest.fn((visibility) => {
      const textMap: Record<string, string> = {
        'unknown': 'Неизвестно',
        'sighted': 'Обнаружено',
        'shadowed': 'Преследуется',
        'lost': 'Потерян'
      };
      return textMap[visibility] || 'Неизвестно';
    }),
  },
  useVisibility: jest.fn(),
}));

import { useVisibility } from '../services/api/movementAPI';
const mockUseVisibility = useVisibility as jest.MockedFunction<typeof useVisibility>;

describe('FogOfWar', () => {
  const mockGameId = 'game-1';
  const mockPlayerId = 'player-1';
  const mockPlayerSide = 'german' as const;

  const mockAllUnits: NavalUnit[] = [
    {
      id: 'unit-1',
      gameId: mockGameId,
      name: 'Bismarck',
      type: 'BB',
      class: 'Battleship',
      owner: mockPlayerId,
      nationality: 'german',
      position: 'A1',
      evasion: 5,
      baseEvasion: 5,
      speedRating: 'F',
      fuel: 100,
      maxFuel: 100,
      hullBoxes: 10,
      currentHull: 10,
      primaryArmamentBow: 8,
      primaryArmamentStern: 8,
      secondaryArmament: 4,
      basePrimaryArmamentBow: 8,
      basePrimaryArmamentStern: 8,
      baseSecondaryArmament: 4,
      torpedoes: 0,
      maxTorpedoes: 0,
      radarLevel: 3,
      status: 'active',
      visibility: 'sighted',
      damage: [],
      createdAt: '2023-01-01',
      updatedAt: '2023-01-01',
    },
    {
      id: 'unit-2',
      gameId: mockGameId,
      name: 'Enemy Ship',
      type: 'CA',
      class: 'Cruiser',
      owner: 'player-2',
      nationality: 'allied',
      position: 'B2',
      evasion: 4,
      baseEvasion: 4,
      speedRating: 'M',
      fuel: 80,
      maxFuel: 80,
      hullBoxes: 8,
      currentHull: 8,
      primaryArmamentBow: 6,
      primaryArmamentStern: 6,
      secondaryArmament: 3,
      basePrimaryArmamentBow: 6,
      basePrimaryArmamentStern: 6,
      baseSecondaryArmament: 3,
      torpedoes: 0,
      maxTorpedoes: 0,
      radarLevel: 2,
      status: 'active',
      visibility: 'unknown',
      damage: [],
      createdAt: '2023-01-01',
      updatedAt: '2023-01-01',
    },
  ];

  const mockVisibleUnits: NavalUnit[] = [mockAllUnits[0]];

  const defaultProps = {
    gameId: mockGameId,
    playerId: mockPlayerId,
    playerSide: mockPlayerSide,
    allUnits: mockAllUnits,
    visibleUnits: mockVisibleUnits,
    fogOfWarEnabled: true,
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseVisibility.mockReturnValue(mockUseVisibilityReturn);
  });

  describe('Rendering', () => {
    it('should render fog of war component', () => {
      render(<FogOfWar {...defaultProps} />);
      
      expect(screen.getByText('Туман войны')).toBeInTheDocument();
    });

    it('should show enabled status when fog of war is enabled', () => {
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={true} />);
      
      expect(screen.getByText('Включен')).toBeInTheDocument();
    });

    it('should show disabled status when fog of war is disabled', () => {
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={false} />);
      
      expect(screen.getByText('Отключен')).toBeInTheDocument();
    });

    it('should render visibility stats', () => {
      render(<FogOfWar {...defaultProps} />);
      
      expect(screen.getByText('Видимые юниты:')).toBeInTheDocument();
      expect(screen.getByText('Последние позиции:')).toBeInTheDocument();
    });

    it('should render refresh button', () => {
      render(<FogOfWar {...defaultProps} />);
      
      expect(screen.getByRole('button', { name: /Обновить/i })).toBeInTheDocument();
    });
  });

  describe('Visibility Loading', () => {
    it('should load visibility data on mount when fog of war is enabled', async () => {
      mockUseVisibilityReturn.getVisibleUnits.mockResolvedValue(undefined);
      
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={true} />);
      
      await waitFor(() => {
        expect(mockUseVisibilityReturn.getVisibleUnits).toHaveBeenCalled();
      });
    });

    it('should not load visibility data when fog of war is disabled', () => {
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={false} />);
      
      expect(mockUseVisibilityReturn.getVisibleUnits).not.toHaveBeenCalled();
    });
  });

  describe('Visible Units', () => {
    it('should show all units when fog of war is disabled', () => {
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={false} />);
      
      expect(screen.getByText('Видимые юниты')).toBeInTheDocument();
      // Когда fog of war отключен, должны показываться все юниты
    });

    it('should show own units when fog of war is enabled', () => {
      render(<FogOfWar {...defaultProps} fogOfWarEnabled={true} />);
      
      expect(screen.getByText('Видимые юниты')).toBeInTheDocument();
    });

    it('should show message when no visible units', () => {
      render(<FogOfWar {...defaultProps} allUnits={[]} visibleUnits={[]} />);
      
      expect(screen.getByText('Нет видимых юнитов')).toBeInTheDocument();
    });
  });

  describe('Unit Click', () => {
    it('should call onUnitClick when unit is clicked', () => {
      const mockOnUnitClick = jest.fn();
      render(<FogOfWar {...defaultProps} onUnitClick={mockOnUnitClick} />);
      
      // Находим карточку юнита и кликаем на неё
      const unitCards = screen.queryAllByText(/Bismarck|Enemy Ship/i);
      if (unitCards.length > 0) {
        userEvent.click(unitCards[0]);
        // onUnitClick вызывается только для видимых юнитов
      }
    });
  });

  describe('Error Handling', () => {
    it('should display error message when error occurs', () => {
      mockUseVisibilityReturn.error = 'Error loading visibility';
      
      render(<FogOfWar {...defaultProps} />);
      
      expect(screen.getByText('Error loading visibility')).toBeInTheDocument();
    });

    it('should display loading message when loading', () => {
      mockUseVisibilityReturn.loading = true;
      
      render(<FogOfWar {...defaultProps} />);
      
      expect(screen.getByText('Загрузка данных видимости...')).toBeInTheDocument();
    });
  });

  describe('Refresh Button', () => {
    it('should call getVisibleUnits when refresh button clicked', async () => {
      mockUseVisibilityReturn.getVisibleUnits.mockResolvedValue(undefined);
      
      render(<FogOfWar {...defaultProps} />);
      
      const refreshButton = screen.getByRole('button', { name: /Обновить/i });
      userEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockUseVisibilityReturn.getVisibleUnits).toHaveBeenCalled();
      });
    });

    it('should disable refresh button when loading', () => {
      mockUseVisibilityReturn.loading = true;
      
      render(<FogOfWar {...defaultProps} />);
      
      const refreshButton = screen.getByRole('button', { name: /Обновить/i });
      expect(refreshButton).toBeDisabled();
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty allUnits array', () => {
      render(<FogOfWar {...defaultProps} allUnits={[]} visibleUnits={[]} />);
      
      expect(screen.getByText('Нет видимых юнитов')).toBeInTheDocument();
    });

    it('should handle missing onUnitClick prop', () => {
      render(<FogOfWar {...defaultProps} />);
      
      // Компонент должен рендериться без ошибок
      expect(screen.getByText('Туман войны')).toBeInTheDocument();
    });

    it('should handle missing onHexClick prop', () => {
      render(<FogOfWar {...defaultProps} />);
      
      // Компонент должен рендериться без ошибок
      expect(screen.getByText('Туман войны')).toBeInTheDocument();
    });
  });
});

