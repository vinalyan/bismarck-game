// Гексагональная карта для игры Bismarck Chase
// Использует алгоритмы из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

import React, { useState, useEffect } from 'react';
import { Hex } from './Hex';
import { HexCoordinate, HexData, coordinateToOffset, offsetToCoordinate } from '../types/mapTypes';
import { 
  Point, OffsetCoord, offsetToPixel, offsetPolygonCorners, calculateMapSize, MAP_CONSTANTS, getCubeNeighbors
} from '../utils/hexUtils';
import './HexMap.css';

interface HexMapProps {
  width?: number;
  height?: number;
  onHexClick?: (hex: HexCoordinate) => void;
  onHexHover?: (hex: HexCoordinate) => void;
  selectedHex?: HexCoordinate | null;
  secondSelectedHex?: HexCoordinate | null;
  neighborHexes?: HexCoordinate[];
  routePath?: HexCoordinate[];
  routeDistance?: number;
  playerSide?: 'german' | 'allied';
}

const HexMap: React.FC<HexMapProps> = ({
  width = MAP_CONSTANTS.HEX_GRID_WIDTH, // 35.5 гексов по горизонтали
  height = MAP_CONSTANTS.HEX_GRID_HEIGHT, // 33 гекса по вертикали (A-AH)
  onHexClick,
  onHexHover,
  selectedHex,
  secondSelectedHex,
  neighborHexes = [],
  routePath = [],
  routeDistance = 0,
  playerSide = 'german'
}) => {
  const [hexes, setHexes] = useState<Map<string, HexData>>(new Map());
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
  useEffect(() => {
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
        
        // Добавляем тестовые юниты разных типов
        let hasUnit = false;
        let unitId = null;
        let unitType = null;
        let unitSide: 'german' | 'allied' | null = null;
        
        // BB - Линейный корабль в K15
        if (letter === 'K' && number === 15) {
          hasUnit = true;
          unitId = 'Bismark';
          unitType = 'BB';
          unitSide = 'german';
        }
        // BC - Линейный крейсер в L15
        else if (letter === 'L' && number === 15) {
          hasUnit = true;
          unitId = 'Scharnhorst';
          unitType = 'BC';
          unitSide = 'german';
        }
        // CV - Авианосец в M15
        else if (letter === 'M' && number === 15) {
          hasUnit = true;
          unitId = 'Graf Zeppelin';
          unitType = 'CV';
          unitSide = 'german';
        }
        // CA - Тяжелый крейсер в N15
        else if (letter === 'N' && number === 15) {
          hasUnit = true;
          unitId = 'Prinz Eugen';
          unitType = 'CA';
          unitSide = 'german';
        }
        // CL - Легкий крейсер в O15
        else if (letter === 'O' && number === 15) {
          hasUnit = true;
          unitId = 'Nurnberg';
          unitType = 'CL';
          unitSide = 'german';
        }
        // DD - Эсминец в P15
        else if (letter === 'P' && number === 15) {
          hasUnit = true;
          unitId = 'Z-23';
          unitType = 'DD';
          unitSide = 'german';
        }
        // CG - Береговая охрана в Q15
        else if (letter === 'Q' && number === 15) {
          hasUnit = true;
          unitId = 'Coast Guard';
          unitType = 'CG';
          unitSide = 'german';
        }
        // TK - Танкер в R15
        else if (letter === 'R' && number === 15) {
          hasUnit = true;
          unitId = 'Tanker';
          unitType = 'TK';
          unitSide = 'german';
        }
        // B - Бомбардировщик в S15
        else if (letter === 'S' && number === 15) {
          hasUnit = true;
          unitId = 'Ju-88';
          unitType = 'B';
          unitSide = 'german';
        }
        // R - Разведчик в T15
        else if (letter === 'T' && number === 15) {
          hasUnit = true;
          unitId = 'Fw-200';
          unitType = 'R';
          unitSide = 'german';
        }
        // Британские юниты - в ряду 20
        // BB - Линейный корабль в K20
        else if (letter === 'K' && number === 20) {
          hasUnit = true;
          unitId = 'Hood';
          unitType = 'BB';
          unitSide = 'allied';
        }
        // BC - Линейный крейсер в L20
        else if (letter === 'L' && number === 20) {
          hasUnit = true;
          unitId = 'Prince of Wales';
          unitType = 'BC';
          unitSide = 'allied';
        }
        // CV - Авианосец в M20
        else if (letter === 'M' && number === 20) {
          hasUnit = true;
          unitId = 'Ark Royal';
          unitType = 'CV';
          unitSide = 'allied';
        }
        // CA - Тяжелый крейсер в N20
        else if (letter === 'N' && number === 20) {
          hasUnit = true;
          unitId = 'Norfolk';
          unitType = 'CA';
          unitSide = 'allied';
        }
        // CL - Легкий крейсер в O20
        else if (letter === 'O' && number === 20) {
          hasUnit = true;
          unitId = 'Sheffield';
          unitType = 'CL';
          unitSide = 'allied';
        }
        // DD - Эсминец в P20
        else if (letter === 'P' && number === 20) {
          hasUnit = true;
          unitId = 'Cossack';
          unitType = 'DD';
          unitSide = 'allied';
        }
        // CG - Береговая охрана в Q20
        else if (letter === 'Q' && number === 20) {
          hasUnit = true;
          unitId = 'Coast Guard';
          unitType = 'CG';
          unitSide = 'allied';
        }
        // TK - Танкер в R20
        else if (letter === 'R' && number === 20) {
          hasUnit = true;
          unitId = 'Tanker';
          unitType = 'TK';
          unitSide = 'allied';
        }
        // B - Бомбардировщик в S20
        else if (letter === 'S' && number === 20) {
          hasUnit = true;
          unitId = 'Swordfish';
          unitType = 'B';
          unitSide = 'allied';
        }
        // R - Разведчик в T20
        else if (letter === 'T' && number === 20) {
          hasUnit = true;
          unitId = 'Sunderland';
          unitType = 'R';
          unitSide = 'allied';
        }
        
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
    
    setHexes(newHexes);
  }, [width, height]);

  // Обработчики событий
  const handleHexClick = (coordinate: HexCoordinate) => {
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
      
      const isSecondSelected = secondSelectedHex && 
        secondSelectedHex.letter === coordinate.letter && 
        secondSelectedHex.number === coordinate.number;
      
      const isNeighbor = neighborHexes.some(neighbor => 
        neighbor.letter === coordinate.letter && neighbor.number === coordinate.number
      );
      
      const isInRoute = routePath.some(routeHex => 
        routeHex.letter === coordinate.letter && routeHex.number === coordinate.number
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
          isSecondSelected={!!isSecondSelected}
          isHighlighted={isNeighbor}
          isHighlightedGreen={isInRoute}
          onClick={() => handleHexClick(coordinate)}
          onHover={() => handleHexHover(coordinate)}
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
