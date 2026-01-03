// Гексагональная карта для игры Bismarck Chase
// Использует алгоритмы из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

import React, { useState, useEffect, useMemo } from 'react';
import { Hex } from './Hex';
import Tooltip from './Tooltip';
import CreateTaskForceDialog from './CreateTaskForceDialog';
import { HexCoordinate, HexData, coordinateToOffset, offsetToCoordinate, MapStructure, EnemyContactSummary } from '../types/mapTypes';
import { MovementHex } from '../utils/movementUtils';
import { ActiveHex } from '../utils/activeHexesUtils';
import { 
  Point, OffsetCoord, offsetToPixel, offsetPolygonCorners, calculateMapSize, MAP_CONSTANTS, getCubeNeighbors
} from '../utils/hexUtils';
import { phaseAPI } from '../services/api/phaseAPI';
import { unitsAPI, EnemyContact } from '../services/api/unitsAPI';
import { gameAPI } from '../services/api/gameAPI';
import { searchAPI, HexMarkers } from '../services/api/searchAPI';
import './HexMap.css';

interface HexMapProps {
  width?: number;
  height?: number;
  onHexClick?: (hex: HexCoordinate) => void;
  onHexHover?: (hex: HexCoordinate) => void;
  onUnitClick?: (unitId: string, unitData: any) => void;
  selectedHex?: HexCoordinate | null;
  playerSide?: 'german' | 'allied';
  availableMovementHexes?: MovementHex[];
  activeHexes?: ActiveHex[];
  gameUnits?: any[]; // Единый источник данных юнитов
  taskForces?: any[]; // Данные Task Forces  
  enemyContacts?: EnemyContact[];
  mapStructures?: MapStructure | null;
  selectedUnit?: string | null;
  expandedStackHex?: string | null;
  currentTurn?: number;
  gameId?: string;
  authToken?: string | null;
  onRefreshData?: () => void; // Callback для обновления данных в родительском компоненте
  onUnitStackClick?: (hexId: string, units: any[]) => void;
  onTaskForceClick?: (taskForceId: string, taskForceData: any) => void;
  onStackedUnitSelect?: (unit: any) => void;
  onRefuelAllShips?: () => void;
  onCompletePhase?: () => void;
  onStartFirstTurn?: () => void;
  isRefuelDisabled?: boolean;
  isCompletePhaseDisabled?: boolean;
  isStartFirstTurnVisible?: boolean;
  currentPhase?: string;
  searchFactorHexes?: Map<string, number>;
  visibilityLevel?: number;
  hexMarkers?: Record<string, HexMarkers>;
  isFog?: boolean;
  onUnitDeselect?: () => void; // Callback для сброса выбора юнита
}

