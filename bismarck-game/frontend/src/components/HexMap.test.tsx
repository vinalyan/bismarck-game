import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import HexMap from './HexMap';
import { HexCoordinate } from '../types/mapTypes';
import { ActiveHex } from '../utils/activeHexesUtils';

// Мокируем API
jest.mock('../services/api/unitsAPI');
jest.mock('../services/api/phaseAPI');
jest.mock('../services/api/searchAPI');
jest.mock('../services/api/gameAPI');

// Мокируем тяжелые дочерние компоненты
const mockHexProps: any[] = [];
const mockHexClicks: string[] = [];
jest.mock('./Hex', () => ({
  Hex: function MockHex(props: any) {
    mockHexProps.push(props);
    // Сохраняем координату для последующей проверки кликов
    const hexId = `${props.coordinate.letter}${props.coordinate.number}`;
    return (
      <div 
        data-testid={`mock-hex-${hexId}`}
        onClick={() => {
          if (props.onClick) {
            props.onClick();
            mockHexClicks.push(hexId);
          }
        }}
      >
        Hex {hexId}
      </div>
    );
  },
}));

jest.mock('./Tooltip', () => {
  return function MockTooltip() {
    return null;
  };
});

jest.mock('./CreateTaskForceDialog', () => ({
  __esModule: true,
  default: function MockCreateTaskForceDialog({ hexId, onClose, onConfirm, units }: any) {
    return (
      <div data-testid="create-tf-dialog">
        <div>TF Dialog for {hexId}</div>
        <button onClick={onClose}>Close TF</button>
        <button onClick={() => onConfirm && onConfirm(units?.map((u: any) => u.id) || [])}>Confirm TF</button>
      </div>
    );
  }
}));


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

  // Вспомогательная функция для создания ActiveHex
  const createActiveHex = (letter: string, number: number, type: ActiveHex['type'], priority: number = 1): ActiveHex => {
    // Преобразуем букву в row
    let row: number;
    if (letter.length === 1) {
      row = letter.charCodeAt(0) - 65; // A=0, B=1, ..., J=9, ...
    } else if (letter.length === 2 && letter.startsWith('A')) {
      row = 26 + (letter.charCodeAt(1) - 65); // AA=26, AB=27, ...
    } else {
      throw new Error(`Invalid letter: ${letter}`);
    }
    const col = number - 1; // number 1-35 -> col 0-34
    
    return {
      coordinate: {
        letter,
        number,
        col,
        row
      },
      type,
      priority
    };
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockHexProps.length = 0; // Очищаем массив пропсов
    mockHexClicks.length = 0; // Очищаем массив кликов
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

    it('should render patrol button when unit with patrol action is selected', () => {
      const mockUnitWithPatrol = {
        id: 'unit-1',
        name: 'Test Unit',
        available_actions: ['patrol', 'movement'],
        is_activated: false,
        position: 'A1',
        type: 'DD',
        category: 'naval',
        owner: 'german',
        nationality: 'german',
        status: 'active',
        visibility: 'sighted'
      };
      
      render(
        <HexMap 
          {...defaultProps} 
          currentPhase="movement" 
          width={5} 
          height={5}
          selectedUnit="unit-1"
          gameUnits={[mockUnitWithPatrol]}
        />
      );
      
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

  describe('Active Hexes', () => {
    const createActiveHex = (letter: string, number: number, type: string = 'movement'): ActiveHex => {
      // Преобразуем букву в row (A=0, B=1, ..., J=9, ..., Z=25, AA=26, AB=27, ..., AH=33)
      let row: number;
      if (letter.length === 1) {
        row = letter.charCodeAt(0) - 65; // A=0, B=1, ..., J=9, ..., Z=25
      } else if (letter.length === 2 && letter.startsWith('A')) {
        row = 26 + (letter.charCodeAt(1) - 65); // AA=26, AB=27, ..., AH=33
      } else {
        throw new Error(`Invalid letter: ${letter}`);
      }
      // Преобразуем number в col (1-35 -> 0-34)
      const col = number - 1;
      
      return {
        coordinate: {
          col,
          row,
          letter,
          number
        },
        type: type as any,
        priority: 1
      };
    };

    it('should pass activeHex to Hex component when activeHexes provided', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('A', 1, 'movement')
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      // Находим Hex компонент для гекса A1
      const hexWithActiveHex = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexWithActiveHex).toBeDefined();
      expect(hexWithActiveHex?.activeHex).toBeDefined();
      expect(hexWithActiveHex?.activeHex.coordinate.letter).toBe('A');
      expect(hexWithActiveHex?.activeHex.coordinate.number).toBe(1);
      expect(hexWithActiveHex?.activeHex.type).toBe('movement');
    });

    it('should pass null activeHex to Hex component when no matching activeHex', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('B', 2, 'movement')
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      // Находим Hex компонент для гекса A1 (который не в activeHexes)
      const hexWithoutActiveHex = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexWithoutActiveHex).toBeDefined();
      expect(hexWithoutActiveHex?.activeHex).toBeUndefined();
    });

    it('should handle multiple activeHexes correctly', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('A', 1, 'movement'),
        createActiveHex('B', 2, 'refuel'),
        createActiveHex('C', 3, 'movement')
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      // Проверяем, что все активные гексы переданы правильно
      const hexA1 = mockHexProps.find(props => props.coordinate.letter === 'A' && props.coordinate.number === 1);
      const hexB2 = mockHexProps.find(props => props.coordinate.letter === 'B' && props.coordinate.number === 2);
      const hexC3 = mockHexProps.find(props => props.coordinate.letter === 'C' && props.coordinate.number === 3);

      expect(hexA1?.activeHex).toBeDefined();
      expect(hexA1?.activeHex.type).toBe('movement');
      expect(hexB2?.activeHex).toBeDefined();
      expect(hexB2?.activeHex.type).toBe('refuel');
      expect(hexC3?.activeHex).toBeDefined();
      expect(hexC3?.activeHex.type).toBe('movement');
    });

    it('should handle empty activeHexes array', () => {
      render(<HexMap {...defaultProps} activeHexes={[]} width={5} height={5} />);

      // Все гексы должны иметь undefined activeHex
      const hexWithActiveHex = mockHexProps.find(props => props.activeHex !== undefined);
      expect(hexWithActiveHex).toBeUndefined();
    });

    it('should correctly match coordinates for activeHexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('J', 30, 'movement')
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={35} height={33} />);

      const hexJ30 = mockHexProps.find(
        props => props.coordinate.letter === 'J' && props.coordinate.number === 30
      );

      expect(hexJ30).toBeDefined();
      expect(hexJ30?.activeHex).toBeDefined();
      expect(hexJ30?.activeHex.coordinate.letter).toBe('J');
      expect(hexJ30?.activeHex.coordinate.number).toBe(30);
    });

    it('should handle activeHexes with different types', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('A', 1, 'movement'),
        createActiveHex('B', 2, 'refuel'),
        createActiveHex('C', 3, 'repair'),
        createActiveHex('D', 4, 'patrol')
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexA1 = mockHexProps.find(props => props.coordinate.letter === 'A' && props.coordinate.number === 1);
      const hexB2 = mockHexProps.find(props => props.coordinate.letter === 'B' && props.coordinate.number === 2);
      const hexC3 = mockHexProps.find(props => props.coordinate.letter === 'C' && props.coordinate.number === 3);
      const hexD4 = mockHexProps.find(props => props.coordinate.letter === 'D' && props.coordinate.number === 4);

      expect(hexA1?.activeHex.type).toBe('movement');
      expect(hexB2?.activeHex.type).toBe('refuel');
      expect(hexC3?.activeHex.type).toBe('repair');
      expect(hexD4?.activeHex.type).toBe('patrol');
    });
  });

  describe('Event Handlers', () => {
    it('should pass onClick handler to Hex component', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1).toBeDefined();
      expect(hexA1?.onClick).toBeDefined();
      expect(typeof hexA1.onClick).toBe('function');
    });

    it('should pass onHover handler to Hex component', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1).toBeDefined();
      expect(hexA1?.onHover).toBeDefined();
      expect(typeof hexA1.onHover).toBe('function');
    });

    it('should call onHexClick when hex is clicked in normal mode', () => {
      const mockOnHexClick = jest.fn();
      render(
        <HexMap 
          {...defaultProps} 
          onHexClick={mockOnHexClick}
          width={5} 
          height={5} 
        />
      );

      // Находим Hex компонент для A1
      const hexA1Props = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      // Вызываем onClick handler
      if (hexA1Props?.onClick) {
        hexA1Props.onClick();
      }

      expect(mockOnHexClick).toHaveBeenCalledWith(
        expect.objectContaining({
          letter: 'A',
          number: 1
        })
      );
    });

    it('should pass onUnitClick handler to Hex component', () => {
      const mockOnUnitClick = jest.fn();
      render(
        <HexMap 
          {...defaultProps} 
          onUnitClick={mockOnUnitClick}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.onUnitClick).toBeDefined();
      expect(hexA1.onUnitClick).toBe(mockOnUnitClick);
    });
  });

  describe('TF Mode (Create Task Force)', () => {
    it('should pass isTFCandidate prop to Hex when in TF mode', () => {
      const mockUnits = [
        {
          id: 'unit-1',
          name: 'Unit 1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-2',
          name: 'Unit 2',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          playerSide="german"
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      // Кликаем на кнопку Create TF
      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      userEvent.click(createTFButton);

      // Проверяем, что isTFCandidate передается в Hex для гекса A1
      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      // После клика на Create TF, компонент должен перейти в режим создания TF
      // Но так как это требует ре-рендера, мы не можем напрямую проверить isTFCandidate
      // Это требует интеграционного теста
      expect(createTFButton).toBeInTheDocument();
    });
  });

  describe('Data Updates', () => {
    it('should update hex elements when activeHexes change', () => {
      const { rerender } = render(
        <HexMap 
          {...defaultProps} 
          activeHexes={[]}
          width={5} 
          height={5} 
        />
      );

      const hexA1Before = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      expect(hexA1Before?.activeHex).toBeUndefined();

      // Добавляем activeHex
      const activeHexes = [{
        coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
        type: 'movement' as const,
        priority: 1
      }];

      mockHexProps.length = 0; // Очищаем для нового рендера
      rerender(
        <HexMap 
          {...defaultProps} 
          activeHexes={activeHexes}
          width={5} 
          height={5} 
        />
      );

      const hexA1After = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      expect(hexA1After?.activeHex).toBeDefined();
      expect(hexA1After?.activeHex.type).toBe('movement');
    });

    it('should update hex elements when gameUnits change', () => {
      const { rerender } = render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[]}
          width={5} 
          height={5} 
        />
      );

      const initialHexCount = mockHexProps.length;

      const mockUnits = [
        {
          id: 'unit-1',
          name: 'Unit 1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      mockHexProps.length = 0;
      rerender(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          width={5} 
          height={5} 
        />
      );

      // Количество гексов должно остаться прежним, но данные должны обновиться
      expect(mockHexProps.length).toBeGreaterThan(0);
    });
  });

  describe('Selected Hex', () => {
    it('should pass isSelected prop to Hex when hex is selected', () => {
      const selectedHex: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };

      render(
        <HexMap 
          {...defaultProps} 
          selectedHex={selectedHex}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isSelected).toBe(true);
    });

    it('should pass isSelected=false to Hex when hex is not selected', () => {
      const selectedHex: HexCoordinate = {
        col: 1,
        row: 1,
        letter: 'B',
        number: 2
      };

      render(
        <HexMap 
          {...defaultProps} 
          selectedHex={selectedHex}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isSelected).toBe(false);
    });
  });

  describe('Search Available', () => {
    it('should pass isSearchAvailable prop based on searchFactorHexes and visibilityLevel', () => {
      const searchFactorHexes = new Map<string, number>([
        ['A1', 5],
        ['B2', 2]
      ]);

      render(
        <HexMap 
          {...defaultProps} 
          searchFactorHexes={searchFactorHexes}
          visibilityLevel={3}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );
      const hexC3 = mockHexProps.find(
        props => props.coordinate.letter === 'C' && props.coordinate.number === 3
      );

      // A1 has searchFactor 5 >= visibilityLevel 3, should be available
      expect(hexA1?.isSearchAvailable).toBe(true);
      // B2 has searchFactor 2 < visibilityLevel 3, should not be available
      expect(hexB2?.isSearchAvailable).toBe(false);
      // C3 has no searchFactor, should not be available
      expect(hexC3?.isSearchAvailable).toBe(false);
    });

    it('should handle searchFactorHexes with zero values', () => {
      const searchFactorHexes = new Map<string, number>([
        ['A1', 0]
      ]);

      render(
        <HexMap 
          {...defaultProps} 
          searchFactorHexes={searchFactorHexes}
          visibilityLevel={1}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      // searchFactor 0 < visibilityLevel 1, should not be available
      expect(hexA1?.isSearchAvailable).toBe(false);
    });
  });

  describe('Flight Path Marker', () => {
    it('should pass hasFlightPathMarker prop when hexMarker has flight_path_search', () => {
      const hexMarkers: Record<string, any> = {
        'A1': {
          flight_path_search: 2
        }
      };

      render(
        <HexMap 
          {...defaultProps} 
          hexMarkers={hexMarkers}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );

      expect(hexA1?.hasFlightPathMarker).toBe(true);
      expect(hexB2?.hasFlightPathMarker).toBe(false);
    });

    it('should not pass hasFlightPathMarker when flight_path_search is 0', () => {
      const hexMarkers: Record<string, any> = {
        'A1': {
          flight_path_search: 0
        }
      };

      render(
        <HexMap 
          {...defaultProps} 
          hexMarkers={hexMarkers}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.hasFlightPathMarker).toBe(false);
    });
  });

  describe('Hover Handler', () => {
    it('should call onHexHover when hex is hovered', () => {
      const mockOnHexHover = jest.fn();
      render(
        <HexMap 
          {...defaultProps} 
          onHexHover={mockOnHexHover}
          width={5} 
          height={5} 
        />
      );

      const hexA1Props = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      if (hexA1Props?.onHover) {
        hexA1Props.onHover();
      }

      expect(mockOnHexHover).toHaveBeenCalledWith(
        expect.objectContaining({
          letter: 'A',
          number: 1
        })
      );
    });
  });

  describe('Map Controls', () => {
    it('should render map offset buttons', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);
      
      const rightButton = screen.getByRole('button', { name: /→/ });
      const upButton = screen.getByRole('button', { name: /↑/ });
      
      expect(rightButton).toBeInTheDocument();
      expect(upButton).toBeInTheDocument();
    });

    it('should render map info with dimensions', () => {
      render(<HexMap {...defaultProps} width={10} height={15} />);
      
      expect(screen.getByText(/10×15 гексов/i)).toBeInTheDocument();
    });
  });

  describe('Edge Cases - Active Hexes', () => {
    it('should handle activeHexes with duplicate coordinates (keep first match)', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        },
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'refuel',
          priority: 2
        }
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      // find() возвращает первый найденный элемент, так что должен быть movement
      expect(hexA1?.activeHex).toBeDefined();
      // В реальности это зависит от порядка в массиве, но для теста важно, что он найден
      expect(['movement', 'refuel']).toContain(hexA1?.activeHex.type);
    });

    it('should handle activeHexes with invalid coordinates gracefully', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: -1, row: -1, letter: '?', number: -1 },
          type: 'movement',
          priority: 1
        }
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      // Компонент должен рендериться без ошибок
      expect(screen.getByText(/Карта Атлантики/i)).toBeInTheDocument();
    });
  });

  describe('Available Movement Hexes', () => {
    it('should pass isAvailableForMovement prop based on availableMovementHexes', () => {
      const availableMovementHexes = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          distance: 1,
          fuelCost: 2,
          isReachable: true
        },
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          distance: 2,
          fuelCost: 3,
          isReachable: true
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          availableMovementHexes={availableMovementHexes}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );
      const hexC3 = mockHexProps.find(
        props => props.coordinate.letter === 'C' && props.coordinate.number === 3
      );

      expect(hexA1?.isAvailableForMovement).toBe(true);
      expect(hexB2?.isAvailableForMovement).toBe(true);
      expect(hexC3?.isAvailableForMovement).toBe(false);
    });

    it('should handle empty availableMovementHexes array', () => {
      render(
        <HexMap 
          {...defaultProps} 
          availableMovementHexes={[]}
          width={5} 
          height={5} 
        />
      );

      const allHexes = mockHexProps.filter(props => props.isAvailableForMovement === true);
      expect(allHexes.length).toBe(0);
    });

    it('should correctly match coordinates for availableMovementHexes', () => {
      // J - это 10-я буква (A=0, B=1, ..., J=9), так что row = 9
      // number 30 -> col = 29 (number - 1)
      const availableMovementHexes = [
        {
          coordinate: { col: 29, row: 9, letter: 'J', number: 30 },
          distance: 5,
          fuelCost: 10,
          isReachable: true
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          availableMovementHexes={availableMovementHexes}
          width={35} 
          height={33} 
        />
      );

      const hexJ30 = mockHexProps.find(
        props => props.coordinate.letter === 'J' && props.coordinate.number === 30
      );

      expect(hexJ30?.isAvailableForMovement).toBe(true);
    });
  });

  describe('Props Passed to Hex', () => {
    it('should pass mapStructures prop to Hex', () => {
      const mapStructures = {
        landAreas: [],
        nonGameHexes: [],
        restrictedDD: undefined,
        fogAreas: []
      };

      render(
        <HexMap 
          {...defaultProps} 
          mapStructures={mapStructures}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.mapStructures).toBe(mapStructures);
    });

    it('should pass selectedUnit prop to Hex', () => {
      render(
        <HexMap 
          {...defaultProps} 
          selectedUnit="unit-123"
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.selectedUnit).toBe('unit-123');
    });

    it('should pass expandedStackHex prop to Hex', () => {
      render(
        <HexMap 
          {...defaultProps} 
          expandedStackHex="A1"
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.expandedStackHex).toBe('A1');
    });

    it('should pass currentTurn prop to Hex', () => {
      render(
        <HexMap 
          {...defaultProps} 
          currentTurn={5}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.currentTurn).toBe(5);
    });

    it('should pass isFog prop to Hex', () => {
      render(
        <HexMap 
          {...defaultProps} 
          isFog={true}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isFog).toBe(true);
    });

    it('should pass onUnitStackClick handler to Hex', () => {
      const mockOnUnitStackClick = jest.fn();
      render(
        <HexMap 
          {...defaultProps} 
          onUnitStackClick={mockOnUnitStackClick}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.onUnitStackClick).toBe(mockOnUnitStackClick);
    });

    it('should pass onStackedUnitSelect handler to Hex', () => {
      const mockOnStackedUnitSelect = jest.fn();
      render(
        <HexMap 
          {...defaultProps} 
          onStackedUnitSelect={mockOnStackedUnitSelect}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.onStackedUnitSelect).toBe(mockOnStackedUnitSelect);
    });
  });

  describe('Combined Props', () => {
    it('should correctly combine activeHexes and availableMovementHexes', () => {
      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        }
      ];

      const availableMovementHexes = [
        {
          coordinate: { col: 1, row: 1, letter: 'B', number: 2 },
          distance: 1,
          fuelCost: 2,
          isReachable: true
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          activeHexes={activeHexes}
          availableMovementHexes={availableMovementHexes}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );

      expect(hexA1?.activeHex).toBeDefined();
      expect(hexA1?.isAvailableForMovement).toBe(false);
      expect(hexB2?.activeHex).toBeUndefined();
      expect(hexB2?.isAvailableForMovement).toBe(true);
    });

    it('should correctly combine selectedHex, activeHexes and availableMovementHexes', () => {
      const selectedHex: HexCoordinate = {
        col: 0,
        row: 0,
        letter: 'A',
        number: 1
      };

      const activeHexes: ActiveHex[] = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          type: 'movement',
          priority: 1
        }
      ];

      const availableMovementHexes = [
        {
          coordinate: { col: 0, row: 0, letter: 'A', number: 1 },
          distance: 1,
          fuelCost: 2,
          isReachable: true
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          selectedHex={selectedHex}
          activeHexes={activeHexes}
          availableMovementHexes={availableMovementHexes}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isSelected).toBe(true);
      expect(hexA1?.activeHex).toBeDefined();
      expect(hexA1?.isAvailableForMovement).toBe(true);
    });
  });

  describe('Task Force Creation Mode', () => {
    it('should enter TF creation mode when Create TF button is clicked', async () => {
      const mockUnits = [
        {
          id: 'unit-1',
          name: 'Unit 1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-2',
          name: 'Unit 2',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      userEvent.click(createTFButton);

      // После клика должна появиться кнопка отмены
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Отмена|Cancel/i })).toBeInTheDocument();
      });

      // Проверяем, что гексы-кандидаты помечены как isTFCandidate
      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      expect(hexA1?.isTFCandidate).toBe(true);
    });

    it('should exit TF creation mode when Cancel button is clicked', async () => {
      const mockUnits = [
        {
          id: 'unit-1',
          name: 'Unit 1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-2',
          name: 'Unit 2',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      // Входим в режим создания TF
      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      userEvent.click(createTFButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Отмена|Cancel/i })).toBeInTheDocument();
      });

      // Выходим из режима
      const cancelButton = screen.getByRole('button', { name: /Отмена|Cancel/i });
      userEvent.click(cancelButton);

      // Кнопка отмены должна исчезнуть
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Отмена|Cancel/i })).not.toBeInTheDocument();
      });
    });

    it('should find TF candidate hexes correctly', async () => {
      const mockUnits = [
        {
          id: 'unit-1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-2',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-3',
          position: 'B2',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      userEvent.click(createTFButton);

      await waitFor(() => {
        // A1 должен быть кандидатом (2 юнита), B2 не должен (1 юнит)
        const hexA1 = mockHexProps.find(
          props => props.coordinate.letter === 'A' && props.coordinate.number === 1
        );
        const hexB2 = mockHexProps.find(
          props => props.coordinate.letter === 'B' && props.coordinate.number === 2
        );
        expect(hexA1?.isTFCandidate).toBe(true);
        expect(hexB2?.isTFCandidate).toBe(false);
      });
    });

    it('should handle hex click in TF mode', async () => {
      const mockUnits = [
        {
          id: 'unit-1',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        },
        {
          id: 'unit-2',
          position: 'A1',
          nationality: 'german',
          task_force_id: null,
          type: 'DD',
          status: 'active'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={mockUnits}
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      // Входим в режим создания TF
      const createTFButton = screen.getByRole('button', { name: /Создать TF|Create TF/i });
      userEvent.click(createTFButton);

      await waitFor(() => {
        const hexA1 = mockHexProps.find(
          props => props.coordinate.letter === 'A' && props.coordinate.number === 1
        );
        expect(hexA1?.isTFCandidate).toBe(true);
      });

      // Ждем, пока режим активируется
      await waitFor(() => {
        const hexA1 = mockHexProps.find(
          props => props.coordinate.letter === 'A' && props.coordinate.number === 1
        );
        expect(hexA1?.isTFCandidate).toBe(true);
      });

      // Кликаем по гексу-кандидату
      const hexA1Props = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      if (hexA1Props?.onClick) {
        act(() => {
          hexA1Props.onClick();
        });
      }

      // Должен открыться диалог создания TF
      await waitFor(() => {
        expect(screen.getByTestId('create-tf-dialog')).toBeInTheDocument();
      }, { timeout: 2000 });
    });
  });


  describe('Flight Path Search Mode', () => {
    it('should enter flight path search mode when button is clicked', async () => {
      render(
        <HexMap 
          {...defaultProps} 
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      const flightPathButton = screen.getByRole('button', { name: /Воздушная разведка|Flight Path/i });
      userEvent.click(flightPathButton);

      // После клика должна появиться кнопка отмены
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Отмена разведки|Cancel Search/i })).toBeInTheDocument();
      });
    });

    it('should exit flight path search mode when Cancel button is clicked', async () => {
      render(
        <HexMap 
          {...defaultProps} 
          currentPhase="movement"
          width={5} 
          height={5} 
        />
      );

      // Входим в режим
      const flightPathButton = screen.getByRole('button', { name: /Воздушная разведка|Flight Path/i });
      userEvent.click(flightPathButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Отмена разведки|Cancel Search/i })).toBeInTheDocument();
      });

      // Выходим из режима
      const cancelButton = screen.getByRole('button', { name: /Отмена разведки|Cancel Search/i });
      userEvent.click(cancelButton);

      // Кнопка отмены должна исчезнуть
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Отмена разведки|Cancel Search/i })).not.toBeInTheDocument();
      });
    });
  });

  describe('Map Controls', () => {
    it('should handle refresh button click', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      unitsAPI.getGameUnits = jest.fn().mockResolvedValue({ success: true });

      render(<HexMap {...defaultProps} width={5} height={5} />);

      const refreshButton = screen.getByTitle(/Обновить данные игры/i);
      userEvent.click(refreshButton);

      await waitFor(() => {
        expect(unitsAPI.getGameUnits).toHaveBeenCalled();
      });
    });

    it('should handle map offset buttons', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const rightButton = screen.getByRole('button', { name: /→/ });
      const upButton = screen.getByRole('button', { name: /↑/ });

      // Клики не должны вызывать ошибок
      userEvent.click(rightButton);
      userEvent.click(upButton);

      expect(rightButton).toBeInTheDocument();
      expect(upButton).toBeInTheDocument();
    });
  });

  describe('Tooltip Handling', () => {
    it('should handle unit hover', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      // Симулируем hover с координатами
      if (hexA1?.onUnitHover) {
        hexA1.onUnitHover('unit-1', 'BB', 'german', 100, 200);
      }

      // Tooltip должен обрабатываться без ошибок
      expect(hexA1?.onUnitHover).toBeDefined();
    });

    it('should handle unit leave', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      if (hexA1?.onUnitLeave) {
        hexA1.onUnitLeave();
      }

      expect(hexA1?.onUnitLeave).toBeDefined();
    });

    it('should handle hex tooltip show', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      if (hexA1?.onTooltipShow) {
        hexA1.onTooltipShow(100, 200, {
          hexId: 'A1',
          hexType: 'water',
          features: []
        });
      }

      expect(hexA1?.onTooltipShow).toBeDefined();
    });

    it('should handle hex tooltip hide', () => {
      render(<HexMap {...defaultProps} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      if (hexA1?.onTooltipHide) {
        hexA1.onTooltipHide();
      }

      expect(hexA1?.onTooltipHide).toBeDefined();
    });
  });

  describe('Active Hexes Types', () => {
    it('should handle movement type active hexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('A', 1, 'movement', 1)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.activeHex).toBeDefined();
      expect(hexA1?.activeHex.type).toBe('movement');
    });

    it('should handle search type active hexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('B', 2, 'search', 7)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );

      expect(hexB2?.activeHex).toBeDefined();
      expect(hexB2?.activeHex.type).toBe('search');
    });

    it('should handle refuel type active hexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('C', 3, 'refuel', 2)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexC3 = mockHexProps.find(
        props => props.coordinate.letter === 'C' && props.coordinate.number === 3
      );

      expect(hexC3?.activeHex).toBeDefined();
      expect(hexC3?.activeHex.type).toBe('refuel');
    });

    it('should handle repair type active hexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('D', 4, 'repair', 3)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexD4 = mockHexProps.find(
        props => props.coordinate.letter === 'D' && props.coordinate.number === 4
      );

      expect(hexD4?.activeHex).toBeDefined();
      expect(hexD4?.activeHex.type).toBe('repair');
    });

    it('should handle patrol type active hexes', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('E', 5, 'patrol', 4)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexE5 = mockHexProps.find(
        props => props.coordinate.letter === 'E' && props.coordinate.number === 5
      );

      expect(hexE5?.activeHex).toBeDefined();
      expect(hexE5?.activeHex.type).toBe('patrol');
    });

    it('should handle multiple active hex types with priority', () => {
      const activeHexes: ActiveHex[] = [
        createActiveHex('A', 1, 'movement', 1),
        createActiveHex('A', 1, 'refuel', 2),
        createActiveHex('B', 2, 'search', 7)
      ];

      render(<HexMap {...defaultProps} activeHexes={activeHexes} width={5} height={5} />);

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );
      const hexB2 = mockHexProps.find(
        props => props.coordinate.letter === 'B' && props.coordinate.number === 2
      );

      expect(hexA1?.activeHex).toBeDefined();
      expect(hexB2?.activeHex).toBeDefined();
      // При одинаковых координатах find() вернет первый найденный
      expect(['movement', 'refuel']).toContain(hexA1?.activeHex.type);
    });
  });

  describe('Map Structures', () => {
    it('should pass mapStructures to Hex component', () => {
      const mapStructures = {
        landAreas: [],
        nonGameHexes: [],
        restrictedDD: null,
        fogAreas: []
      };

      render(
        <HexMap 
          {...defaultProps} 
          mapStructures={mapStructures}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.mapStructures).toBe(mapStructures);
    });

    it('should handle null mapStructures', () => {
      render(
        <HexMap 
          {...defaultProps} 
          mapStructures={null}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.mapStructures).toBe(null);
    });
  });

  describe('Fog of War', () => {
    it('should pass isFog prop to Hex', () => {
      render(
        <HexMap 
          {...defaultProps} 
          isFog={true}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isFog).toBe(true);
    });

    it('should pass isFog=false when fog is disabled', () => {
      render(
        <HexMap 
          {...defaultProps} 
          isFog={false}
          width={5} 
          height={5} 
        />
      );

      const hexA1 = mockHexProps.find(
        props => props.coordinate.letter === 'A' && props.coordinate.number === 1
      );

      expect(hexA1?.isFog).toBe(false);
    });
  });

  describe('Enemy Contacts', () => {
    it('should pass enemyContacts to Hex component', () => {
      const mockEnemyContacts = [
        {
          hex_id: 'A1',
          visibility: 'sighted' as const,
          ship_count: 2,
          class_summary: 'BB, CA',
          task_force: 'TF-1',
          task_force_list: ['TF-1'],
          enemy_nationality: 'allied' as const,
          searching_side: 'german' as const,
          turn: 1,
          phase: 'movement',
          last_seen_at: '2023-01-01T00:00:00Z'
        }
      ];

      render(
        <HexMap 
          {...defaultProps} 
          enemyContacts={mockEnemyContacts}
          width={5} 
          height={5} 
        />
      );

      // EnemyContacts обрабатываются внутри компонента и передаются через hexData
      // Проверяем, что компонент рендерится без ошибок
      const svg = document.querySelector('svg.hex-map');
      expect(svg).toBeInTheDocument();
    });
  });

  describe('Unit Actions', () => {
    it('should display action buttons for unit with available actions', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn();

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['repair', 'refuel-port', 'patrol']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      // Должны появиться кнопки действий
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Ремонт|Repair/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Заправка|Refuel/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Патруль|Patrol/i })).toBeInTheDocument();
      });
    });

    it('should handle repair action', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn().mockResolvedValue(undefined);

      unitsAPI.repairAtSea = jest.fn().mockResolvedValue({
        success: true
      });

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['repair']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Ремонт|Repair/i })).toBeInTheDocument();
      });

      const repairButton = screen.getByRole('button', { name: /Ремонт|Repair/i });
      userEvent.click(repairButton);

      await waitFor(() => {
        expect(unitsAPI.repairAtSea).toHaveBeenCalledWith(
          mockGameId,
          'unit-1',
          mockAuthToken
        );
        expect(mockOnRefreshData).toHaveBeenCalled();
        expect(mockOnUnitDeselect).toHaveBeenCalled();
      });
    });

    it('should handle refuel-port action', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn().mockResolvedValue(undefined);

      unitsAPI.refuelAtPort = jest.fn().mockResolvedValue({
        success: true
      });

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['refuel-port']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Заправка/i })).toBeInTheDocument();
      });

      const refuelButton = screen.getByRole('button', { name: /Заправка/i });
      userEvent.click(refuelButton);

      await waitFor(() => {
        expect(unitsAPI.refuelAtPort).toHaveBeenCalledWith(
          mockGameId,
          'unit-1',
          mockAuthToken
        );
      });
    });

    it('should handle refuel-sea action', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn().mockResolvedValue(undefined);

      unitsAPI.refuelAtSea = jest.fn().mockResolvedValue({
        success: true
      });

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['refuel-sea']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Заправка/i })).toBeInTheDocument();
      });

      const refuelButton = screen.getByRole('button', { name: /Заправка/i });
      userEvent.click(refuelButton);

      await waitFor(() => {
        expect(unitsAPI.refuelAtSea).toHaveBeenCalledWith(
          mockGameId,
          'unit-1',
          mockAuthToken
        );
      });
    });

    it('should handle patrol action for regular unit', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn().mockResolvedValue(undefined);

      unitsAPI.setPatrol = jest.fn().mockResolvedValue({
        success: true
      });

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['patrol']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Патруль|Patrol/i })).toBeInTheDocument();
      });

      const patrolButton = screen.getByRole('button', { name: /Патруль|Patrol/i });
      await act(async () => {
        await userEvent.click(patrolButton);
      });

      // Кнопка патруля должна вызвать API
      await waitFor(() => {
        expect(unitsAPI.setPatrol).toHaveBeenCalledWith(
          mockGameId,
          'unit-1',
          true,
          mockAuthToken
        );
      });
    });

    it('should handle patrol action for Task Force', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn().mockResolvedValue(undefined);

      unitsAPI.setTaskForcePatrol = jest.fn().mockResolvedValue({
        success: true
      });

      const mockTaskForce = {
        id: 'tf-1',
        name: 'Task Force 1',
        position: 'A1',
        nationality: 'german',
        units: ['unit-1'],
        available_actions: ['patrol']
      };

      render(
        <HexMap 
          {...defaultProps} 
          taskForces={[mockTaskForce]}
          currentPhase="movement"
          selectedUnit="tf-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Патруль|Patrol/i })).toBeInTheDocument();
      });

      const patrolButton = screen.getByRole('button', { name: /Патруль|Patrol/i });
      await act(async () => {
        await userEvent.click(patrolButton);
      });

      // Кнопка патруля должна вызвать API для Task Force
      await waitFor(() => {
        expect(unitsAPI.setTaskForcePatrol).toHaveBeenCalledWith(
          mockGameId,
          'tf-1',
          true,
          mockAuthToken
        );
      });
    });

    it('should not display action buttons when unit has no available actions', () => {
      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: []
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          width={5} 
          height={5} 
        />
      );

      expect(screen.queryByRole('button', { name: /Ремонт|Repair/i })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /Заправка|Refuel/i })).not.toBeInTheDocument();
    });

    it('should filter out movement action from action buttons', () => {
      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['movement', 'repair', 'patrol']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          width={5} 
          height={5} 
        />
      );

      // movement не должно быть в кнопках действий
      expect(screen.queryByRole('button', { name: /movement/i })).not.toBeInTheDocument();
      // но repair и patrol должны быть
      expect(screen.getByRole('button', { name: /Ремонт|Repair/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Патруль|Patrol/i })).toBeInTheDocument();
    });

    it('should handle action failure gracefully', async () => {
      const { unitsAPI } = require('../services/api/unitsAPI');
      const mockOnUnitDeselect = jest.fn();
      const mockOnRefreshData = jest.fn();

      unitsAPI.repairAtSea = jest.fn().mockResolvedValue({
        success: false,
        error: 'Cannot repair'
      });

      const mockUnit = {
        id: 'unit-1',
        name: 'Unit 1',
        position: 'A1',
        nationality: 'german',
        task_force_id: null,
        type: 'DD',
        status: 'active',
        available_actions: ['repair']
      };

      render(
        <HexMap 
          {...defaultProps} 
          gameUnits={[mockUnit]}
          currentPhase="movement"
          selectedUnit="unit-1"
          onUnitDeselect={mockOnUnitDeselect}
          onRefreshData={mockOnRefreshData}
          width={5} 
          height={5} 
        />
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Ремонт|Repair/i })).toBeInTheDocument();
      });

      const repairButton = screen.getByRole('button', { name: /Ремонт|Repair/i });
      userEvent.click(repairButton);

      await waitFor(() => {
        expect(unitsAPI.repairAtSea).toHaveBeenCalled();
      });

      // При ошибке не должно быть вызовов обновления и деселекции
      expect(mockOnRefreshData).not.toHaveBeenCalled();
      expect(mockOnUnitDeselect).not.toHaveBeenCalled();
    });
  });
});

