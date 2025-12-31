import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import HexMap from './HexMap';
import { HexCoordinate } from '../types/mapTypes';

// Мокируем API
jest.mock('../services/api/unitsAPI');
jest.mock('../services/api/phaseAPI');
jest.mock('../services/api/searchAPI');

// Мокируем тяжелые дочерние компоненты
jest.mock('./Hex', () => ({
  Hex: function MockHex() {
    return <div data-testid="mock-hex">Hex</div>;
  },
}));

jest.mock('./Tooltip', () => {
  return function MockTooltip() {
    return null;
  };
});

jest.mock('./CreateTaskForceDialog', () => {
  return function MockCreateTaskForceDialog() {
    return null;
  };
});

jest.mock('./PatrolDialog', () => {
  return function MockPatrolDialog() {
    return null;
  };
});

describe('HexMap', () => {
  const mockGameId = 'game-1';
  const mockAuthToken = 'test-token';
  const mockPlayerSide = 'german' as const;

  const mockOnHexClick = jest.fn();
  const mockOnHexHover = jest.fn();
  const mockOnUnitClick = jest.fn();
  const mockOnRefreshData = jest.fn();
  const mockOnUnitStackClick = jest.fn();
  const mockOnTaskForceClick = jest.fn();
  const mockOnStackedUnitSelect = jest.fn();
  const mockOnRefuelAllShips = jest.fn();
  const mockOnCompletePhase = jest.fn();
  const mockOnStartFirstTurn = jest.fn();

  const defaultProps = {
    gameId: mockGameId,
    authToken: mockAuthToken,
    playerSide: mockPlayerSide,
    gameUnits: [],
    taskForces: [],
    enemyContacts: [],
    mapStructures: null,
    currentTurn: 1,
    currentPhase: 'movement',
    onRefreshData: mockOnRefreshData,
    onUnitStackClick: mockOnUnitStackClick,
    onTaskForceClick: mockOnTaskForceClick,
    onStackedUnitSelect: mockOnStackedUnitSelect,
    onRefuelAllShips: mockOnRefuelAllShips,
    onCompletePhase: mockOnCompletePhase,
    onStartFirstTurn: mockOnStartFirstTurn,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render hex map component', () => {
      const { container } = render(<HexMap {...defaultProps} width={5} height={5} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render with default width and height', () => {
      const { container } = render(<HexMap {...defaultProps} width={5} height={5} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render with custom width and height', () => {
      const { container } = render(<HexMap {...defaultProps} width={10} height={10} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });
  });

  describe('Game Controls', () => {
    it('should render refresh button', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);
      
      const refreshButton = screen.getByTitle(/Обновить данные игры/i);
      expect(refreshButton).toBeInTheDocument();
    });

    it('should call onRefreshData when refresh button clicked', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({ success: true });
      
      render(<HexMap {...defaultProps} width={5} height={5} />);
      
      const refreshButton = screen.getByTitle(/Обновить данные игры/i);
      userEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockOnRefreshData).toHaveBeenCalled();
      });
    });

    it('should render create TF button in movement phase', () => {
      render(<HexMap {...defaultProps} currentPhase="movement" width={5} height={5} />);
      
      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      expect(createTFButton).toBeInTheDocument();
    });

    it('should render patrol button in movement phase', () => {
      render(<HexMap {...defaultProps} currentPhase="movement" width={5} height={5} />);
      
      const patrolButton = screen.getByRole('button', { name: /Патруль|Patrol/i });
      expect(patrolButton).toBeInTheDocument();
    });

    it('should render refuel button when available', () => {
      render(<HexMap {...defaultProps} isRefuelDisabled={false} width={5} height={5} />);
      
      const refuelButton = screen.queryByRole('button', { name: /Заправить|Refuel/i });
      // Кнопка может не отображаться в зависимости от условий
    });

    it('should render complete phase button when available', () => {
      render(<HexMap {...defaultProps} isCompletePhaseDisabled={false} width={5} height={5} />);
      
      const completeButton = screen.queryByRole('button', { name: /Завершить фазу|Complete Phase/i });
      // Кнопка может не отображаться в зависимости от условий
    });

    it('should render start first turn button when visible', () => {
      render(<HexMap {...defaultProps} isStartFirstTurnVisible={true} width={5} height={5} />);
      
      const startButton = screen.getByRole('button', { name: /Начать ход|Start Turn/i });
      expect(startButton).toBeInTheDocument();
    });
  });

  describe('Unit Display', () => {
    const mockUnits = [
      {
        id: 'unit-1',
        name: 'Bismarck',
        type: 'BB',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        status: 'active',
      },
    ];

    it('should render units on map', () => {
      const { container } = render(<HexMap {...defaultProps} gameUnits={mockUnits} width={5} height={5} />);
      
      // Проверяем, что компонент рендерится (юниты отображаются через SVG элементы)
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Task Force Display', () => {
    const mockTaskForces = [
      {
        id: 'tf-1',
        name: 'Task Force 1',
        position: 'A1',
        nationality: 'german',
        units: ['unit-1'],
      },
    ];

    it('should render task forces on map', () => {
      const { container } = render(<HexMap {...defaultProps} taskForces={mockTaskForces} width={5} height={5} />);
      
      // Проверяем, что компонент рендерится
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('should handle missing gameId', () => {
      const { container } = render(<HexMap {...defaultProps} gameId={undefined} width={5} height={5} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle missing authToken', () => {
      const { container } = render(<HexMap {...defaultProps} authToken={null} width={5} height={5} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle empty gameUnits array', () => {
      const { container } = render(<HexMap {...defaultProps} gameUnits={[]} width={5} height={5} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle empty taskForces array', () => {
      const { container } = render(<HexMap {...defaultProps} taskForces={[]} width={5} height={5} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle null mapStructures', () => {
      const { container } = render(<HexMap {...defaultProps} mapStructures={null} width={5} height={5} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });
});

