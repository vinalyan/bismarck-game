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
  isSearchAvailable?: boolean;
  activeHex?: ActiveHex | null;
  mapStructures?: MapStructure | null;
  selectedUnit?: string | null;
  expandedStackHex?: string | null;
  currentTurn?: number;
  isTFCandidate?: boolean;
  hasFlightPathMarker?: boolean;
  isFog?: boolean;
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
  isSearchAvailable = false,
  activeHex = null,
  mapStructures = null,
  selectedUnit = null,
  expandedStackHex = null,
  currentTurn = 0,
  isTFCandidate = false,
  hasFlightPathMarker = false,
  isFog = false,
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
  const hexId = `${coordinate.letter}${coordinate.number}`;
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

  type UnitVisualState = 'idle' | 'selected' | 'active' | 'cannot-move' | 'emergency-fuel' | 'sighted' | 'shadowed';
  type TaskForceVisualState = 'idle' | 'selected' | 'active' | 'cannot-move' | 'sighted' | 'shadowed';

  // Функция для определения состояния юнита
  const getUnitState = (unit: any): UnitVisualState => {
    if (selectedUnit === unit.id) {
      return 'selected';
    }
    
    // Проверяем аварийное топливо
    if (unit.is_emergency_fuel === true) {
      return 'emergency-fuel';
    }

    const detectionLevel = typeof unit.detection_level === 'string' ? unit.detection_level.toLowerCase() : '';
    if (detectionLevel === 'shadowed') {
      return 'shadowed';
    }
    if (detectionLevel === 'sighted') {
      return 'sighted';
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
  const getStackState = (units: any[]): UnitVisualState => {
    // Если есть выбранный юнит в стеке
    if (units.some(unit => selectedUnit === unit.id)) {
      return 'selected';
    }
    
    // Если есть юнит с аварийным топливом
    if (units.some(unit => unit.is_emergency_fuel === true)) {
      return 'emergency-fuel';
    }

    if (units.some(unit => (unit.detection_level || '').toLowerCase() === 'shadowed')) {
      return 'shadowed';
    }
    if (units.some(unit => (unit.detection_level || '').toLowerCase() === 'sighted')) {
      return 'sighted';
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

// Функция для определения состояния Task Force
const getTaskForceState = (taskForce: any): TaskForceVisualState => {
  // Если Task Force выбран
  if (selectedUnit === taskForce.id) {
    return 'selected';
  }

  const detectionLevel = typeof taskForce.detection_level === 'string' ? taskForce.detection_level.toLowerCase() : '';
  if (detectionLevel === 'shadowed') {
    return 'shadowed';
  }
  if (detectionLevel === 'sighted') {
    return 'sighted';
  }
  
  // Проверяем, может ли двигаться (если есть данные о последнем ходе)
  if (taskForce.last_move_turn === currentTurn) {
    return 'cannot-move';
  }
  
  // По умолчанию idle
  return 'idle';
};

  // Функция рендеринга Task Force маркера
  const renderTaskForce = (taskForce: any) => {
    const tfName = taskForce.name || 'TF';
    const nationality = taskForce.nationality === 'german' ? 'german' : 'allied';
    const unitCount = taskForce.units?.length || 0;
    const svgPath = `/assets/units/${nationality}/TF/TF.svg`;
    const tfState = getTaskForceState(taskForce);
    
    return (
      <g 
        className={`task-force-container ${nationality} ${tfState}`}
        onClick={(e) => {
          e.stopPropagation();
          if (onUnitClick) {
            onUnitClick(taskForce.id, taskForce);
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
          stroke={nationality === 'german' ? '#1e3a8a' : '#991b1b'}
          strokeWidth={2}
          className="task-force-background"
        />
        
        {/* Кольцо для выбранного Task Force */}
        {tfState === 'selected' && (
          <circle
            cx={center.x}
            cy={center.y}
            r={size * 0.6}
            className="task-force-selected-ring"
            fill="none"
            stroke="#FFD700"
            strokeWidth={3}
          />
        )}
        
        {/* Task Force SVG иконка */}
        <image
          x={center.x - size * 0.5}
          y={center.y - size * 0.5}
          width={size * 1.0}
          height={size * 1.0}
          href={svgPath}
          className={`task-force-icon ${nationality}`}
        />
        
        {/* Имя Task Force */}
        <text
          x={center.x}
          y={center.y + size*0.85
          }
          className="task-force-name"
          textAnchor="middle"
          fontSize="10"
          fill={nationality === 'german' ? '#1D3A43' : '#CA6649'}
          fontWeight="bold"
        >
          {tfName}
        </text>
      </g>
    );
  };


  // Функция рендеринга смешанных элементов (Task Forces + Units)
  const renderSingleUnit = (item: any) => {
    if (item.isTaskForce) {
      return renderTaskForce(item);
    } else {
      const unitState = getUnitState(item);
      return (
        <g 
          className={`unit-container ${unitState}`}
          onClick={(e) => {
            e.stopPropagation();
            if (onUnitClick) {
              onUnitClick(item.id, item);
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
            stroke={item.nationality === 'german' ? '#1e3a8a' : '#991b1b'}
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
            href={getUnitIcon(item.type, item.nationality === 'german' ? 'german' : 'allied')}
            x={center.x - size * 0.5}
            y={center.y - size * 0.5}
            width={size * 1.0}
            height={size * 1.0}
            className="unit-icon"
            preserveAspectRatio="xMidYMid meet"
          />
        </g>
      );
    }
  };

  // Функция для определения состояния смешанного стека (Task Forces + Units)
  const getStackStateForMixedItems = (items: Array<any & { isTaskForce: boolean }>): UnitVisualState => {
    // Если есть выбранный объект в стеке
    if (items.some(item => selectedUnit === item.id)) {
      return 'selected';
    }

    // Если есть юнит с аварийным топливом
    if (items.some(item => !item.isTaskForce && item.is_emergency_fuel === true)) {
      return 'emergency-fuel';
    }

    if (items.some(item => {
      const level = (item.detection_level || '').toLowerCase();
      return level === 'shadowed';
    })) {
      return 'shadowed';
    }

    if (items.some(item => {
      const level = (item.detection_level || '').toLowerCase();
      return level === 'sighted';
    })) {
      return 'sighted';
    }

    // Если все объекты не могут двигаться
    if (items.every(item => {
      if (item.isTaskForce) {
        return item.last_move_turn === currentTurn;
      } else {
        return item.last_move_turn === currentTurn ||
               item.no_movement_turns_left > 0 ||
               item.fuel <= 0;
      }
    })) {
      return 'cannot-move';
    }

    // По умолчанию idle
    return 'idle';
  };

  // Функция рендеринга юнитов
  const renderUnits = () => {
    // Собираем все объекты в гексе для стекирования
    const allItems: Array<any & { isTaskForce: boolean }> = [];

    // Добавляем Task Forces
    if (hexData.taskForces && hexData.taskForces.length > 0) {
      hexData.taskForces.forEach(tf => {
        allItems.push({ ...tf, type: 'taskforce', isTaskForce: true });
      });
    }

    // Добавляем обычные юниты
    if (hexData.hasUnit && hexData.units && hexData.units.length > 0) {
      hexData.units.forEach(unit => {
        allItems.push({ ...unit, isTaskForce: false });
      });
    }

    // Если нет объектов для отображения
    if (allItems.length === 0) {
      return null;
    }

    // Если только один объект (Task Force или Unit)
    if (allItems.length === 1) {
      const item = allItems[0];
      return renderSingleUnit(item);
    }

    // Если несколько объектов - показываем стек
    const hexId = `${coordinate.letter}${coordinate.number}`;
    const isExpanded = expandedStackHex === hexId;
    const stackState = getStackStateForMixedItems(allItems);

    if (!isExpanded) {
      // Отображаем свернутый стек (Task Forces + Units)
      return (
        <g
          className={`unit-stack-container ${stackState}`}
          onClick={(e) => {
            e.stopPropagation();
            if (onUnitStackClick) {
              onUnitStackClick(hexId, allItems);
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

          {/* Иконка первого объекта (Task Force или Unit) */}
          {allItems[0].isTaskForce ? (
            <image
              href={`/assets/units/${allItems[0].nationality === 'german' ? 'german' : 'allied'}/TF/TF.svg`}
              x={center.x - size * 0.4}
              y={center.y - size * 0.4}
              width={size * 0.8}
              height={size * 0.8}
              className={`unit-stack-icon taskforce ${stackState}`}
              preserveAspectRatio="xMidYMid meet"
            />
          ) : (
            <image
              href={getUnitIcon(allItems[0].type, allItems[0].nationality === 'german' ? 'german' : 'allied')}
              x={center.x - size * 0.4}
              y={center.y - size * 0.4}
              width={size * 0.8}
              height={size * 0.8}
              className={`unit-stack-icon ${stackState}`}
              preserveAspectRatio="xMidYMid meet"
            />
          )}

          {/* Индикатор количества объектов */}
          <circle
            cx={center.x + size * 0.4}
            cy={center.y - size * 0.4}
            r={8}
            className="unit-count-badge"
          />
          <text
            x={center.x + size * 0.4}
            y={center.y - size * 0.4 + 1}
            className="unit-count-text"
          >
            {allItems.length}
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

    if (isExpanded) {
      // Отображаем развернутый стек (вертикальный список смешанных объектов)
      return (
        <g className="expanded-unit-stack">
          {allItems.map((item, index) => {
            const itemY = center.y + (index - (allItems.length - 1) / 2) * size * 1.2;

            const itemState = item.isTaskForce ? getTaskForceState(item) : getUnitState(item);

            return (
              <g
                key={item.id}
                className={`stacked-unit ${itemState} ${item.isTaskForce ? 'task-force' : 'naval-unit'}`}
                onClick={(e) => {
                  e.stopPropagation();
                  if (onStackedUnitSelect) {
                    onStackedUnitSelect(item);
                  }
                }}
                style={{ cursor: 'pointer' }}
              >
                {/* Фоновый кружок для иконки */}
                <circle
                  cx={center.x}
                  cy={itemY}
                  r={12}
                  className={`stacked-unit-background ${itemState}`}
                />

                {/* Кольцо для выбранного объекта */}
                {itemState === 'selected' && (
                  <circle
                    cx={center.x}
                    cy={itemY}
                    r={15}
                    className="stacked-unit-selected-ring"
                  />
                )}

                {/* Иконка объекта (Task Force или Unit) */}
                {item.isTaskForce ? (
                  <image
                    href={`/assets/units/${item.nationality === 'german' ? 'german' : 'allied'}/TF/TF.svg`}
                    x={center.x - 10}
                    y={itemY - 10}
                    width={20}
                    height={20}
                    className={`stacked-unit-icon taskforce ${itemState}`}
                    preserveAspectRatio="xMidYMid meet"
                  />
                ) : (
                  <image
                    href={getUnitIcon(item.type, item.nationality === 'german' ? 'german' : 'allied')}
                    x={center.x - 10}
                    y={itemY - 10}
                    width={20}
                    height={20}
                    className={`stacked-unit-icon ${itemState}`}
                    preserveAspectRatio="xMidYMid meet"
                  />
                )}
              </g>
            );
          })}
        </g>
      );
    }

    // Fallback для одиночных юнитов (не должно происходить с новой логикой)
    return null;
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
        fill = 'url(#waterGradient)'; // Убрана специальная подсветка для суши
        fillOpacity = 0; // Полная прозрачность
        stroke = 'transparent';
        break;
      case 'non_game':
        fill = 'url(#waterGradient)'; // Убрана специальная подсветка для неигровых
        fillOpacity = 0; // Полная прозрачность
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
    } else if (isSearchAvailable) {
      // Подсветка гексов с достаточными факторами поиска (желтый)
      stroke = '#FBBF24'; // Желтый для гексов с достаточными факторами поиска
      strokeWidth = 2;
      fillOpacity = 0.2; // Легкая подсветка
      fill = '#FBBF24'; // Желтый фон
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
    
    // НЕ применяем туманные стили к non_game и land гексам - возвращаем стили сразу
    if (hexData.hexType === 'non_game' || hexData.hexType === 'land') {
      return { fill, stroke, strokeWidth, fillOpacity };
    }
    
    // Подсветка туманных гексов (применяется только если isFog = true и гекс туманный)
    // Приоритет: после isAvailableForMovement, но может перекрываться выбранным гексом
    // Применяем только к water гексам (non_game и land уже обработаны выше)
    if (isFog && hexData.isFogHex && !isSelected && hexData.hexType === 'water') {
      // Полная подсветка тумана (не перекрываем стили доступных для движения или поиска)
      if (!isAvailableForMovement && !isSearchAvailable && !activeHex) {
        stroke = '#a1bcab';
        strokeWidth = 3;
        fillOpacity = 0.2;
        fill = '#3a4840';
      }
    }
    
    return { fill, stroke, strokeWidth, fillOpacity };
  };

  const hexStyle = getHexStyle();
  const points = getHexPoints();

  // Вычисляем координаты для маркера пути полета заранее
  const topY = Math.min(...corners.map(c => c.y));
  const topCorners = corners.filter(c => Math.abs(c.y - topY) < 1);
  const topRightCorner = topCorners.reduce((max, c) => c.x > max.x ? c : max, topCorners[0]);
  const iconSize = 18; // Фиксированный размер, как у иконок юнитов (20), но чуть меньше
  const iconX = topRightCorner.x - iconSize * 0.6;
  const iconY = topRightCorner.y - iconSize * 0.8;

  // Проверяем, нужно ли рендерить маркер
  const shouldRenderMarker = Boolean(hasFlightPathMarker);

  return (
    <>
      <g
        className={`hex ${hexData.type} ${isSelected ? 'selected' : ''} ${isAvailableForMovement ? 'available-for-movement' : ''} ${isTFCandidate ? 'tf-candidate' : ''} ${isSearchAvailable ? 'search-available' : ''}`}
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
      
      {/* Маркер пути полета (Flight Path) - ВНУТРИ основного g, но в конце, чтобы был поверх */}
      {shouldRenderMarker && (
        <g className="flight-path-marker">
          <image
            href="/assets/markers/FP.svg"
            x={iconX}
            y={iconY}
            width={iconSize}
            height={iconSize}
            preserveAspectRatio="xMidYMid meet"
            style={{ pointerEvents: 'none' }}
          />
        </g>
      )}
      </g>
    </>
  );
};

export { Hex };
