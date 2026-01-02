import React from 'react';
import { render, waitFor, fireEvent } from '@testing-library/react';
import { Hex } from './Hex';
import { HexCoordinate, HexData, MapStructure } from '../types/mapTypes';
import { Point } from '../utils/hexUtils';
import { ActiveHex } from '../utils/activeHexesUtils';

// Мокируем gameStore
jest.mock('../stores/gameStore', () => ({
  useGameStore: jest.fn(),
}));

// Мокируем утилиты
jest.mock('../utils/hexTooltipUtils', () => ({
  createHexTooltip: jest.fn((hexId: string, mapStructures: MapStructure | null) => ({
    hexId,
    hexType: 'water',
    features: [],
  })),
}));

jest.mock('../utils/activeHexesUtils', () => ({
  ACTIVE_HEX_CONFIGS: {
    movement: {
      type: 'movement',
      enabled: true,
      priority: 1,
      color: '#22C55E',
      opacity: 0.3,
      strokeColor: '#16A34A',
      strokeWidth: 2,
    },
    refuel: {
      type: 'refuel',
      enabled: true,
      priority: 2,
      color: '#F59E0B',
      opacity: 0.4,
      strokeColor: '#D97706',
      strokeWidth: 2,
    },
    search: {
      type: 'search',
      enabled: true,
      priority: 7,
      color: '#F97316',
      opacity: 0.3,
      strokeColor: '#EA580C',
      strokeWidth: 2,
    },
  },
}));

// Мокируем CSS файл
jest.mock('./Hex.css', () => ({}));

// Helper функции для создания mock-данных
const createMockCoordinate = (letter: string = 'A', number: number = 1): HexCoordinate => ({
  letter,
  number,
  col: number - 1,
  row: letter.charCodeAt(0) - 65,
});

const createMockPoint = (x: number = 0, y: number = 0): Point => ({ x, y });

const createMockHexData = (overrides?: Partial<HexData>): HexData => ({
  coordinate: createMockCoordinate(),
  type: 'water',
  isVisible: true,
  isHighlighted: false,
  hasUnit: false,
  units: [],
  taskForces: [],
  enemyContacts: [],
  weather: 'clear',
  hexType: 'water',
  isFogHex: false,
  ...overrides,
});

const createMockCenter = (): Point => createMockPoint(100, 100);

const createMockCorners = (center: Point = createMockCenter(), size: number = 20): Point[] => {
  // Создаем 6 углов гексагона
  const corners: Point[] = [];
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i;
    corners.push({
      x: center.x + size * Math.cos(angle),
      y: center.y + size * Math.sin(angle),
    });
  }
  return corners;
};

const createMockUnit = (overrides?: any) => ({
  id: 'unit-1',
  name: 'Bismarck',
  type: 'BB',
  nationality: 'german',
  fuel: 100,
  maxFuel: 100,
  last_move_turn: -1,
  no_movement_turns_left: 0,
  visibility: 'unknown',
  is_emergency_fuel: false,
  isEnemyContact: false,
  ...overrides,
});

const createMockTaskForce = (overrides?: any) => ({
  id: 'tf-1',
  name: 'Task Force 1',
  nationality: 'german',
  units: [],
  last_move_turn: -1,
  visibility: 'unknown',
  isEnemyContact: false,
  ...overrides,
});

const createMockActiveHex = (type: 'movement' | 'refuel' | 'search' = 'movement'): ActiveHex => ({
  coordinate: createMockCoordinate(),
  type,
  priority: 1,
});

const createMockMapStructure = (): MapStructure => ({
  landAreas: [],
  nonGameHexes: [],
});

// Mock callback функции
const createMockCallbacks = () => ({
  onClick: jest.fn(),
  onHover: jest.fn(),
  onUnitClick: jest.fn(),
  onUnitHover: jest.fn(),
  onUnitLeave: jest.fn(),
  onTooltipShow: jest.fn(),
  onTooltipHide: jest.fn(),
  onUnitStackClick: jest.fn(),
  onStackedUnitSelect: jest.fn(),
});

