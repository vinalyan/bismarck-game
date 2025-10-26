// Компонент отдельного гекса

import React, { useState, useRef, useEffect } from 'react';
import { HexCoordinate, HexData, MapStructure } from '../types/mapTypes';
import { Point } from '../utils/hexUtils';
import { useGameStore } from '../stores/gameStore';
import { ActiveHex, ACTIVE_HEX_CONFIGS } from '../utils/activeHexesUtils';
import { createHexTooltip } from '../utils/hexTooltipUtils';
import Tooltip from './Tooltip';
import './Hex.css';

interface HexProps {
  coordinate: HexCoordinate;
  hexData: HexData;
  center: Point;
  corners: Point[];
  size: number;
  isSelected: boolean;
  isAvailableForMovement?: boolean;
  activeHex?: ActiveHex | null;
  mapStructures?: MapStructure | null;
  onClick: () => void;
  onHover: () => void;
  onUnitClick?: (unitId: string, unitData: any) => void;
  onUnitHover?: (unitId: string, unitType: string, unitSide: string, x: number, y: number) => void;
  onUnitLeave?: () => void;
}

const Hex: React.FC<HexProps> = ({
  coordinate,
  hexData,
  center,
  corners,
  size,
  isSelected,
  isAvailableForMovement = false,
  activeHex = null,
  mapStructures = null,
  onClick,
  onHover,
  onUnitClick,
  onUnitHover,
  onUnitLeave
}) => {
  // Состояние для подсказки
  const [showTooltip, setShowTooltip] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 });
  const [tooltipContent, setTooltipContent] = useState({ hexId: '', hexType: '', features: [] as string[] });
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hexRef = useRef<SVGPolygonElement>(null);

  // Обработчики для подсказок
  const handleHexMouseEnter = (event: React.MouseEvent) => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }

    hoverTimeoutRef.current = setTimeout(() => {
      if (!hexRef.current) return;
      
      const rect = hexRef.current.getBoundingClientRect();
      const hexId = `${coordinate.letter}${coordinate.number}`;
      const content = createHexTooltip(hexId, mapStructures);
      
      setTooltipPosition({
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2
      });
      setTooltipContent(content);
      setShowTooltip(true);
    }, 2000); // 2 секунды задержка

    onHover();
  };

  const handleHexMouseLeave = () => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
    setShowTooltip(false);
  };

  const handleHexMouseMove = (event: React.MouseEvent) => {
    if (showTooltip && hexRef.current) {
      const rect = hexRef.current.getBoundingClientRect();
      setTooltipPosition({
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2
      });
    }
  };

  // Очистка таймера при размонтировании
  useEffect(() => {
    return () => {
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current);
      }
    };
  }, []);

  // Функция для определения пути к иконке юнита
  const getUnitIcon = (unitType: string, unitSide: string) => {
    if (['BB', 'BC', 'CV', 'CA', 'CL', 'DD', 'CG', 'TK'].includes(unitType)) {
      return `/assets/units/${unitSide}/naval/${unitType}.svg`;
    } else if (unitType === 'B') {
      return `/assets/units/${unitSide}/air/Bomber.svg`;
    } else if (unitType === 'R') {
      return `/assets/units/${unitSide}/air/Recon.svg`;
    }
    return `/assets/units/${unitSide}/${unitType}.svg`;
  };

  // Функция для получения имени юнита
  const getUnitName = (unitType: string, unitSide: string) => {
    // Fallback для авиации
    const typeNames: { [key: string]: string } = {
      'B': 'Бомбардировщик',
      'R': 'Разведчик'
    };

    const sideName = unitSide === 'german' ? 'Немецкий' : 'Британский';
    const typeName = typeNames[unitType] || unitType; // Упрощенная версия без shipUtils
    
    return `${sideName} ${typeName}`;
  };

  // Функция для получения описания юнита
  const getUnitDescription = (unitType: string, unitId: string, unitSide: string) => {
    const sideName = unitSide === 'german' ? 'Немецкий' : 'Британский';
    const sideFlag = unitSide === 'german' ? '🇩🇪' : '🇬🇧';
    
    const typeNames: { [key: string]: string } = {
      'BB': 'Линейный корабль',
      'BC': 'Линейный крейсер',
      'CV': 'Авианосец',
      'CA': 'Тяжелый крейсер',
      'CL': 'Легкий крейсер',
      'DD': 'Эсминец',
      'CG': 'Береговая охрана',
      'TK': 'Танкер',
      'B': 'Бомбардировщик',
      'R': 'Разведчик'
    };

    const typeName = typeNames[unitType] || unitType;
    const coordinates = `${coordinate.letter}${coordinate.number}`;
    
    return `${sideFlag} ${sideName} ${typeName}\n${unitId}\nПозиция: ${coordinates}`;
  };

  // Обработчики для tooltip
  const handleMouseEnter = (e: React.MouseEvent) => {
    e.stopPropagation(); // Останавливаем всплытие события
    if (hexData.hasUnit && hexData.unitId && onUnitHover) {
      onUnitHover(
        hexData.unitId,
        hexData.unitType || '',
        hexData.unitSide || 'german',
        e.clientX,
        e.clientY
      );
    }
  };

  const handleMouseLeave = (e: React.MouseEvent) => {
    e.stopPropagation(); // Останавливаем всплытие события
    if (onUnitLeave) {
      onUnitLeave();
    }
  };
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
    switch (hexData.hexType) {
      case 'water':
        fill = 'url(#waterGradient)';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
        break;
      case 'land':
        fill = '#8B7355'; // Коричневый для суши
        fillOpacity = 0.3;
        stroke = 'transparent';
        break;
      case 'non_game':
        fill = '#333333'; // Серый для неигровых
        fillOpacity = 0.1;
        stroke = 'transparent';
        break;
      default:
        fill = 'url(#waterGradient)';
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
    }
    
    // Выделение выбранного гекса
    if (isSelected) {
      stroke = '#ff0000'; // Красный для первого выбранного
      strokeWidth = 3;
    }
    
    // Подсветка доступных гексов для движения (приоритетный способ)
    if (isAvailableForMovement) {
      stroke = '#22C55E'; // Зеленый для доступных гексов
      strokeWidth = 2;
      fillOpacity = 0.2; // Легкая подсветка
    } else if (activeHex) {
      // Подсветка активных гексов (альтернативный способ, только если нет доступных для движения)
      const config = ACTIVE_HEX_CONFIGS[activeHex.type];
      if (config.enabled) {
        stroke = config.strokeColor;
        strokeWidth = config.strokeWidth;
        fillOpacity = config.opacity;
        fill = config.color;
      }
    }
    
    // Специальная подсветка для restricted DD гексов
    if (hexData.isRestrictedDD) {
      stroke = '#FF6B35'; // Оранжевый для restricted DD гексов
      strokeWidth = 1;
      if (!isAvailableForMovement && !activeHex) {
        fillOpacity = 0.1; // Легкая подсветка
      }
    }
    
    return { fill, stroke, strokeWidth, fillOpacity };
  };

  const hexStyle = getHexStyle();
  const points = getHexPoints();

  return (
    <>
      <g
        className={`hex ${hexData.type} ${isSelected ? 'selected' : ''} ${isAvailableForMovement ? 'available-for-movement' : ''}`}
        onClick={onClick}
        onMouseEnter={onHover}
        style={{ cursor: activeHex ? 'pointer' : 'default' }}
      >
      {/* Основной гекс */}
      <polygon
        ref={hexRef}
        points={points}
        fill={hexStyle.fill}
        fillOpacity={hexStyle.fillOpacity}
        stroke={hexStyle.stroke}
        strokeWidth={hexStyle.strokeWidth}
        className="hex-shape"
        onMouseEnter={handleHexMouseEnter}
        onMouseLeave={handleHexMouseLeave}
        onMouseMove={handleHexMouseMove}
      />
      
      
      {/* Юнит на гексе */}
      {hexData.hasUnit && hexData.unitId && (
        <g 
          className="unit-container"
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
          onClick={(e) => {
            e.stopPropagation(); // Останавливаем всплытие события
            if (onUnitClick) {
              onUnitClick(hexData.unitId!, {
                id: hexData.unitId,
                type: hexData.unitType,
                side: hexData.unitSide,
                position: coordinate,
                name: getUnitName(hexData.unitType || '', hexData.unitSide || 'german'),
                // Данные корабля будут загружены через API при необходимости
                maxFuel: 10, // Временное значение, будет заменено данными из API
                currentFuel: 8 // Временное значение, будет заменено данными из API
              });
            }
          }}
          style={{ cursor: 'pointer' }}
        >
          {/* Фоновый кружок для лучшей видимости */}
          <circle
            cx={center.x}
            cy={center.y}
            r={size * 0.5}
            fill="rgba(255, 255, 255, 0.9)"
            stroke={hexData.unitSide === 'german' ? '#1e3a8a' : '#991b1b'}
            strokeWidth={2}
            className="unit-background"
          />
          {/* Иконка юнита */}
          <image
            href={getUnitIcon(hexData.unitType || '', hexData.unitSide || 'german')}
            x={center.x - size * 0.5}
            y={center.y - size * 0.5}
            width={size * 1.0}
            height={size * 1.0}
            className="unit-icon"
            preserveAspectRatio="xMidYMid meet"
          />
        </g>
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
      
      {/* Подсказка */}
      <Tooltip
        visible={showTooltip}
        x={tooltipPosition.x}
        y={tooltipPosition.y}
        content={tooltipContent}
      />
    </>
  );
};

export { Hex };
