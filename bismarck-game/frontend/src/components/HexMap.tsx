// Гексагональная карта для игры Bismarck Chase

import React, { useState, useEffect } from 'react';
import { Hex } from './Hex';
import { HexCoordinate, HexData } from '../types/mapTypes';
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

  // Генерируем координаты гексов
  useEffect(() => {
    const newHexes = new Map<string, HexData>();
    
    for (let row = 0; row < height; row++) {
      for (let col = 0; col < width; col++) {
        const letter = String.fromCharCode(65 + row); // A, B, C, ..., AH
        const number = col + 1; // 1, 2, 3, ..., 35
        const coordinate: HexCoordinate = {
          letter: letter,
          number: number,
          q: col,
          r: row
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
  const hexSize = 24; // Размер гекса в пикселях
  const hexWidth = Math.sqrt(3) * hexSize;
  const hexHeight = 2 * hexSize;
  
  const svgWidth = width * hexWidth * 0.75 + hexWidth * 0.25;
  const svgHeight = height * hexHeight * 0.5 + hexHeight * 0.5;

  // Рендерим гексы
  const renderHexes = () => {
    const hexElements: React.JSX.Element[] = [];
    
    hexes.forEach((hexData, hexId) => {
      const { coordinate } = hexData;
      
      // Вычисляем позицию гекса в point-top ориентации
      const x = coordinate.q * hexWidth * 0.75;
      const y = coordinate.r * hexHeight * 0.5 + (coordinate.q % 2) * hexHeight * 0.25;
      
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
          x={x}
          y={y}
          size={hexSize}
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
