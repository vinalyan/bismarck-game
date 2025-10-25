// Компонент отдельного гекса (упрощенная версия для редактора)

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
  onClick,
  onHover,
}) => {
  // Преобразуем углы в строку для SVG polygon
  const getHexPoints = () => {
    return corners.map(corner => `${corner.x},${corner.y}`).join(' ');
  };

  // Определяем стиль гекса
  const getHexStyle = () => {
    let fill = '#c8e6f5';
    let stroke = '#b0b0b0';
    let strokeWidth = 1;
    let fillOpacity = 0.3;
    
    // Цвет в зависимости от типа гекса
    switch (hexData.type) {
      case 'water':
        fill = '#c8e6f5';
        fillOpacity = 0.2;
        break;
      case 'land':
        fill = '#8fbc8f';
        fillOpacity = 0.5;
        break;
      case 'port':
        fill = '#87ceeb';
        fillOpacity = 0.4;
        strokeWidth = 2;
        break;
      default:
        fill = '#cccccc';
        fillOpacity = 0.3;
    }
    
    // Выделение выбранного гекса
    if (isSelected) {
      fill = '#2196f3';
      fillOpacity = 0.6;
      stroke = '#2196f3';
      strokeWidth = 3;
    }
    
    return { fill, stroke, strokeWidth, fillOpacity };
  };

  const hexStyle = getHexStyle();
  const points = getHexPoints();

  return (
    <g
      className={`hex ${hexData.type} ${isSelected ? 'selected' : ''}`}
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
      
      {/* Отображение координат */}
      <text
        x={center.x}
        y={center.y + size * 0.15}
        fontSize={size * 0.3}
        fill="#333"
        textAnchor="middle"
        pointerEvents="none"
        className="hex-coord"
      >
        {coordinate.letter}{coordinate.number}
      </text>
    </g>
  );
};

export { Hex };
