// Гексагональная карта для редактора структур

import React, { useState, useMemo } from 'react';
import { HexEditor } from './HexEditor';
import { HexCoordinate, HexData, offsetToCoordinate } from '../types/mapTypes';
import { 
  Point, OffsetCoord, offsetToPixel, offsetPolygonCorners, calculateMapSize, MAP_CONSTANTS
} from '../utils/hexUtils';
import './HexMap.css';

interface MapSettings {
  startX: number;
  startY: number;
  mapWidth: number;
  mapHeight: number;
  backgroundWidth: number;
  backgroundHeight: number;
}

interface HexMapEditorProps {
  width?: number;
  height?: number;
  onHexClick?: (hex: HexCoordinate) => void;
  selectedHexIds?: string[];
  selectionColor?: string;
  savedStructures?: Array<{ hexIds: string[]; color: string }>;
  mapSettings?: MapSettings;
}

const HexMapEditor: React.FC<HexMapEditorProps> = ({
  width = MAP_CONSTANTS.HEX_GRID_WIDTH,
  height = MAP_CONSTANTS.HEX_GRID_HEIGHT,
  onHexClick,
  selectedHexIds = [],
  selectionColor,
  savedStructures = [],
  mapSettings
}) => {
  const [mapOffset] = useState({ x: 0, y: 0 });
  const [hexRadius] = useState(MAP_CONSTANTS.DEFAULT_HEX_RADIUS);

  // Генерируем координаты гексов
  const hexes = useMemo(() => {
    const newHexes = new Map<string, HexData>();
    
    for (let row = 0; row < height; row++) {
      for (let col = 0; col < width; col++) {
        let letter: string;
        if (row < 26) {
          letter = String.fromCharCode(65 + row);
        } else {
          const secondLetterIndex = row - 26;
          letter = 'A' + String.fromCharCode(65 + secondLetterIndex);
        }
        const number = col + 1;
        
        const coordinate: HexCoordinate = {
          letter,
          number,
          col,
          row
        };
        
        const hexId = `${letter}${number}`;
        
        newHexes.set(hexId, {
          coordinate,
          type: 'water',
          isVisible: true,
          isHighlighted: false,
          hasUnit: false,
          weather: 'clear',
          fogLevel: 0
        });
      }
    }
    
    return newHexes;
  }, [width, height]);

  // Получаем размер карты
  const mapSize = useMemo(() => {
    if (mapSettings) {
      return { width: mapSettings.mapWidth, height: mapSettings.mapHeight };
    }
    return calculateMapSize(width, height, hexRadius);
  }, [width, height, hexRadius, mapSettings]);

  // Рендерим гексы
  const renderHexes = () => {
    const renderedHexes: JSX.Element[] = [];
    
    hexes.forEach((hexData, hexId) => {
      const offset: OffsetCoord = { col: hexData.coordinate.col, row: hexData.coordinate.row };
      const center = offsetToPixel(offset, hexRadius);
      const corners = offsetPolygonCorners(offset, hexRadius);
      
      const isSelected = selectedHexIds.includes(hexId);
      
      // Проверяем, принадлежит ли гекс сохраненной структуре
      let savedStructureColor: string | undefined;
      savedStructures.forEach(structure => {
        if (structure.hexIds.includes(hexId)) {
          savedStructureColor = structure.color;
        }
      });
      
      const handleClick = () => {
        if (onHexClick) {
          onHexClick(hexData.coordinate);
        }
      };
      
      const handleHover = () => {
        // Можно добавить логику подсветки при наведении
      };
      
      renderedHexes.push(
        <HexEditor
          key={hexId}
          coordinate={hexData.coordinate}
          hexData={hexData}
          center={center}
          corners={corners}
          size={hexRadius}
          isSelected={isSelected}
          selectionColor={isSelected ? selectionColor : savedStructureColor}
          onClick={handleClick}
          onHover={handleHover}
        />
      );
    });
    
    return renderedHexes;
  };

  return (
    <div className="hex-map-container" style={{ position: 'relative', width: '100%', height: '100%' }}>
      {/* Фоновое изображение карты */}
      <div 
        style={{
          position: 'absolute',
          top: mapSettings ? `-${mapSettings.startY}px` : 0,
          left: mapSettings ? `-${mapSettings.startX}px` : 0,
          width: mapSettings ? `${mapSettings.backgroundWidth}px` : '100%',
          height: mapSettings ? `${mapSettings.backgroundHeight}px` : '100%',
          backgroundImage: 'url(/allied-map.jpg)',
          backgroundSize: mapSettings ? `${mapSettings.backgroundWidth}px ${mapSettings.backgroundHeight}px` : 'auto',
          backgroundPosition: '0 0',
          backgroundRepeat: 'no-repeat',
          opacity: 0.7,
          zIndex: 0
        }}
      />
      
      {/* SVG с гексами */}
      <svg
        width={mapSize.width}
        height={mapSize.height}
        style={{
          position: 'relative',
          zIndex: 1,
          backgroundColor: 'transparent'
        }}
      >
        {/* Градиенты для разных типов гексов */}
        <defs>
          <linearGradient id="waterGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#c8e6f5" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#87ceeb" stopOpacity="0.3" />
          </linearGradient>
          <linearGradient id="landGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#8fbc8f" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#6b8e6b" stopOpacity="0.3" />
          </linearGradient>
          <linearGradient id="portGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#87ceeb" stopOpacity="0.4" />
            <stop offset="100%" stopColor="#4682b4" stopOpacity="0.4" />
          </linearGradient>
        </defs>
        
        <g transform={`translate(${mapOffset.x}, ${mapOffset.y})`}>
          {renderHexes()}
        </g>
      </svg>
    </div>
  );
};

export default HexMapEditor;

