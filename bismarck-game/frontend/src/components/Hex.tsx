// Компонент отдельного гекса

import React, { useState, useRef, useEffect } from 'react';
import { HexCoordinate, HexData, MapStructure } from '../types/mapTypes';
import { Point } from '../utils/hexUtils';
import { useGameStore } from '../stores/gameStore';
import { ActiveHex, ACTIVE_HEX_CONFIGS } from '../utils/activeHexesUtils';
import { createHexTooltip } from '../utils/hexTooltipUtils';
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
  selectedUnit?: string | null;
  expandedStackHex?: string | null;
  currentTurn?: number;
  onClick: () => void;
  onHover: () => void;
  onUnitClick?: (unitId: string, unitData: any) => void;
  onUnitHover?: (unitId: string, unitType: string, unitSide: string, x: number, y: number) => void;
  onUnitLeave?: () => void;
  onTooltipShow?: (x: number, y: number, content: { hexId: string; hexType: string; features: string[] }) => void;
  onTooltipHide?: () => void;
  onUnitStackClick?: (hexId: string, units: any[]) => void;
  onStackedUnitSelect?: (unit: any) => void;
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
  selectedUnit = null,
  expandedStackHex = null,
  currentTurn = 0,
  onClick,
  onHover,
  onUnitClick,
  onUnitHover,
  onUnitLeave,
  onTooltipShow,
  onTooltipHide,
  onUnitStackClick,
  onStackedUnitSelect
}) => {
  // Состояние для подсказки
  const [showTooltip, setShowTooltip] = useState(false);
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hexRef = useRef<SVGPolygonElement>(null);

  // Обработчики для подсказок
  const handleHexMouseEnter = (event: React.MouseEvent) => {
    
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
    }

    hoverTimeoutRef.current = setTimeout(() => {
      if (!hexRef.current) {
        return;
      }
      
      const rect = hexRef.current.getBoundingClientRect();
      const hexId = `${coordinate.letter}${coordinate.number}`;
      const content = createHexTooltip(hexId, mapStructures);
      
      setShowTooltip(true);
      
      // Передаем данные подсказки в HexMap
      if (onTooltipShow) {
        onTooltipShow(rect.left + rect.width / 2, rect.top + rect.height / 2, content);
      }
    }, 2000); // 2 секунды задержка

    onHover();
  };

  const handleHexMouseLeave = () => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
    setShowTooltip(false);
    
    // Скрываем подсказку в HexMap
    if (onTooltipHide) {
      onTooltipHide();
    }
  };

  const handleHexMouseMove = (event: React.MouseEvent) => {
    // Подсказка теперь управляется на уровне HexMap
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

  // Функция для определения состояния юнита
  const getUnitState = (unit: any): 'idle' | 'selected' | 'active' | 'cannot-move' => {
    if (selectedUnit === unit.id) {
      return 'selected';
    }
    
    // Проверяем условия "не может двигаться"
    if (unit.last_move_turn === currentTurn || 
        unit.no_movement_turns_left > 0 || 
        unit.fuel <= 0) {
      return 'cannot-move';
    }
    
    // Здесь можно добавить логику для "active" состояния
    // Пока возвращаем 'idle'
    return 'idle';
  };

  // Функция для получения CSS класса состояния юнита
  const getUnitStateClass = (unit: any): string => {
    const state = getUnitState(unit);
    return `unit-container ${state}`;
  };

  // Функция для определения состояния стека
  const getStackState = (units: any[]): 'idle' | 'selected' | 'active' | 'cannot-move' => {
    // Если есть выбранный юнит в стеке
    if (units.some(unit => selectedUnit === unit.id)) {
      return 'selected';
    }
    
    // Если все юниты не могут двигаться
    if (units.every(unit => 
      unit.last_move_turn === currentTurn || 
      unit.no_movement_turns_left > 0 || 
      unit.fuel <= 0
    )) {
      return 'cannot-move';
    }
    
    // По умолчанию idle
    return 'idle';
  };

  // Функция рендеринга юнитов
  const renderUnits = () => {
    if (!hexData.hasUnit || !hexData.units || hexData.units.length === 0) {
      return null;
    }

    const units = hexData.units;
    const isStack = units.length > 1;
    const hexId = `${coordinate.letter}${coordinate.number}`;
    const isExpanded = expandedStackHex === hexId;

    if (isStack && !isExpanded) {
      // Отображаем свернутый стек юнитов
      const stackState = getStackState(units);
      return (
        <g 
          className={`unit-stack-container ${stackState}`}
          onClick={(e) => {
            e.stopPropagation();
            if (onUnitStackClick) {
              onUnitStackClick(hexId, units);
            }
          }}
          style={{ cursor: 'pointer' }}
        >
          {/* Фоновый кружок для стека */}
          <circle
            cx={center.x}
            cy={center.y}
            r={size * 0.6}
            className={`unit-stack-background ${stackState}`}
          />
          
          {/* Иконка первого юнита */}
          <image
            href={getUnitIcon(units[0].type, units[0].nationality === 'german' ? 'german' : 'allied')}
            x={center.x - size * 0.4}
            y={center.y - size * 0.4}
            width={size * 0.8}
            height={size * 0.8}
            className="unit-stack-icon"
            preserveAspectRatio="xMidYMid meet"
          />
          
          {/* Индикатор количества юнитов */}
          <circle
            cx={center.x + size * 0.3}
            cy={center.y - size * 0.3}
            r={size * 0.2}
            className="unit-count-badge"
          />
          <text
            x={center.x + size * 0.3}
            y={center.y - size * 0.25}
            className="unit-count-text"
          >
            {units.length}
          </text>
          
          {/* Анимированное кольцо для привлечения внимания */}
          <circle
            cx={center.x}
            cy={center.y}
            r={size * 0.7}
            className="unit-stack-ring"
          />
        </g>
      );
    }

    if (isStack && isExpanded) {
      // Отображаем развернутый стек (вертикальный список)
      return (
        <g className="expanded-unit-stack">
          {units.map((unit, index) => {
            const unitY = center.y + (index - (units.length - 1) / 2) * size * 0.8;
            
            return (
              <g
                key={unit.id}
                onClick={(e) => {
                  e.stopPropagation();
                  if (onStackedUnitSelect) {
                    onStackedUnitSelect(unit);
                  }
                }}
                style={{ cursor: 'pointer' }}
              >
                {/* Иконка юнита */}
                <image
                  href={getUnitIcon(unit.type, unit.nationality === 'german' ? 'german' : 'allied')}
                  x={center.x - 10}
                  y={unitY - 10}
                  width={20}
                  height={20}
                  className="stacked-unit-icon"
                  preserveAspectRatio="xMidYMid meet"
                />
              </g>
            );
          })}
        </g>
      );
    }

    // Отображаем одиночный юнит (существующая логика)
    const unit = units[0];
    const unitState = getUnitState(unit);
    
    return (
      <g 
        className={getUnitStateClass(unit)}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onClick={(e) => {
          e.stopPropagation();
          if (onUnitClick) {
            onUnitClick(hexData.unitId!, {
              id: hexData.unitId,
              type: hexData.unitType,
              side: hexData.unitSide,
              position: coordinate,
              name: getUnitName(hexData.unitType || '', hexData.unitSide || 'german'),
              maxFuel: 10,
              currentFuel: 8
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
        
        {/* Кольцо для выбранного юнита */}
        {unitState === 'selected' && (
          <circle
            cx={center.x}
            cy={center.y}
            r={size * 0.6}
            className="unit-selected-ring"
          />
        )}
        
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
    );
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
        onMouseEnter={(e) => {
          onHover();
          handleHexMouseEnter(e);
        }}
        onMouseLeave={handleHexMouseLeave}
        onMouseMove={handleHexMouseMove}
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
      />
      
      
      {/* Юниты на гексе */}
      {renderUnits()}
      
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
    </>
  );
};

export { Hex };
