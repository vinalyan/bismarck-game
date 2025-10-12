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
    let stroke = '#000';
    let strokeWidth = 1;
    
    // Цвет в зависимости от типа гекса
    switch (hexData.type) {
      case 'water':
        fill = 'url(#waterGradient)';
        break;
      case 'land':
        fill = 'url(#landGradient)';
        break;
      case 'port':
        fill = 'url(#portGradient)';
        break;
      default:
        fill = '#cccccc';
    }
    
    // Выделение выбранного гекса
    if (isSelected) {
      stroke = '#ff0000';
      strokeWidth = 3;
    }
    
    // Выделение подсвеченных гексов
    if (isHighlighted) {
      stroke = '#ffff00';
      strokeWidth = 2;
    }
    
    return { fill, stroke, strokeWidth };
  };

  const hexStyle = getHexStyle();
  const points = getHexPoints();

  return (
    <g
      className={`hex ${hexData.type} ${isSelected ? 'selected' : ''} ${isHighlighted ? 'highlighted' : ''}`}
      onClick={onClick}
      onMouseEnter={onHover}
      style={{ cursor: 'pointer' }}
    >
      {/* Основной гекс */}
      <polygon
        points={points}
        fill={hexStyle.fill}
        stroke={hexStyle.stroke}
        strokeWidth={hexStyle.strokeWidth}
        className="hex-shape"
      />
      
      {/* Координаты гекса */}
      <text
        x={center.x}
        y={center.y + 4}
        textAnchor="middle"
        fontSize="10"
        fill="#ffffff"
        className="hex-coordinate"
      >
        {coordinate.letter}{coordinate.number}
      </text>
      
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
