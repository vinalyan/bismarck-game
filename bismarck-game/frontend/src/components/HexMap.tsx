// Гексагональная карта для игры Bismarck Chase
// Использует алгоритмы из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

import React, { useState, useEffect, useMemo } from 'react';
import { Hex } from './Hex';
import { HexCoordinate, HexData, coordinateToOffset, offsetToCoordinate } from '../types/mapTypes';
import { MovementHex } from '../utils/movementUtils';
import { ActiveHex } from '../utils/activeHexesUtils';
import { 
  Point, OffsetCoord, offsetToPixel, offsetPolygonCorners, calculateMapSize, MAP_CONSTANTS, getCubeNeighbors
} from '../utils/hexUtils';
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
  unitPositions?: Map<string, HexCoordinate>;
  gameUnits?: any[]; // Добавляем данные юнитов из API
}

const HexMap: React.FC<HexMapProps> = ({
  width = MAP_CONSTANTS.HEX_GRID_WIDTH, // 35.5 гексов по горизонтали
  height = MAP_CONSTANTS.HEX_GRID_HEIGHT, // 33 гекса по вертикали (A-AH)
  onHexClick,
  onHexHover,
  onUnitClick,
  selectedHex,
  playerSide = 'german',
  availableMovementHexes = [],
  activeHexes = [],
  unitPositions = new Map(),
  gameUnits = []
}) => {
  const [mapOffset, setMapOffset] = useState({ x: 0, y: 0 });
  const [hexRadius] = useState(MAP_CONSTANTS.DEFAULT_HEX_RADIUS); // Стандартный радиус гекса
  const [tooltip, setTooltip] = useState<{
    show: boolean;
    unitId: string;
    unitType: string;
    unitSide: string;
    x: number;
    y: number;
  } | null>(null);

  // Генерируем координаты гексов
  const hexes = useMemo(() => {
    const newHexes = new Map<string, HexData>();
    
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
        
        // Проверяем, есть ли юнит в этой позиции
        let hasUnit = false;
        let unitId = null;
        let unitType = null;
        let unitSide: 'german' | 'allied' | null = null;

        // Ищем юнит в этой позиции
        unitPositions.forEach((pos, id) => {
          if (pos.col === col && pos.row === row) {
            hasUnit = true;
            unitId = id;
            
            // Находим данные юнита в gameUnits
            const unitData = gameUnits.find(unit => unit.id === id);
            if (unitData) {
              unitType = unitData.type;
              unitSide = unitData.nationality === 'german' ? 'german' : 'allied';
            } else {
              // Fallback к старой логике, если данные не найдены
              if (id === 'Bismark' || id === 'Scharnhorst' || id === 'Graf Zeppelin' || 
                  id === 'Prinz Eugen' || id === 'Nurnberg' || id === 'Z-23' || 
                  id === 'Ju-88' || id === 'Fw-200') {
                unitSide = 'german';
              } else {
                unitSide = 'allied';
              }

              // Определяем тип юнита
              if (id === 'Bismark' || id === 'Hood') {
                unitType = 'BB';
              } else if (id === 'Scharnhorst' || id === 'Prince of Wales') {
                unitType = 'BC';
              } else if (id === 'Graf Zeppelin' || id === 'Ark Royal') {
                unitType = 'CV';
              } else if (id === 'Prinz Eugen' || id === 'Norfolk') {
                unitType = 'CA';
              } else if (id === 'Nurnberg' || id === 'Sheffield') {
                unitType = 'CL';
              } else if (id === 'Z-23' || id === 'Cossack') {
                unitType = 'DD';
              } else if (id === 'Ju-88' || id === 'Swordfish') {
                unitType = 'B';
              } else if (id === 'Fw-200' || id === 'Sunderland') {
                unitType = 'R';
              } else {
                unitType = 'CG'; // По умолчанию
              }
            }
          }
        });
        
        newHexes.set(hexId, {
          coordinate,
          type: 'water', // По умолчанию все гексы - вода
          isVisible: true,
          isHighlighted: false,
          hasUnit,
          unitId,
          unitType,
          unitSide,
          weather: 'clear',
          fogLevel: 0
        });
      }
    }
    
    return newHexes;
  }, [width, height, unitPositions, gameUnits]);

  // Обработчики событий
  const handleHexClick = (coordinate: HexCoordinate) => {
    // Проверяем, является ли гекс активным
    const isActiveHex = activeHexes.some(hex => 
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    );
    
    // Кликабелен только если гекс активный
    if (isActiveHex && onHexClick) {
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
  const renderHexes = () => {
    const hexElements: React.JSX.Element[] = [];
    
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

      // Проверяем, является ли этот гекс активным
      const activeHex = activeHexes.find(
        hex => 
          hex.coordinate.col === coordinate.col && 
          hex.coordinate.row === coordinate.row
      );

      hexElements.push(
        <Hex
          key={hexId}
          coordinate={coordinate}
          hexData={hexData}
          center={center}
          corners={corners}
          size={hexRadius}
          isSelected={!!isSelected}
          isAvailableForMovement={isAvailableForMovement}
          activeHex={activeHex}
          onClick={() => handleHexClick(coordinate)}
          onHover={() => handleHexHover(coordinate)}
          onUnitClick={onUnitClick}
          onUnitHover={handleUnitHover}
          onUnitLeave={handleUnitLeave}
        />
      );
    });
    
    return hexElements;
  };

  return (
    <div className="hex-map-container">
      <div className="map-info">
        <h3>Карта Атлантики</h3>
        <p>Размер: {width}×{height} гексов</p>
      </div>
      
      <div className="map-controls">
        <button onClick={() => setMapOffset({ x: mapOffset.x - 50, y: mapOffset.y })}>
          ←
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
        <button onClick={() => setMapOffset({ x: 0, y: 0 })}>
          Центр
        </button>
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
          
          {renderHexes()}
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
    </div>
  );
};

export default HexMap;
