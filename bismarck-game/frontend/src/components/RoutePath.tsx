import React from 'react';
import { HexCoordinate } from '../types/mapTypes';
import { MAP_CONSTANTS, offsetToPixel } from '../utils/hexUtils';

interface RoutePathProps {
  path: HexCoordinate[];
  playerSide: 'german' | 'allied';
}

const RoutePath: React.FC<RoutePathProps> = ({ path, playerSide }) => {
  if (path.length < 2) {
    return null;
  }

  // Создаем SVG path для отображения маршрута
  const createPathData = () => {
    const points = path.map(coord => {
      const offset = { col: coord.col, row: coord.row };
      const pixel = offsetToPixel(offset, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      return `${pixel.x},${pixel.y}`;
    });

    return `M ${points.join(' L ')}`;
  };

  return (
    <g className="route-path">
      <path
        d={createPathData()}
        fill="none"
        stroke={playerSide === 'german' ? '#ff6b6b' : '#4ecdc4'}
        strokeWidth="4"
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity="0.8"
      />
      
      {/* Добавляем стрелки на пути */}
      {path.map((coord, index) => {
        if (index === 0 || index === path.length - 1) {
          return null; // Не рисуем стрелки на начальной и конечной точках
        }
        
        const offset = { col: coord.col, row: coord.row };
        const pixel = offsetToPixel(offset, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
        
        return (
          <circle
            key={`route-point-${index}`}
            cx={pixel.x}
            cy={pixel.y}
            r="3"
            fill={playerSide === 'german' ? '#ff6b6b' : '#4ecdc4'}
            opacity="0.9"
          />
        );
      })}
    </g>
  );
};

export default RoutePath;
