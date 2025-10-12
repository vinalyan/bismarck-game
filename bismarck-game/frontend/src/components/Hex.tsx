// Компонент отдельного гекса

import React from 'react';
import { HexCoordinate, HexData } from '../types/mapTypes';
import './Hex.css';

interface HexProps {
  coordinate: HexCoordinate;
  hexData: HexData;
  x: number;
  y: number;
  size: number;
  isSelected: boolean;
  isHighlighted: boolean;
  onClick: () => void;
  onHover: () => void;
}

const Hex: React.FC<HexProps> = ({
  coordinate,
  hexData,
  x,
  y,
  size,
  isSelected,
  isHighlighted,
  onClick,
  onHover
}) => {
  // Вычисляем координаты вершин гекса для point-top ориентации
  const getHexPoints = () => {
    const width = Math.sqrt(3) * size;
    const height = 2 * size;
    
    const points = [
      { x: width / 2, y: 0 },                    // Верхняя точка
      { x: width, y: height / 4 },               // Верхний правый
      { x: width, y: (3 * height) / 4 },         // Нижний правый
      { x: width / 2, y: height },               // Нижняя точка
      { x: 0, y: (3 * height) / 4 },             // Нижний левый
      { x: 0, y: height / 4 }                    // Верхний левый
    ];
    
    return points.map(p => `${p.x},${p.y}`).join(' ');
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
      transform={`translate(${x}, ${y})`}
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
        x={size * Math.sqrt(3) / 2}
        y={size + 4}
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
          cx={size * Math.sqrt(3) / 2}
          cy={size * 0.8}
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
          cx={size * Math.sqrt(3) / 2}
          cy={size * 1.2}
          r={size * 0.15}
          fill="#333333"
          opacity={hexData.fogLevel / 100}
          className="fog-marker"
        />
      )}
      
      {/* Погодные эффекты */}
      {hexData.weather === 'storm' && (
        <path
          d={`M ${size * 0.3} ${size * 0.3} L ${size * 0.7} ${size * 0.7} M ${size * 0.7} ${size * 0.3} L ${size * 0.3} ${size * 0.7}`}
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
