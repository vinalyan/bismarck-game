// Гексагональная карта для игры Bismarck Chase
// Использует алгоритмы из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

import React, { useState, useEffect } from 'react';
import { Hex } from './Hex';
import { HexCoordinate, HexData, coordinateToOffset, offsetToCoordinate } from '../types/mapTypes';
import { 
  Point, OffsetCoord, offsetToPixel, offsetPolygonCorners
} from '../utils/hexUtils';
import './HexMap.css';

interface HexMapProps {
  width?: number;
  height?: number;
  onHexClick?: (hex: HexCoordinate) => void;
  onHexHover?: (hex: HexCoordinate) => void;
  selectedHex?: HexCoordinate | null;
  highlightedHexes?: HexCoordinate[];
}

const HexMap: React.FC<HexMapProps> = ({
  width = 35, // 35 гексов по горизонтали (1-35)
  height = 33, // 33 гекса по вертикали (A-AH)
  onHexClick,
  onHexHover,
  selectedHex,
  highlightedHexes = []
}) => {
  const [hexes, setHexes] = useState<Map<string, HexData>>(new Map());
  const [mapOffset, setMapOffset] = useState({ x: 0, y: 0 });
  const [hexRadius] = useState(18.9); // 0.5 см в пикселях (96 DPI)

  // Генерируем координаты гексов
  useEffect(() => {
    const newHexes = new Map<string, HexData>();
    
    // Создаем гексы используя offset координаты (col, row)
    for (let row = 0; row < height; row++) {
      for (let col = 0; col < width; col++) {
        // Генерируем правильные буквы: A-Y, затем AA-AH
        let letter: string;
        if (row < 25) {
          // A, B, C, ..., Y (0-24)
          letter = String.fromCharCode(65 + row);
        } else {
          // AA, AB, AC, ..., AH (25-33, но только до H)
          const secondLetterIndex = row - 25;
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
        newHexes.set(hexId, {
          coordinate,
          type: 'water', // По умолчанию все гексы - вода
          isVisible: true,
          isHighlighted: false,
          hasUnit: false,
          unitId: null,
          unitSide: null,
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

  // Вычисляем размеры SVG с учетом увеличенных расстояний
  const hexWidth = hexRadius * Math.sqrt(3);
  const hexHeight = hexRadius * 2;
  
  // Используем те же коэффициенты, что и в offsetToPixel
  const horizontalSpacing = hexWidth * 0.75 + 10; // Соответствует формуле в hexUtils
  const verticalSpacing = hexRadius * 1.5 + 2;   // Соответствует формуле в hexUtils
  
  // Учитываем максимальное смещение нечетных строк
  const maxHorizontalOffset = (hexWidth * 0.375) + 3; // Максимальное смещение
  
  const svgWidth = width * horizontalSpacing + maxHorizontalOffset + 100; // +100 для отступов
  const svgHeight = height * verticalSpacing + 100;

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
      
      const isHighlighted = highlightedHexes.some(h => 
        h.letter === coordinate.letter && h.number === coordinate.number
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
          isHighlighted={isHighlighted}
          onClick={() => handleHexClick(coordinate)}
          onHover={() => handleHexHover(coordinate)}
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
        <p>Координаты: A1 - {String.fromCharCode(64 + height)}{width}</p>
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
          
          {renderHexes()}
        </svg>
      </div>
    </div>
  );
};

export default HexMap;