const HexMap: React.FC<HexMapProps> = ({
  width = MAP_CONSTANTS.HEX_GRID_WIDTH, // 35.5 гексов по горизонтали
  height = MAP_CONSTANTS.HEX_GRID_HEIGHT, // 33 гекса по вертикали (A-AH)
  onHexClick,
  onHexHover,
  onUnitClick,
  selectedHex,
  playerSide = 'allied',
  availableMovementHexes = [],
  activeHexes = [],
  gameUnits = [],
  taskForces = [],
  enemyContacts = [],
  mapStructures = null,
  selectedUnit = null,
  expandedStackHex = null,
  currentTurn = 0,
  gameId,
  authToken,
  onRefreshData,
  onUnitStackClick,
  onTaskForceClick,
  onStackedUnitSelect,
  onRefuelAllShips,
  onCompletePhase,
  onStartFirstTurn,
  isRefuelDisabled = false,
  isCompletePhaseDisabled = false,
  isStartFirstTurnVisible = false,
  currentPhase = 'setup',
  searchFactorHexes = new Map<string, number>(),
  visibilityLevel = 1,
  hexMarkers = {},
  isFog = false,
  onUnitDeselect
}) => {
  const [mapOffset, setMapOffset] = useState({ x: 0, y: 0 });
  const [hexRadius] = useState(MAP_CONSTANTS.DEFAULT_HEX_RADIUS); // Стандартный радиус гекса
  const [isCreateTFMode, setIsCreateTFMode] = useState(false);
  const [tfCandidateHexes, setTfCandidateHexes] = useState<string[]>([]);
  const [showTFDialog, setShowTFDialog] = useState(false);
  const [selectedTFHex, setSelectedTFHex] = useState<string | null>(null);
  const [isFlightPathSearchMode, setIsFlightPathSearchMode] = useState(false);
  
  const [tooltip, setTooltip] = useState<{
    show: boolean;
    unitId: string;
    unitType: string;
    unitSide: string;
    x: number;
    y: number;
  } | null>(null);

  // Состояние для подсказки гекса
  const [hexTooltip, setHexTooltip] = useState<{
    visible: boolean;
    x: number;
    y: number;
    content: {
      hexId: string;
      hexType: string;
      features: string[];
    };
  } | null>(null);

  // Функция обновления данных игры
  const handleRefresh = async () => {
    if (!gameId || !playerSide || !authToken) {
      console.warn('Missing required data for refresh:', { gameId, playerSide, authToken: !!authToken });
      return;
    }

    try {
      console.log('🔄 Refreshing game data...');
      
      // Удалены вызовы phaseAPI.getCurrentPhase и gameEventAPI.getGameEvents
      // Информация о текущей фазе и событиях теперь приходит через GameModel
      
      // Обновляем юниты (GameModel содержит информацию о текущей фазе и событиях)
      console.log('⚔️ Fetching game units...');
      await unitsAPI.getGameUnits(gameId, authToken);
      
      // 4. Вызываем callback для обновления данных в родительском компоненте
      if (onRefreshData) {
        onRefreshData();
      }
      
      // 5. Уведомляем GameLog о необходимости обновления событий
      console.log('📢 Triggering GameLog refresh...');
      window.dispatchEvent(new CustomEvent('gameLogRefresh'));
      
      console.log('✅ Game data refreshed successfully');
    } catch (error) {
      console.error('❌ Error refreshing game data:', error);
    }
  };

  // Функция для определения гексов-кандидатов для создания TF
  const findTFCandidateHexes = (): string[] => {
    const hexUnitsMap = new Map<string, any[]>();
    
    // Собираем юниты по гексам
    gameUnits.forEach(unit => {
      if (unit.position && unit.nationality === playerSide && !unit.task_force_id) {
        const hexId = unit.position;
        if (!hexUnitsMap.has(hexId)) {
          hexUnitsMap.set(hexId, []);
        }
        hexUnitsMap.get(hexId)!.push(unit);
      }
    });
    
    // Добавляем TF в те же гексы
    taskForces.forEach(tf => {
      if (tf.position && tf.nationality === playerSide) {
        const hexId = tf.position;
        if (!hexUnitsMap.has(hexId)) {
          hexUnitsMap.set(hexId, []);
        }
        hexUnitsMap.get(hexId)!.push(tf);
      }
    });
    
    // Выбираем гексы-кандидаты:
    // 1) гекс с более чем 1 объектом (юниты/TF) своей стороны
    // 2) гекс, где есть хотя бы один свой TF (даже если он один)
    const candidates: string[] = [];
    hexUnitsMap.forEach((units, hexId) => {
      const hasMultipleObjects = units.length > 1;
      const hasAtLeastOneTF = units.some(obj => Array.isArray((obj as any).units));

      if (hasMultipleObjects || hasAtLeastOneTF) {
        candidates.push(hexId);
        console.log('✅ Added hex as candidate:', hexId, 'with', units.length, 'objects, hasAtLeastOneTF:', hasAtLeastOneTF);
      }
    });
    
    console.log('🎯 Final candidates:', candidates);
    return candidates;
  };

  // Обработчик кнопки создания TF
  const handleCreateTFClick = () => {
    console.log('🚢 handleCreateTFClick called');
    const candidates = findTFCandidateHexes();
    console.log('🎯 TF candidates found:', candidates);
    setTfCandidateHexes(candidates);
    setIsCreateTFMode(true);
  };

  // Выход из режима создания TF
  const handleCancelCreateTF = () => {
    setIsCreateTFMode(false);
    setTfCandidateHexes([]);
  };

  // Обработчик кнопки воздушной разведки
  const handleFlightPathSearchClick = () => {
    setIsFlightPathSearchMode(true);
  };

  // Выход из режима воздушной разведки
  const handleCancelFlightPathSearch = () => {
    setIsFlightPathSearchMode(false);
  };

  // Обработчик клика по гексу в режиме воздушной разведки
  const handleHexClickInFlightPathSearchMode = async (hexId: string) => {
    if (!isFlightPathSearchMode || !gameId || !authToken) {
      return;
    }

    try {
      const response = await searchAPI.addHexMarker(gameId, hexId, 'flight_path_search', authToken);
      if (response.success) {
        // Обновляем данные с сервера - маркер уже добавлен в GameModel
        if (onRefreshData) {
          onRefreshData();
        }
      } else {
        console.error('Failed to add hex marker:', response.error);
      }
    } catch (error: any) {
      console.error('Error adding hex marker:', error);
    }
  };

  // Обработчик клика по гексу в режиме TF
  const handleHexClickInTFMode = (hexId: string) => {
    if (isCreateTFMode && tfCandidateHexes.includes(hexId)) {
      setSelectedTFHex(hexId);
      setShowTFDialog(true);
    }
  };

  // Обработчик создания TF
  const handleCreateTF = async (selectedUnitIds: string[]) => {
    if (!gameId || !authToken) {
      return;
    }
    
    try {
      const response = await gameAPI.createTaskForce(gameId, {
        unitIds: selectedUnitIds,
        formation: 'line', // Используем стандартную формацию 'line'
        nationality: playerSide, // Передаем сторону игрока для правильного именования
        existingTaskForces: taskForces, // Передаем существующие TF для правильной нумерации
      });
      
      if (response.success) {
        // Обновить данные
        if (onRefreshData) {
          onRefreshData();
        }
      }
    } catch (error: any) {
      // Ошибка создания TF - игнорируем
    } finally {
      setShowTFDialog(false);
      setSelectedTFHex(null);
      handleCancelCreateTF();
    }
  };

  // Обработчик добавления юнита к существующему TF
  const handleAddToExistingTF = async (taskForceId: string, unitId: string) => {
    if (!gameId || !authToken) {
      return;
    }
    
    try {
      const response = await gameAPI.addUnitToTaskForce(gameId, {
        taskForceId: taskForceId,
        unitId: unitId,
      });
      
      if (response.success) {
        // Обновить данные
        if (onRefreshData) {
          onRefreshData();
        }
      }
    } catch (error: any) {
      // Ошибка добавления юнита к Task Force - игнорируем
    } finally {
      setShowTFDialog(false);
      setSelectedTFHex(null);
      handleCancelCreateTF();
    }
  };

  // Обработчик удаления юнита из существующего TF
  const handleRemoveFromExistingTF = async (taskForceId: string, unitId: string) => {
    if (!gameId || !authToken) {
      return;
    }
    try {
      const response = await gameAPI.removeUnitFromTaskForce(gameId, {
        taskForceId,
        unitId,
      });
      if (response.success) {
        if (onRefreshData) onRefreshData();
      }
    } catch (error: any) {
      // Ошибка удаления юнита из Task Force - игнорируем
    }
  };

  // Генерируем координаты гексов
  const hexes = useMemo(() => {
    const newHexes = new Map<string, HexData>();
    const contactsByHex = new Map<string, EnemyContactSummary[]>();

    enemyContacts.forEach(contact => {
      if (!contact || !contact.hex_id) {
        return;
      }
      const existing = contactsByHex.get(contact.hex_id) || [];
      existing.push(contact);
      contactsByHex.set(contact.hex_id, existing);
    });
    
    // Создаем гексы используя offset координаты (col, row)
    for (let row = 0; row < height; row++) {
      for (let col = 0; col < width; col++) {
        // Генерируем правильные буквы: A-Z, затем AA-AH
        let letter: string;
        if (row < 26) {
          // A, B, C, ..., Z (0-25)
          letter = String.fromCharCode(65 + row);
        } else {
          // AA, AB, AC, ..., AH (26-33)
          const secondLetterIndex = row - 26;
          letter = 'A' + String.fromCharCode(65 + secondLetterIndex);
        }
        const number = col + 1; // 1, 2, 3, ..., 35
        
        const coordinate: HexCoordinate = {
          letter: letter,
          number: number,
          col: col,
          row: row
        };
        
        const hexId = `${letter}${number}`;
        
        // Находим Task Forces в этом гексе
        const taskForcesInHex = taskForces.filter(tf => {
          if (!tf.position || tf.position.trim() === '') return false;
          
          const match = tf.position.match(/^([A-Z]+)(\d+)$/);
          if (!match) return false;
          
          const tfLetter = match[1];
          const tfNumber = parseInt(match[2]);
          
          let tfRow: number;
          if (tfLetter.length === 1) {
            tfRow = tfLetter.charCodeAt(0) - 65;
          } else if (tfLetter.length === 2 && tfLetter.startsWith('A')) {
            tfRow = 26 + (tfLetter.charCodeAt(1) - 65);
          } else {
            return false;
          }
          
          return tfRow === row && (tfNumber - 1) === col;
        });

        // Группируем юниты по позиции, исключая те что в Task Forces
        const unitsInHex = gameUnits.filter(unit => {
          if (!unit.position || unit.position.trim() === '') return false;
          
          // Если юнит в Task Force, не показываем его отдельно
          if (unit.task_force_id) return false;
          
          const match = unit.position.match(/^([A-Z]+)(\d+)$/);
          if (!match) return false;
          
          const unitLetter = match[1];
          const unitNumber = parseInt(match[2]);
          
          let unitRow: number;
          if (unitLetter.length === 1) {
            // A, B, C, ..., Z (0-25)
            unitRow = unitLetter.charCodeAt(0) - 65;
          } else if (unitLetter.length === 2 && unitLetter.startsWith('A')) {
            // AA, AB, AC, ..., AH (26-33)
            unitRow = 26 + (unitLetter.charCodeAt(1) - 65);
          } else {
            return false;
          }
          
          return unitRow === row && (unitNumber - 1) === col;
        });

        const hasUnit = unitsInHex.length > 0;
        const hasTaskForce = taskForcesInHex.length > 0;
        
        // Приоритет отображения: сначала Task Forces, затем отдельные юниты
        const unitId = hasTaskForce ? taskForcesInHex[0].id : (hasUnit ? unitsInHex[0].id : null);
        const unitType = hasTaskForce ? 'TF' : (hasUnit ? unitsInHex[0].type : null);
        const unitSide = hasTaskForce 
          ? (taskForcesInHex[0].nationality === 'german' ? 'german' : 'allied')
          : (hasUnit ? (unitsInHex[0].nationality === 'german' ? 'german' : 'allied') : null);
        
        // Определяем тип гекса на основе структур карты
        let hexType: 'water' | 'land' | 'non_game' = 'water';
        let isRestrictedDD = false;
        let isFogHex = false;
        
        if (mapStructures) {
          // Проверяем неигровые гексы
          for (const nonGame of mapStructures.nonGameHexes) {
            if (nonGame.hexIds.includes(hexId)) {
              hexType = 'non_game';
              break;
            }
          }
          
          // Проверяем сухопутные гексы (только если не неигровой)
          if (hexType === 'water') {
            for (const landArea of mapStructures.landAreas) {
              if (landArea.hexIds.includes(hexId)) {
                hexType = 'land';
                break;
              }
            }
          }
          
          // Проверяем ограничения для немецких DD
          if (mapStructures.restrictedDD && mapStructures.restrictedDD.hexIds.includes(hexId)) {
            isRestrictedDD = true;
          }
          
          // Проверяем туманные гексы
          if (mapStructures.fogAreas && mapStructures.fogAreas.length > 0) {
            for (const fogArea of mapStructures.fogAreas) {
              if (fogArea.hexIds && fogArea.hexIds.includes(hexId)) {
                isFogHex = true;
                break;
              }
            }
          }
        }
        
        // Отладка для туманных гексов
        if (isFogHex && isFog) {
        }
        
        newHexes.set(hexId, {
          coordinate,
          type: 'water', // Визуальный тип остается водой
          isVisible: true,
          isHighlighted: false,
          hasUnit: hasUnit || hasTaskForce,
          unitId,
          unitType,
          unitSide,
          units: unitsInHex, // Массив отдельных юнитов в гексе
          taskForces: taskForcesInHex, // Массив Task Forces в гексе
          weather: 'clear',
          hexType,
          isRestrictedDD,
          isFogHex,
          enemyContacts: contactsByHex.get(hexId) || []
        });
      }
    }
    
    return newHexes;
  }, [width, height, gameUnits, taskForces, enemyContacts, mapStructures, isFog]);

  // Обработчики событий
  const handleHexClick = (coordinate: HexCoordinate) => {
    const hexId = `${coordinate.letter}${coordinate.number}`;
    
    // Если в режиме воздушной разведки, обрабатываем клик
    if (isFlightPathSearchMode) {
      handleHexClickInFlightPathSearchMode(hexId);
      return;
    }
    
    // Если в режиме создания TF, проверяем клик по кандидату
    if (isCreateTFMode) {
      handleHexClickInTFMode(hexId);
      return;
    }
    
    // Проверяем, является ли гекс доступным для движения
    const isAvailableForMovement = availableMovementHexes.some(hex => 
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    );
    
    
    if (onHexClick) {
      onHexClick(coordinate);
    }
  };

  const handleHexHover = (coordinate: HexCoordinate) => {
    if (onHexHover) {
      onHexHover(coordinate);
    }
  };

  // Обработчики для tooltip
  const handleUnitHover = (unitId: string, unitType: string, unitSide: string, x: number, y: number) => {
    // Конвертируем координаты мыши в координаты относительно SVG
    const svgRect = document.querySelector('.hex-map')?.getBoundingClientRect();
    if (svgRect) {
      const relativeX = x - svgRect.left;
      const relativeY = y - svgRect.top;
      
      setTooltip({
        show: true,
        unitId,
        unitType,
        unitSide,
        x: relativeX,
        y: relativeY
      });
    }
  };

  const handleUnitLeave = () => {
    setTooltip(null);
  };

  // Обработчики для подсказки гекса
  const handleHexTooltipShow = (x: number, y: number, content: { hexId: string; hexType: string; features: string[] }) => {
    setHexTooltip({
      visible: true,
      x,
      y,
      content
    });
  };

  const handleHexTooltipHide = () => {
    setHexTooltip(null);
  };

  // Вычисляем размеры SVG с использованием универсальной функции
  const { width: svgWidth, height: svgHeight } = calculateMapSize(width, height, hexRadius);

  // Функция для получения описания юнита
  const getUnitDescription = (unitType: string, unitId: string, unitSide: string) => {
    const sideFlag = unitSide === 'german' ? '🇩🇪' : '🇬🇧';
    const sideName = unitSide === 'german' ? 'Германия' : 'Союзники';
    
    const typeNames: { [key: string]: string } = {
      'BB': 'Линкор',
      'BC': 'Линейный крейсер',
      'CV': 'Авианосец',
      'CA': 'Тяжелый крейсер',
      'CL': 'Легкий крейсер',
      'DD': 'Эсминец',
      'CG': 'Береговая охрана',
      'TK': 'Танкер',
      'B': 'Бомбардировщик',
      'R': 'Разведчик',
      'RE': 'Разведчик (долгий полет)'
    };
    
    const typeName = typeNames[unitType] || unitType;
    
    return `${sideFlag} ${sideName} ${typeName}\n${unitId}`;
  };

  // Рендерим гексы
  // Используем useMemo для пересчета при изменении маркеров
  const hexElements = useMemo(() => {
    const elements: React.JSX.Element[] = [];
    
    hexes.forEach((hexData, hexId) => {
      const { coordinate } = hexData;
      
      // Преобразуем координаты в offset систему
      const offsetCoord = coordinateToOffset(coordinate);
      
      // Получаем позицию центра гекса
      const center = offsetToPixel(offsetCoord, hexRadius);
      
      // Получаем углы гекса для отрисовки
      const corners = offsetPolygonCorners(offsetCoord, hexRadius);
      
      const isSelected = selectedHex && 
        selectedHex.letter === coordinate.letter && 
        selectedHex.number === coordinate.number;

      // Проверяем, является ли этот гекс доступным для движения
      const isAvailableForMovement = availableMovementHexes.some(
        movementHex => 
          movementHex.coordinate.col === coordinate.col && 
          movementHex.coordinate.row === coordinate.row
      );
      
      // Проверяем, достаточны ли факторы поиска для обнаружения в этом гексе
      // Используем данные из пропсов searchFactorHexes
      const hexSearchFactors = searchFactorHexes.get(hexId) || 0;
      const isSearchAvailable = hexSearchFactors > 0 && hexSearchFactors >= visibilityLevel;
      
      // Проверяем, является ли этот гекс активным
      const activeHex = activeHexes.find(
        hex => 
          hex.coordinate.col === coordinate.col && 
          hex.coordinate.row === coordinate.row
      );

      // Проверяем, есть ли маркер пути полета в этом гексе (используем данные из GameModel)
      const hexMarkerData = hexMarkers?.[hexId];
      const flightPathSearchCount = hexMarkerData?.flight_path_search || 0;
      const hasFlightPathMarker = flightPathSearchCount > 0;

      // Используем ключ с маркером, чтобы React обновлял компонент при изменении маркеров
      const markerKey = hasFlightPathMarker ? `marker-${flightPathSearchCount}` : 'no-marker';
      elements.push(
        <Hex
          key={`${hexId}-${markerKey}`}
          coordinate={coordinate}
          hexData={hexData}
          center={center}
          corners={corners}
          size={hexRadius}
          isSelected={!!isSelected}
          isAvailableForMovement={isAvailableForMovement}
          isSearchAvailable={isSearchAvailable}
          activeHex={activeHex}
          mapStructures={mapStructures}
          selectedUnit={selectedUnit}
          expandedStackHex={expandedStackHex}
          currentTurn={currentTurn}
          isTFCandidate={isCreateTFMode && tfCandidateHexes.includes(hexId)}
          hasFlightPathMarker={hasFlightPathMarker}
          isFog={isFog}
          onClick={() => {
            const hexId = `${coordinate.letter}${coordinate.number}`;
            // Проверяем режимы в порядке приоритета
            if (isFlightPathSearchMode) {
              handleHexClickInFlightPathSearchMode(hexId);
            } else if (isCreateTFMode) {
              handleHexClickInTFMode(hexId);
            } else if (onHexClick) {
              onHexClick(coordinate);
            }
          }}
          onHover={() => handleHexHover(coordinate)}
          onUnitClick={onUnitClick}
          onUnitHover={handleUnitHover}
          onUnitLeave={handleUnitLeave}
          onTooltipShow={handleHexTooltipShow}
          onTooltipHide={handleHexTooltipHide}
          onUnitStackClick={onUnitStackClick}
          onStackedUnitSelect={onStackedUnitSelect}
        />
      );
    });
    
    return elements;
  }, [hexes, hexMarkers, hexRadius, selectedHex, availableMovementHexes, searchFactorHexes, visibilityLevel, activeHexes, mapStructures, selectedUnit, expandedStackHex, currentTurn, isCreateTFMode, tfCandidateHexes, onHexClick, onUnitClick, onUnitStackClick, onStackedUnitSelect, isFlightPathSearchMode, handleHexClickInFlightPathSearchMode, handleHexClickInTFMode]);

  return (
    <div className="hex-map-container">
      <div className="map-info">
        <h3>Карта Атлантики</h3>
        <p>Размер: {width}×{height} гексов</p>
      </div>
      
      <div className="map-controls">
        <button onClick={handleRefresh} title="Обновить данные игры">
          🔄
        </button>
        <button onClick={() => setMapOffset({ x: mapOffset.x + 50, y: mapOffset.y })}>
          →
        </button>
        <button onClick={() => setMapOffset({ x: mapOffset.x, y: mapOffset.y - 50 })}>
          ↑
        </button>
        <button onClick={() => setMapOffset({ x: mapOffset.x, y: mapOffset.y + 50 })}>
          ↓
        </button>
        
        {/* Кнопки действий для выбранного юнита или Task Force */}
        {selectedUnit && currentPhase === 'movement' && !isCreateTFMode && !isFlightPathSearchMode && (() => {
          // Сначала проверяем, является ли выбранный элемент юнитом
          let unit = gameUnits.find(u => u.id === selectedUnit);
          let isTaskForce = false;
          
          // Если не найден в gameUnits, проверяем, является ли это Task Force
          if (!unit) {
            const taskForce = taskForces.find(tf => tf.id === selectedUnit);
            if (taskForce) {
              isTaskForce = true;
              // Преобразуем Task Force в формат, совместимый с unit для отображения действий
              unit = {
                id: taskForce.id,
                available_actions: taskForce.available_actions || [],
                is_activated: taskForce.is_activated || false
              } as any;
            }
          }
          
          // #region agent log
          fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({location:'HexMap.tsx:790',message:'Checking unit/taskforce actions display',data:{selectedUnit,unitFound:!!unit,isTaskForce,hasAvailableActions:!!(unit?.available_actions),availableActionsCount:unit?.available_actions?.length||0,availableActions:unit?.available_actions},timestamp:Date.now(),sessionId:'debug-session',runId:'run2',hypothesisId:'C'})}).catch(()=>{});
          // #endregion
          
          if (!unit || !unit.available_actions || unit.available_actions.length === 0) {
            return null;
          }
          
          const getActionLabel = (action: string) => {
            const labels: { [key: string]: string } = {
              'repair': '🔧 Ремонт',
              'refuel-port': '⛽ Заправка в порту',
              'refuel-sea': '⛽ Заправка в море',
              'patrol': '🛡️ Патруль'
            };
            return labels[action] || action;
          };

          const handleAction = async (action: string) => {
            if (!gameId || !authToken || !selectedUnit) return;
            
            try {
              let result;
              if (isTaskForce) {
                // Обработка действий для Task Force
                switch (action) {
                  case 'patrol':
                    result = await unitsAPI.setTaskForcePatrol(gameId, selectedUnit, true, authToken);
                    break;
                  default:
                    return;
                }
              } else {
                // Обработка действий для обычного юнита
                switch (action) {
                  case 'repair':
                    result = await unitsAPI.repairAtSea(gameId, selectedUnit, authToken);
                    break;
                  case 'refuel-port':
                    result = await unitsAPI.refuelAtPort(gameId, selectedUnit, authToken);
                    break;
                  case 'refuel-sea':
                    result = await unitsAPI.refuelAtSea(gameId, selectedUnit, authToken);
                    break;
                  case 'patrol':
                    result = await unitsAPI.setPatrol(gameId, selectedUnit, true, authToken);
                    break;
                  default:
                    return;
                }
              }
              
              if (result.success) {
                if (onRefreshData) {
                  await onRefreshData();
                }
                // Сбрасываем выбор юнита после успешного выполнения действия
                if (onUnitDeselect) {
                  onUnitDeselect();
                }
              } else {
                console.error('Action failed:', result.error);
              }
            } catch (error) {
              console.error('Error executing action:', error);
            }
          };

          return (
            <div className="unit-actions" style={{ display: 'flex', gap: '5px', marginLeft: '10px' }}>
              {unit.available_actions
                .filter((action: string) => action !== 'movement') // Движение обрабатывается отдельно
                .map((action: string) => (
                  <button
                    key={action}
                    onClick={() => handleAction(action)}
                    className="action-button"
                    title={getActionLabel(action)}
                  >
                    {getActionLabel(action)}
                  </button>
                ))}
            </div>
          );
        })()}
        
        {/* Кнопки Task Force и Воздушная разведка */}
        {currentPhase === 'movement' && !isCreateTFMode && !isFlightPathSearchMode && (
          <>
            <button 
              onClick={handleCreateTFClick}
              title="Создать Task Force"
            >
              🚢 Создать TF
            </button>
            <button 
              onClick={handleFlightPathSearchClick}
              title="Воздушная разведка"
            >
              ✈️ Воздушная разведка
            </button>
          </>
        )}
        {isCreateTFMode && (
          <button 
            onClick={handleCancelCreateTF}
            title="Отменить создание TF"
          >
            ❌ Отмена
          </button>
        )}
        {isFlightPathSearchMode && (
          <button 
            onClick={() => {
              console.log('❌ Cancel Flight Path Search button clicked');
              handleCancelFlightPathSearch();
            }} 
            title="Отменить воздушную разведку"
          >
            ❌ Отмена разведки
          </button>
        )}
        
        {/* Кнопки управления игрой */}
        <div className="game-controls">
          {isStartFirstTurnVisible && (
            <button 
              className="action-button primary"
              onClick={onStartFirstTurn}
            >
              🚀 Начать ход 1
            </button>
          )}
          <button 
            className="action-button"
            onClick={onCompletePhase}
            disabled={isCompletePhaseDisabled}
          >
            Завершить ход
          </button>
        </div>
      </div>

      <div className="hex-map-wrapper">
        <svg
          className="hex-map"
          width={svgWidth}
          height={svgHeight}
          style={{
            transform: `translate(${mapOffset.x}px, ${mapOffset.y}px)`
          }}
        >
          <defs>
            {/* Фоновое изображение карты */}
            <pattern 
              id="mapBackground" 
              patternUnits="userSpaceOnUse" 
              width={svgWidth} 
              height={svgHeight}
              x="0" 
              y="0"
            >
              <image 
                href={`/assets/maps/${playerSide}-map.jpg`}
                width={svgWidth} 
                height={svgHeight} 
                preserveAspectRatio="xMidYMid slice"
              />
            </pattern>
            
            {/* Градиенты для разных типов гексов */}
            <radialGradient id="waterGradient" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#4A90E2" />
              <stop offset="100%" stopColor="#2E5C8A" />
            </radialGradient>
            <radialGradient id="landGradient" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#8B4513" />
              <stop offset="100%" stopColor="#654321" />
            </radialGradient>
            <radialGradient id="portGradient" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stopColor="#CD853F" />
              <stop offset="100%" stopColor="#A0522D" />
            </radialGradient>
          </defs>
          
          {/* Фоновое изображение карты */}
          <rect 
            x="0" 
            y="0" 
            width={svgWidth} 
            height={svgHeight} 
            fill="url(#mapBackground)"
          />
          
          {hexElements}
        </svg>
      </div>
      
      {/* Tooltip */}
      {tooltip && (
        <div
          className="unit-tooltip"
          style={{
            position: 'absolute',
            left: tooltip.x + 20,
            top: tooltip.y - 30,
            zIndex: 1000,
            pointerEvents: 'none',
            transform: 'translate(-50%, -100%)'
          }}
        >
          {getUnitDescription(tooltip.unitType, tooltip.unitId, tooltip.unitSide).split('\n').map((line, index) => (
            <div key={index}>{line}</div>
          ))}
          {/* Стрелочка */}
          <div 
            style={{
              position: 'absolute',
              top: '100%',
              left: '50%',
              transform: 'translateX(-50%)',
              width: 0,
              height: 0,
              borderLeft: '6px solid transparent',
              borderRight: '6px solid transparent',
              borderTop: '6px solid rgba(0, 0, 0, 0.95)'
            }}
          />
        </div>
      )}
      
      {/* Подсказка для гекса */}
      {hexTooltip && (
        <Tooltip
          visible={hexTooltip.visible}
          x={hexTooltip.x}
          y={hexTooltip.y}
          content={hexTooltip.content}
        />
      )}
      
      {/* Диалог создания Task Force */}
      {showTFDialog && selectedTFHex && (
        <CreateTaskForceDialog
          hexId={selectedTFHex}
          units={gameUnits.filter(u => u.position === selectedTFHex)}
          taskForces={taskForces.filter(tf => tf.position === selectedTFHex)}
          allUnits={gameUnits}
          onConfirm={handleCreateTF}
          onAddToExisting={handleAddToExistingTF}
          onRemoveFromTF={handleRemoveFromExistingTF}
          onCancel={() => {
            setShowTFDialog(false);
            setSelectedTFHex(null);
          }}
        />
      )}
    </div>
  );
};

export default HexMap;
