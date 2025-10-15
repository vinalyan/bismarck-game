// Компонент отдельного гекса

import React from 'react';
import { HexCoordinate, HexData } from '../types/mapTypes';
import { Point } from '../utils/hexUtils';
import './Hex.css';

interface HexProps {
  coordinate: HexCoordinate;
  hexData: HexData;
  center: Point;
  corners: Point[];
  size: number;
  isSelected: boolean;
  isHighlighted: boolean;
  isHighlightedGreen: boolean;
  onClick: () => void;
  onHover: () => void;
}

const Hex: React.FC<HexProps> = ({
  coordinate,
  hexData,
  center,
  corners,
  size,
  isSelected,
  isHighlighted,
  isHighlightedGreen,
  onClick,
  onHover
}) => {
  // Преобразуем углы в строку для SVG polygon
  const getHexPoints = () => {
    return corners.map(corner => `${corner.x},${corner.y}`).join(' ');
  };

  // Определяем стиль гекса в зависимости от типа и состояния
  const getHexStyle = () => {
    let fill = '';
    let stroke = 'transparent'; // Полностью прозрачные границы
    let strokeWidth = 1;
    let fillOpacity = 0; // Полностью прозрачные гексы
    
    // Цвет в зависимости от типа гекса (все прозрачные)
    switch (hexData.type) {
      case 'water':
        fill = 'url(#waterGradient)';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
        break;
      case 'land':
        fill = 'url(#landGradient)';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
        break;
      case 'port':
        fill = 'url(#portGradient)';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
        strokeWidth = 2;
        break;
      default:
        fill = '#cccccc';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
    }
    
    // Выделение выбранного гекса
    if (isSelected) {
      stroke = '#ff0000';
      strokeWidth = 3;
    }
    
    // Выделение подсвеченных гексов (приоритет: зеленое > желтое)
    if (isHighlightedGreen) {
      stroke = '#00ff00';
      strokeWidth = 3;
    } else if (isHighlighted) {
      stroke = '#ffff00';
      strokeWidth = 2;
    }
    
    return { fill, stroke, strokeWidth, fillOpacity };
  };

  const hexStyle = getHexStyle();
  const points = getHexPoints();

  return (
    <g
      className={`hex ${hexData.type} ${isSelected ? 'selected' : ''} ${isHighlighted ? 'highlighted' : ''} ${isHighlightedGreen ? 'highlighted-green' : ''}`}
      onClick={onClick}
      onMouseEnter={onHover}
      style={{ cursor: 'pointer' }}
    >
      {/* Основной гекс */}
      <polygon
        points={points}
        fill={hexStyle.fill}
        fillOpacity={hexStyle.fillOpacity}
        stroke={hexStyle.stroke}
        strokeWidth={hexStyle.strokeWidth}
        className="hex-shape"
      />
      
      
      {/* Юнит на гексе */}
      {hexData.hasUnit && hexData.unitId && (
        <circle
          cx={center.x}
          cy={center.y - size * 0.2}
          r={size * 0.2}
          fill={hexData.unitSide === 'german' ? '#ff0000' : '#0000ff'}
          stroke="#ffffff"
          strokeWidth={1}
          className="unit-marker"
        />
      )}
      
      {/* Маркер тумана войны */}
      {hexData.fogLevel > 0 && (
        <circle
          cx={center.x}
          cy={center.y + size * 0.2}
          r={size * 0.15}
          fill="#333333"
          opacity={hexData.fogLevel / 100}
          className="fog-marker"
        />
      )}
      
      {/* Погодные эффекты */}
      {hexData.weather === 'storm' && (
        <path
          d={`M ${center.x - size * 0.3} ${center.y - size * 0.3} L ${center.x + size * 0.3} ${center.y + size * 0.3} M ${center.x + size * 0.3} ${center.y - size * 0.3} L ${center.x - size * 0.3} ${center.y + size * 0.3}`}
          stroke="#ffffff"
          strokeWidth={1}
          opacity={0.7}
          className="weather-effect"
        />
      )}
    </g>
  );
};

export { Hex };
