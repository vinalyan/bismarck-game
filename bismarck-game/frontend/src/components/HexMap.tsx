// Гексагональная карта для игры Bismarck Chase
// Использует алгоритмы из Red Blob Games: https://www.redblobgames.com/grids/hexagons/implementation.html

import React, { useState, useEffect } from 'react';
import { Hex } from './Hex';
import { HexCoordinate, HexData, coordinateToHex, hexToCoordinate } from '../types/mapTypes';
import { 
  hex, hexToPixel, polygonCorners, createLayout, LAYOUT_FLAT, Point,
  hexRange, hexDistance, isValidHex, qoffsetToCube
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
  height = 34, // 34 гекса по вертикали (A-AH)
  onHexClick,
  onHexHover,
  selectedHex,
  highlightedHexes = []
}) => {
  const [hexes, setHexes] = useState<Map<string, HexData>>(new Map());
  const [mapOffset, setMapOffset] = useState({ x: 0, y: 0 });
  const [layout, setLayout] = useState<any>(null);

  // Создаем макет для отрисовки
  useEffect(() => {
    // Размеры согласно требованиям: радиус 0.5см, расстояние между центрами 1см
    const hexRadius = 18.9; // 0.5 см в пикселях (96 DPI)
    const hexSize = { x: hexRadius, y: hexRadius };
    const origin = { x: 50, y: 50 }; // Начальная позиция
    
    const newLayout = createLayout(LAYOUT_FLAT, hexSize, origin);
    setLayout(newLayout);
  }, []);

  // Генерируем координаты гексов
  useEffect(() => {
    const newHexes = new Map<string, HexData>();
    
    // Создаем гексы используя правильные offset координаты
    for (let row = 0; row < height; row++) {
      for (let col = 0; col < width; col++) {
        const letter = String.fromCharCode(65 + row); // A, B, C, ..., AH
        const number = col + 1; // 1, 2, 3, ..., 35
        
        // Для правильного отображения карты используем простую систему:
        // q = col (горизонтальная координата)
        // r = row (вертикальная координата)
        // s = -q - r (третья координата для гексагональной системы)
        const hexCoord = hex(col, row);
        
        const coordinate: HexCoordinate = {
          letter: letter,
          number: number,
          q: hexCoord.q,
          r: hexCoord.r
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

  // Вычисляем размеры SVG
  if (!layout) return <div>Loading map...</div>;
  
  // Вычисляем границы карты
  const hexRadius = layout.size.x;
  const hexWidth = hexRadius * Math.sqrt(3);
  const hexHeight = hexRadius * 2;
  
  const svgWidth = width * hexWidth * 0.75 + 100; // +100 для отступов
  const svgHeight = height * hexHeight * 0.5 + 100;

  // Рендерим гексы
  const renderHexes = () => {
    const hexElements: React.JSX.Element[] = [];
    
    hexes.forEach((hexData, hexId) => {
      const { coordinate } = hexData;
      
      // Преобразуем координаты в гексагональную систему
      const hexCoord = coordinateToHex(coordinate);
      
      // Получаем позицию центра гекса
      const center = hexToPixel(layout, hexCoord);
      
      // Получаем углы гекса для отрисовки
      const corners = polygonCorners(layout, hexCoord);
      
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
