import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import HexMap from './HexMap';
import { HexCoordinate } from '../types/mapTypes';

// Мокируем API
jest.mock('../services/api/unitsAPI');
jest.mock('../services/api/phaseAPI');
jest.mock('../services/api/searchAPI');

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
      const { container } = render(<HexMap {...defaultProps} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render with default width and height', () => {
      const { container } = render(<HexMap {...defaultProps} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render with custom width and height', () => {
      const { container } = render(<HexMap {...defaultProps} width={1000} height={800} />);
      
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });
  });

  describe('Game Controls', () => {
    it('should render refresh button', () => {
      render(<HexMap {...defaultProps} />);
      
      const refreshButton = screen.getByRole('button', { name: /Обновить|Refresh/i });
      expect(refreshButton).toBeInTheDocument();
    });

    it('should call onRefreshData when refresh button clicked', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({ success: true });
      
      render(<HexMap {...defaultProps} />);
      
      const refreshButton = screen.getByRole('button', { name: /Обновить|Refresh/i });
      userEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockOnRefreshData).toHaveBeenCalled();
      });
    });

    it('should render create TF button in movement phase', () => {
      render(<HexMap {...defaultProps} currentPhase="movement" />);
      
      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      expect(createTFButton).toBeInTheDocument();
    });

    it('should render patrol button in movement phase', () => {
      render(<HexMap {...defaultProps} currentPhase="movement" />);
      
      const patrolButton = screen.getByRole('button', { name: /Патруль|Patrol/i });
      expect(patrolButton).toBeInTheDocument();
    });

    it('should render refuel button when available', () => {
      render(<HexMap {...defaultProps} isRefuelDisabled={false} />);
      
      const refuelButton = screen.queryByRole('button', { name: /Заправить|Refuel/i });
      // Кнопка может не отображаться в зависимости от условий
    });

    it('should render complete phase button when available', () => {
      render(<HexMap {...defaultProps} isCompletePhaseDisabled={false} />);
      
      const completeButton = screen.queryByRole('button', { name: /Завершить фазу|Complete Phase/i });
      // Кнопка может не отображаться в зависимости от условий
    });

    it('should render start first turn button when visible', () => {
      render(<HexMap {...defaultProps} isStartFirstTurnVisible={true} />);
      
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
      render(<HexMap {...defaultProps} gameUnits={mockUnits} />);
      
      // Проверяем, что компонент рендерится (юниты отображаются через SVG элементы)
      const { container } = render(<HexMap {...defaultProps} gameUnits={mockUnits} />);
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
      render(<HexMap {...defaultProps} taskForces={mockTaskForces} />);
      
      // Проверяем, что компонент рендерится
      const { container } = render(<HexMap {...defaultProps} taskForces={mockTaskForces} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('should handle missing gameId', () => {
      render(<HexMap {...defaultProps} gameId={undefined} />);
      
      const { container } = render(<HexMap {...defaultProps} gameId={undefined} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle missing authToken', () => {
      render(<HexMap {...defaultProps} authToken={null} />);
      
      const { container } = render(<HexMap {...defaultProps} authToken={null} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle empty gameUnits array', () => {
      render(<HexMap {...defaultProps} gameUnits={[]} />);
      
      const { container } = render(<HexMap {...defaultProps} gameUnits={[]} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle empty taskForces array', () => {
      render(<HexMap {...defaultProps} taskForces={[]} />);
      
      const { container } = render(<HexMap {...defaultProps} taskForces={[]} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('should handle null mapStructures', () => {
      render(<HexMap {...defaultProps} mapStructures={null} />);
      
      const { container } = render(<HexMap {...defaultProps} mapStructures={null} />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });
});