describe('Hex', () => {
  // Мокируем таймеры
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  // Мокируем getBoundingClientRect для tooltip тестов
  const mockGetBoundingClientRect = () => {
    const mockRect = {
      left: 100,
      top: 100,
      width: 50,
      height: 50,
      right: 150,
      bottom: 150,
      x: 100,
      y: 100,
      toJSON: jest.fn(),
    };

    Element.prototype.getBoundingClientRect = jest.fn(() => mockRect as DOMRect);
    return mockRect;
  };

  describe('Этап 2: Базовый рендеринг', () => {
    describe('2.1 Рендеринг основного гекса', () => {
      it('should render component without errors with minimal props', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should render SVG polygon with correct coordinates', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
        expect(polygon?.getAttribute('points')).toBeTruthy();
      });

      it('should correctly compute hexId from coordinates', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();
        const coordinate = createMockCoordinate('B', 5);

        const { container } = render(
          <Hex
            coordinate={coordinate}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        // hexId должен быть 'B5'
        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });
    });

    describe('2.2 Рендеринг разных типов гексов', () => {
      it('should render water hex correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({ hexType: 'water', type: 'water' });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should render land hex correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({ hexType: 'land', type: 'land' });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should render non_game hex correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({ hexType: 'non_game', type: 'non_game' });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });
    });

    describe('2.3 Рендеринг пустого гекса', () => {
      it('should render hex without units correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({ hasUnit: false, units: [], taskForces: [] });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
        // Не должно быть юнитов
        const unitImages = container.querySelectorAll('image[class*="unit"]');
        expect(unitImages.length).toBe(0);
      });

      it('should render hex without map structures correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            mapStructures={null}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });
    });
  });

  describe('Этап 3: Состояния гекса', () => {
    describe('3.1 Состояние выбора (isSelected)', () => {
      it('should apply red stroke when hex is selected', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
        expect(polygon?.getAttribute('stroke')).toBe('#ff0000');
        expect(polygon?.getAttribute('stroke-width')).toBe('3');
      });

      it('should have selected class when hex is selected', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        expect(hexGroup?.classList.contains('selected')).toBe(true);
      });

      it('should not have red stroke when hex is not selected', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon?.getAttribute('stroke')).not.toBe('#ff0000');
      });
    });

    describe('3.2 Состояние доступности для движения (isAvailableForMovement)', () => {
      it('should apply green stroke when hex is available for movement', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isAvailableForMovement={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon?.getAttribute('stroke')).toBe('#22C55E');
        expect(polygon?.getAttribute('stroke-width')).toBe('2');
      });

      it('should have available-for-movement class when hex is available', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isAvailableForMovement={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        expect(hexGroup?.classList.contains('available-for-movement')).toBe(true);
      });

      it('should prioritize isAvailableForMovement over activeHex', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();
        const activeHex = createMockActiveHex('refuel');

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isAvailableForMovement={true}
            activeHex={activeHex}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        // Должен быть зеленый (isAvailableForMovement), а не оранжевый (refuel)
        expect(polygon?.getAttribute('stroke')).toBe('#22C55E');
      });
    });

    describe('3.3 Состояние доступности для поиска (isSearchAvailable)', () => {
      it('should apply yellow stroke when hex is available for search', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isSearchAvailable={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon?.getAttribute('stroke')).toBe('#FBBF24');
        expect(polygon?.getAttribute('fill')).toBe('#FBBF24');
      });

      it('should have search-available class when hex is available', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isSearchAvailable={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        expect(hexGroup?.classList.contains('search-available')).toBe(true);
      });
    });
  });

  describe('Этап 4: Обработчики событий', () => {
    describe('4.1 Обработчик клика (onClick)', () => {
      it('should call onClick when hex is clicked', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        fireEvent.click(hexGroup!);

        expect(callbacks.onClick).toHaveBeenCalledTimes(1);
      });
    });

    describe('4.2 Обработчик наведения (onHover)', () => {
      it('should call onHover when mouse enters hex', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        fireEvent.mouseEnter(hexGroup!);

        // onHover вызывается в onMouseEnter обработчике
        expect(callbacks.onHover).toHaveBeenCalled();
      });
    });

    describe('4.3 Обработчики tooltip', () => {
      it('should call onTooltipShow after 2 seconds delay', async () => {
        mockGetBoundingClientRect();
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onTooltipShow={callbacks.onTooltipShow}
            mapStructures={createMockMapStructure()}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        fireEvent.mouseEnter(hexGroup!);

        // Tooltip не должен появиться сразу
        expect(callbacks.onTooltipShow).not.toHaveBeenCalled();

        // Перематываем время на 2 секунды
        jest.advanceTimersByTime(2000);

        await waitFor(() => {
          expect(callbacks.onTooltipShow).toHaveBeenCalledTimes(1);
        });
      });

      it('should call onTooltipHide when mouse leaves hex', () => {
        mockGetBoundingClientRect();
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onTooltipHide={callbacks.onTooltipHide}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        fireEvent.mouseEnter(hexGroup!);
        fireEvent.mouseLeave(hexGroup!);

        expect(callbacks.onTooltipHide).toHaveBeenCalledTimes(1);
      });

      it('should clear timeout when mouse leaves before 2 seconds', () => {
        mockGetBoundingClientRect();
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onTooltipShow={callbacks.onTooltipShow}
            mapStructures={createMockMapStructure()}
          />
        );

        const hexGroup = container.querySelector('g.hex');
        fireEvent.mouseEnter(hexGroup!);

        // Уходим до истечения 2 секунд
        jest.advanceTimersByTime(1000);
        fireEvent.mouseLeave(hexGroup!);
        jest.advanceTimersByTime(2000);

        // Tooltip не должен появиться
        expect(callbacks.onTooltipShow).not.toHaveBeenCalled();
      });
    });
  });

  describe('Этап 5: Рендеринг юнитов', () => {
    describe('5.1 Рендеринг одиночного юнита', () => {
      it('should render single unit correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit();
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
          unitId: unit.id,
          unitType: unit.type,
          unitSide: unit.nationality,
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const unitImage = container.querySelector('image.unit-icon');
        expect(unitImage).toBeInTheDocument();
      });

      it('should render unit background circle', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit();
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const backgroundCircle = container.querySelector('circle.unit-background');
        expect(backgroundCircle).toBeInTheDocument();
      });

      it('should render selected ring for selected unit', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            selectedUnit="unit-1"
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const selectedRing = container.querySelector('circle.unit-selected-ring');
        expect(selectedRing).toBeInTheDocument();
      });

      it('should call onUnitClick when unit is clicked', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const unitContainer = container.querySelector('g.unit-container');
        fireEvent.click(unitContainer!);

        // Компонент добавляет isTaskForce: false к юниту
        expect(callbacks.onUnitClick).toHaveBeenCalledWith('unit-1', expect.objectContaining({
          ...unit,
          isTaskForce: false,
        }));
      });

      it('should not call onUnitClick for enemy contact', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', isEnemyContact: true });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const unitContainer = container.querySelector('g.unit-container');
        fireEvent.click(unitContainer!);

        expect(callbacks.onUnitClick).not.toHaveBeenCalled();
      });
    });

    describe('5.2 Рендеринг Task Force', () => {
      it('should render Task Force correctly', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const taskForce = createMockTaskForce({ id: 'tf-1', name: 'TF Alpha' });
        const hexData = createMockHexData({
          hasUnit: true,
          taskForces: [taskForce],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const tfImage = container.querySelector('image.task-force-icon');
        expect(tfImage).toBeInTheDocument();
      });

      it('should display Task Force name', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const taskForce = createMockTaskForce({ id: 'tf-1', name: 'TF Alpha' });
        const hexData = createMockHexData({
          hasUnit: true,
          taskForces: [taskForce],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const tfName = container.querySelector('text.task-force-name');
        expect(tfName).toBeInTheDocument();
        expect(tfName?.textContent).toBe('TF Alpha');
      });

      it('should render selected ring for selected Task Force', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const taskForce = createMockTaskForce({ id: 'tf-1' });
        const hexData = createMockHexData({
          hasUnit: true,
          taskForces: [taskForce],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            selectedUnit="tf-1"
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitClick={callbacks.onUnitClick}
          />
        );

        const selectedRing = container.querySelector('circle.task-force-selected-ring');
        expect(selectedRing).toBeInTheDocument();
      });
    });

    describe('5.3 Рендеринг стека юнитов (свернутый)', () => {
      it('should render collapsed stack for multiple units', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const units = [
          createMockUnit({ id: 'unit-1' }),
          createMockUnit({ id: 'unit-2' }),
        ];
        const hexData = createMockHexData({
          hasUnit: true,
          units,
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitStackClick={callbacks.onUnitStackClick}
          />
        );

        const stackContainer = container.querySelector('g.unit-stack-container');
        expect(stackContainer).toBeInTheDocument();
      });

      it('should display unit count badge', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const units = [
          createMockUnit({ id: 'unit-1' }),
          createMockUnit({ id: 'unit-2' }),
          createMockUnit({ id: 'unit-3' }),
        ];
        const hexData = createMockHexData({
          hasUnit: true,
          units,
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitStackClick={callbacks.onUnitStackClick}
          />
        );

        const countText = container.querySelector('text.unit-count-text');
        expect(countText).toBeInTheDocument();
        expect(countText?.textContent).toBe('3');
      });

      it('should call onUnitStackClick when stack is clicked', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const units = [
          createMockUnit({ id: 'unit-1' }),
          createMockUnit({ id: 'unit-2' }),
        ];
        const hexData = createMockHexData({
          hasUnit: true,
          units,
        });
        const callbacks = createMockCallbacks();
        const coordinate = createMockCoordinate('A', 1);

        const { container } = render(
          <Hex
            coordinate={coordinate}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onUnitStackClick={callbacks.onUnitStackClick}
          />
        );

        const stackContainer = container.querySelector('g.unit-stack-container');
        fireEvent.click(stackContainer!);

        // Компонент добавляет isTaskForce: false к каждому юниту в стеке
        const expectedUnits = units.map(u => ({ ...u, isTaskForce: false }));
        expect(callbacks.onUnitStackClick).toHaveBeenCalledWith('A1', expect.arrayContaining(expectedUnits));
      });
    });

    describe('5.4 Рендеринг стека юнитов (развернутый)', () => {
      it('should render expanded stack with all units', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const units = [
          createMockUnit({ id: 'unit-1' }),
          createMockUnit({ id: 'unit-2' }),
        ];
        const hexData = createMockHexData({
          hasUnit: true,
          units,
        });
        const callbacks = createMockCallbacks();
        const coordinate = createMockCoordinate('A', 1);

        const { container } = render(
          <Hex
            coordinate={coordinate}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            expandedStackHex="A1"
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onStackedUnitSelect={callbacks.onStackedUnitSelect}
          />
        );

        const expandedStack = container.querySelector('g.expanded-unit-stack');
        expect(expandedStack).toBeInTheDocument();

        const stackedUnits = container.querySelectorAll('g.stacked-unit');
        expect(stackedUnits.length).toBe(2);
      });

      it('should call onStackedUnitSelect when unit in expanded stack is clicked', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        // Развернутый стек рендерится только при наличии нескольких юнитов
        const unit1 = createMockUnit({ id: 'unit-1' });
        const unit2 = createMockUnit({ id: 'unit-2' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit1, unit2],
        });
        const callbacks = createMockCallbacks();
        const coordinate = createMockCoordinate('A', 1);

        const { container } = render(
          <Hex
            coordinate={coordinate}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            expandedStackHex="A1"
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
            onStackedUnitSelect={callbacks.onStackedUnitSelect}
          />
        );

        // Проверяем, что развернутый стек рендерится
        const expandedStack = container.querySelector('g.expanded-unit-stack');
        expect(expandedStack).toBeInTheDocument();

        // Ищем stacked-unit внутри expanded stack
        const stackedUnits = expandedStack?.querySelectorAll('g.stacked-unit');
        expect(stackedUnits?.length).toBeGreaterThan(0);

        if (stackedUnits && stackedUnits.length > 0) {
          fireEvent.click(stackedUnits[0]);

          // Компонент передает item с isTaskForce: false
          expect(callbacks.onStackedUnitSelect).toHaveBeenCalledWith(
            expect.objectContaining({
              id: expect.any(String),
              isTaskForce: false,
            })
          );
        }
      });
    });
  });

  describe('Этап 6: Состояния юнитов', () => {
    describe('6.1 Функция getUnitState', () => {
      it('should return selected for selected unit', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            selectedUnit="unit-1"
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.selected');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return emergency-fuel for unit with emergency fuel', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', is_emergency_fuel: true });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.emergency-fuel');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return shadowed for unit with shadowed visibility', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', visibility: 'shadowed' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.shadowed');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return sighted for unit with sighted visibility', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', visibility: 'sighted' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.sighted');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return cannot-activate for unit with no available actions', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', available_actions: [] });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            currentTurn={1}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.cannot-activate');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return cannot-activate for unit with undefined available_actions', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', available_actions: undefined });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.cannot-activate');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return cannot-activate for unit with null available_actions', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ id: 'unit-1', available_actions: null });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.cannot-activate');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return idle for normal unit with available actions', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ 
          id: 'unit-1', 
          available_actions: ['movement', 'patrol'],
          is_activated: false 
        });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.idle');
        expect(unitContainer).toBeInTheDocument();
      });

      it('should return cannot-activate for activated unit', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ 
          id: 'unit-1', 
          is_activated: true,
          available_actions: [] // Активированные юниты не имеют доступных действий
        });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitContainer = container.querySelector('g.unit-container.cannot-activate');
        expect(unitContainer).toBeInTheDocument();
      });
    });
  });

  describe('Этап 7: Вспомогательные функции', () => {
    describe('7.1 Функция getUnitIcon', () => {
      it('should return correct path for naval units', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ type: 'BB', nationality: 'german' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitImage = container.querySelector('image.unit-icon');
        expect(unitImage?.getAttribute('href')).toContain('/assets/units/german/naval/BB.svg');
      });

      it('should return correct path for bomber', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ type: 'B', nationality: 'german' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitImage = container.querySelector('image.unit-icon');
        expect(unitImage?.getAttribute('href')).toContain('/assets/units/german/air/Bomber.svg');
      });

      it('should return correct path for recon', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = createMockUnit({ type: 'R', nationality: 'allied' });
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const unitImage = container.querySelector('image.unit-icon');
        expect(unitImage?.getAttribute('href')).toContain('/assets/units/allied/air/Recon.svg');
      });
    });
  });

  describe('Этап 8: Вражеские контакты', () => {
    describe('8.1 Рендеринг enemy contact', () => {
      it('should render enemy contact when present in hexData', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const enemyContact = {
          hex_id: 'A1',
          visibility: 'sighted' as const,
          ship_count: 2,
          class_summary: 'BB',
          task_force: 'нет',
          task_force_list: [],
          enemy_nationality: 'german' as const,
          searching_side: 'allied' as const,
          turn: 1,
          phase: 'search',
          last_seen_at: '2023-01-01',
        };
        const hexData = createMockHexData({
          enemyContacts: [enemyContact],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const enemyContactGroup = container.querySelector('g.enemy-contact');
        expect(enemyContactGroup).toBeInTheDocument();
      });

      it('should not render enemy contact when array is empty', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({
          enemyContacts: [],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const enemyContactGroup = container.querySelector('g.enemy-contact');
        expect(enemyContactGroup).not.toBeInTheDocument();
      });

      it('should prioritize shadowed contact over sighted', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const enemyContacts = [
          {
            hex_id: 'A1',
            visibility: 'sighted' as const,
            ship_count: 1,
            class_summary: 'CA',
            task_force: 'нет',
            task_force_list: [],
            enemy_nationality: 'german' as const,
            searching_side: 'allied' as const,
            turn: 1,
            phase: 'search',
            last_seen_at: '2023-01-01',
          },
          {
            hex_id: 'A1',
            visibility: 'shadowed' as const,
            ship_count: 2,
            class_summary: 'BB',
            task_force: 'нет',
            task_force_list: [],
            enemy_nationality: 'german' as const,
            searching_side: 'allied' as const,
            turn: 1,
            phase: 'search',
            last_seen_at: '2023-01-01',
          },
        ];
        const hexData = createMockHexData({
          enemyContacts,
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const enemyContactGroup = container.querySelector('g.enemy-contact.shadowed');
        expect(enemyContactGroup).toBeInTheDocument();
      });
    });

    describe('8.2 Рендеринг enemy contact как Task Force', () => {
      it('should render enemy contact as Task Force when task_force is present', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const enemyContact = {
          hex_id: 'A1',
          visibility: 'sighted' as const,
          ship_count: 2,
          class_summary: 'BB',
          task_force: 'TF Alpha',
          task_force_list: ['unit-1', 'unit-2'],
          enemy_nationality: 'german' as const,
          searching_side: 'allied' as const,
          turn: 1,
          phase: 'search',
          last_seen_at: '2023-01-01',
        };
        const hexData = createMockHexData({
          enemyContacts: [enemyContact],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const taskForceName = container.querySelector('text.task-force-name');
        expect(taskForceName).toBeInTheDocument();
        expect(taskForceName?.textContent).toBe('TF Alpha');
      });
    });
  });

  describe('Этап 9: Дополнительные элементы', () => {
    describe('9.1 Погодные эффекты', () => {
      it('should render storm weather effect when hexData.weather is storm', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({
          weather: 'storm',
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const weatherEffect = container.querySelector('path.weather-effect');
        expect(weatherEffect).toBeInTheDocument();
      });

      it('should not render weather effect for clear weather', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({
          weather: 'clear',
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const weatherEffect = container.querySelector('path.weather-effect');
        expect(weatherEffect).not.toBeInTheDocument();
      });
    });

    describe('9.2 Маркер пути полета (Flight Path)', () => {
      it('should render flight path marker when hasFlightPathMarker is true', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            hasFlightPathMarker={true}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const flightPathMarker = container.querySelector('g.flight-path-marker');
        expect(flightPathMarker).toBeInTheDocument();

        const markerImage = container.querySelector('g.flight-path-marker image');
        expect(markerImage).toBeInTheDocument();
        expect(markerImage?.getAttribute('href')).toBe('/assets/markers/FP.svg');
      });

      it('should not render flight path marker when hasFlightPathMarker is false', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            hasFlightPathMarker={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const flightPathMarker = container.querySelector('g.flight-path-marker');
        expect(flightPathMarker).not.toBeInTheDocument();
      });
    });
  });

  describe('Этап 10: Edge cases и граничные условия', () => {
    describe('10.1 Граничные значения props', () => {
      it('should work with minimal required props', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should work with undefined optional props', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            isAvailableForMovement={undefined}
            isSearchAvailable={undefined}
            activeHex={undefined}
            mapStructures={undefined}
            selectedUnit={undefined}
            expandedStackHex={undefined}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should work with null optional props', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData();
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            activeHex={null}
            mapStructures={null}
            selectedUnit={null}
            expandedStackHex={null}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should work with empty arrays', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const hexData = createMockHexData({
          units: [],
          taskForces: [],
          enemyContacts: [],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });
    });

    describe('10.2 Граничные значения данных', () => {
      it('should work with unit missing some fields', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const unit = { id: 'unit-1', type: 'BB' }; // Минимальные поля
        const hexData = createMockHexData({
          hasUnit: true,
          units: [unit],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });

      it('should work with Task Force without units', () => {
        const center = createMockCenter();
        const corners = createMockCorners(center, 20);
        const taskForce = createMockTaskForce({ units: undefined });
        const hexData = createMockHexData({
          hasUnit: true,
          taskForces: [taskForce],
        });
        const callbacks = createMockCallbacks();

        const { container } = render(
          <Hex
            coordinate={createMockCoordinate()}
            hexData={hexData}
            center={center}
            corners={corners}
            size={20}
            isSelected={false}
            onClick={callbacks.onClick}
            onHover={callbacks.onHover}
          />
        );

        const polygon = container.querySelector('polygon');
        expect(polygon).toBeInTheDocument();
      });
    });
  });
});

