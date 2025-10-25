// Главный компонент редактора карты

import React, { useState, useCallback } from 'react';
import HexMapEditor from './components/HexMapEditor';
import StructurePanel from './components/StructurePanel';
import StructureForm from './components/StructureForm';
import { HexCoordinate } from './types/mapTypes';
import { StructureType, Structure, MapStructures, STRUCTURE_COLORS } from './types/editorTypes';
import { 
  exportToJSON, 
  importFromJSON, 
  downloadJSON, 
  readJSONFile 
} from './utils/structureUtils';
import './App.css';

interface MapSettings {
  startX: number;
  startY: number;
  mapWidth: number;
  mapHeight: number;
  backgroundWidth: number;
  backgroundHeight: number;
}

const App: React.FC = () => {
  const [selectedStructureType, setSelectedStructureType] = useState<StructureType | null>(null);
  const [selectedHexes, setSelectedHexes] = useState<string[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [mapSettings, setMapSettings] = useState<MapSettings>({
    startX: 21,
    startY: 0,
    mapWidth: 1673,
    mapHeight: 1500,
    backgroundWidth: 1681,
    backgroundHeight: 1430,
  });
  const [structures, setStructures] = useState<MapStructures>({
    ports: [],
    canals: [],
    convoyRoutes: [],
    airSectors: [],
    nonGameHexes: [],
    landAreas: [],
  });

  // Обработка клика по гексу
  const handleHexClick = useCallback((hex: HexCoordinate) => {
    if (!selectedStructureType) return;

    const hexId = `${hex.letter}${hex.number}`;
    
    // Для канала - только 2 гекса
    if (selectedStructureType === 'canal') {
      if (selectedHexes.length === 0) {
        setSelectedHexes([hexId]);
      } else if (selectedHexes.length === 1 && selectedHexes[0] !== hexId) {
        setSelectedHexes([...selectedHexes, hexId]);
      }
      return;
    }

    // Для остальных - множество гексов
    if (selectedHexes.includes(hexId)) {
      setSelectedHexes(selectedHexes.filter(id => id !== hexId));
    } else {
      setSelectedHexes([...selectedHexes, hexId]);
    }
  }, [selectedStructureType, selectedHexes]);

  // Сохранение структуры
  const handleSaveStructure = useCallback(() => {
    if (!selectedStructureType || selectedHexes.length === 0) return;
    
    // Для канала проверяем что выбрано 2 гекса
    if (selectedStructureType === 'canal' && selectedHexes.length !== 2) {
      alert('Для канала нужно выбрать ровно 2 гекса');
      return;
    }
    
    setShowForm(true);
  }, [selectedStructureType, selectedHexes]);

  // Создание структуры из формы
  const handleFormSubmit = useCallback((structure: Structure) => {
    setStructures(prev => {
      switch (structure.type) {
        case 'port':
          return { ...prev, ports: [...prev.ports, structure] };
        case 'canal':
          return { ...prev, canals: [...prev.canals, structure] };
        case 'convoy_route':
          return { ...prev, convoyRoutes: [...prev.convoyRoutes, structure] };
        case 'air_sector':
          return { ...prev, airSectors: [...prev.airSectors, structure] };
        case 'english_channel':
          return { ...prev, englishChannel: structure };
        case 'restricted_dd':
          return { ...prev, restrictedDD: structure };
        case 'non_game_hex':
          return { ...prev, nonGameHexes: [...prev.nonGameHexes, structure] };
        case 'land':
          return { ...prev, landAreas: [...prev.landAreas, structure] };
      }
    });
    
    setSelectedHexes([]);
    setSelectedStructureType(null);
    setShowForm(false);
  }, []);

  // Отмена
  const handleCancel = useCallback(() => {
    setSelectedHexes([]);
    setSelectedStructureType(null);
    setShowForm(false);
  }, []);

  // Экспорт
  const handleExport = useCallback(() => {
    const json = exportToJSON(structures);
    downloadJSON(json, 'map-structures.json');
  }, [structures]);

  // Импорт
  const handleImport = useCallback(async (file: File) => {
    try {
      const json = await readJSONFile(file);
      const imported = importFromJSON(json);
      setStructures(imported);
      alert('Структуры успешно импортированы!');
    } catch (error) {
      alert(`Ошибка импорта: ${error}`);
    }
  }, []);

  // Подготовка данных для визуализации сохраненных структур
  const savedStructures = React.useMemo(() => {
    const result: Array<{ hexIds: string[]; color: string }> = [];
    
    structures.ports.forEach(port => {
      result.push({ hexIds: port.hexIds, color: STRUCTURE_COLORS.port });
    });
    
    structures.canals.forEach(canal => {
      result.push({ hexIds: [canal.fromHex, canal.toHex], color: STRUCTURE_COLORS.canal });
    });
    
    structures.convoyRoutes.forEach(route => {
      result.push({ hexIds: route.hexIds, color: STRUCTURE_COLORS.convoy_route });
    });
    
    structures.airSectors.forEach(sector => {
      result.push({ hexIds: sector.hexIds, color: STRUCTURE_COLORS.air_sector });
    });
    
    if (structures.englishChannel) {
      result.push({ hexIds: structures.englishChannel.hexIds, color: STRUCTURE_COLORS.english_channel });
    }
    
    if (structures.restrictedDD) {
      result.push({ hexIds: structures.restrictedDD.hexIds, color: STRUCTURE_COLORS.restricted_dd });
    }
    
    structures.nonGameHexes.forEach(hex => {
      result.push({ hexIds: hex.hexIds, color: STRUCTURE_COLORS.non_game_hex });
    });
    
    structures.landAreas.forEach(land => {
      result.push({ hexIds: land.hexIds, color: STRUCTURE_COLORS.land });
    });
    
    return result;
  }, [structures]);

  return (
    <div style={{ 
      display: 'flex', 
      height: '100vh', 
      overflow: 'hidden'
    }}>
      {/* Панель управления */}
      <StructurePanel
        selectedType={selectedStructureType}
        onSelectType={setSelectedStructureType}
        selectedHexes={selectedHexes}
        onSave={handleSaveStructure}
        onCancel={handleCancel}
        onExport={handleExport}
        onImport={handleImport}
        structures={structures}
        mapSettings={mapSettings}
        onMapSettingsChange={setMapSettings}
      />

      {/* Карта */}
      <div style={{ 
        flex: 1, 
        position: 'relative',
        overflow: 'auto'
      }}>
        <div style={{ 
          width: `${mapSettings.mapWidth}px`,
          height: `${mapSettings.mapHeight}px`,
          position: 'relative',
          backgroundColor: '#fff'
        }}>
          <HexMapEditor
            onHexClick={handleHexClick}
            selectedHexIds={selectedHexes}
            selectionColor={selectedStructureType ? STRUCTURE_COLORS[selectedStructureType] : undefined}
            savedStructures={savedStructures}
            mapSettings={mapSettings}
          />
        </div>
      </div>

      {/* Форма создания структуры */}
      {showForm && selectedStructureType && (
        <div style={{
          position: 'fixed',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          zIndex: 1000,
          backgroundColor: '#fff',
          padding: '20px',
          borderRadius: '8px',
          boxShadow: '0 4px 20px rgba(0,0,0,0.3)',
          maxWidth: '400px',
          width: '90%',
        }}>
          <StructureForm
            type={selectedStructureType}
            hexIds={selectedHexes}
            onSubmit={handleFormSubmit}
            onCancel={handleCancel}
          />
        </div>
      )}
    </div>
  );
};

export default App;

