// Панель управления структурами

import React from 'react';
import { StructureType, MapStructures, STRUCTURE_LABELS } from '../types/editorTypes';

interface MapSettings {
  startX: number;
  startY: number;
  mapWidth: number;
  mapHeight: number;
  backgroundWidth: number;
  backgroundHeight: number;
}

interface StructurePanelProps {
  selectedType: StructureType | null;
  onSelectType: (type: StructureType | null) => void;
  selectedHexes: string[];
  onSave: () => void;
  onCancel: () => void;
  onExport: () => void;
  onImport: (file: File) => void;
  structures: MapStructures;
  mapSettings: MapSettings;
  onMapSettingsChange: (settings: MapSettings) => void;
}

const StructurePanel: React.FC<StructurePanelProps> = ({
  selectedType,
  onSelectType,
  selectedHexes,
  onSave,
  onCancel,
  onExport,
  onImport,
  structures,
  mapSettings,
  onMapSettingsChange,
}) => {
  const handleImportClick = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (file) {
        onImport(file);
      }
    };
    input.click();
  };

  return (
    <div style={{
      width: '300px',
      padding: '20px',
      backgroundColor: '#f5f5f5',
      borderRight: '1px solid #ddd',
      overflowY: 'auto',
      height: '100vh',
    }}>
      <h2 style={{ marginTop: 0 }}>Редактор структур</h2>

      {/* Настройки карты */}
      <div style={{ marginBottom: '20px', padding: '10px', backgroundColor: '#fff', borderRadius: '5px' }}>
        <h3 style={{ marginTop: 0 }}>Настройки карты</h3>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Начало X:
          </label>
          <input
            type="number"
            value={mapSettings.startX}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, startX: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Начало Y:
          </label>
          <input
            type="number"
            value={mapSettings.startY}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, startY: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Ширина карты:
          </label>
          <input
            type="number"
            value={mapSettings.mapWidth}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, mapWidth: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Высота карты:
          </label>
          <input
            type="number"
            value={mapSettings.mapHeight}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, mapHeight: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Ширина подложки:
          </label>
          <input
            type="number"
            value={mapSettings.backgroundWidth}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, backgroundWidth: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
        
        <div style={{ marginBottom: '10px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontSize: '12px' }}>
            Высота подложки:
          </label>
          <input
            type="number"
            value={mapSettings.backgroundHeight}
            onChange={(e) => onMapSettingsChange({ ...mapSettings, backgroundHeight: Number(e.target.value) })}
            style={{ width: '100%', padding: '5px', fontSize: '12px' }}
          />
        </div>
      </div>

      {/* Кнопки выбора типа */}
      <div style={{ marginBottom: '20px' }}>
        <h3>Выберите структуру:</h3>
        {(['port', 'canal', 'convoy_route', 'air_sector', 'english_channel', 'restricted_dd', 'non_game_hex', 'land'] as StructureType[]).map(type => (
          <button
            key={type}
            onClick={() => onSelectType(type)}
            style={{
              display: 'block',
              width: '100%',
              padding: '10px',
              marginBottom: '8px',
              backgroundColor: selectedType === type ? '#2196f3' : '#fff',
              color: selectedType === type ? '#fff' : '#333',
              border: '1px solid #ddd',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            {STRUCTURE_LABELS[type]}
          </button>
        ))}
      </div>

      {/* Текущий выбор */}
      {selectedType && (
        <div style={{
          padding: '15px',
          backgroundColor: '#e3f2fd',
          borderRadius: '5px',
          marginBottom: '20px',
        }}>
          <p><strong>Выбран:</strong> {STRUCTURE_LABELS[selectedType]}</p>
          <p><strong>Гексов:</strong> {selectedHexes.length}</p>
          {selectedHexes.length > 0 && (
            <div style={{ marginTop: '10px' }}>
              <strong>Гексы:</strong>
              <div style={{ 
                maxHeight: '150px', 
                overflowY: 'auto',
                marginTop: '5px',
                padding: '5px',
                backgroundColor: '#fff',
                borderRadius: '3px',
                fontSize: '12px'
              }}>
                {selectedHexes.map(id => (
                  <span 
                    key={id} 
                    style={{ 
                      display: 'inline-block', 
                      margin: '2px',
                      padding: '2px 6px',
                      backgroundColor: '#e0e0e0',
                      borderRadius: '3px'
                    }}
                  >
                    {id}
                  </span>
                ))}
              </div>
            </div>
          )}
          
          <div style={{ marginTop: '15px', display: 'flex', gap: '8px' }}>
            <button 
              onClick={onSave} 
              disabled={selectedHexes.length === 0}
              style={{
                flex: 1,
                padding: '8px',
                backgroundColor: selectedHexes.length > 0 ? '#4caf50' : '#ccc',
                color: '#fff',
                border: 'none',
                borderRadius: '4px',
                cursor: selectedHexes.length > 0 ? 'pointer' : 'not-allowed',
              }}
            >
              Сохранить
            </button>
            <button 
              onClick={onCancel}
              style={{
                flex: 1,
                padding: '8px',
                backgroundColor: '#f44336',
                color: '#fff',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              Отмена
            </button>
          </div>
        </div>
      )}

      {/* Список созданных структур */}
      <div style={{ marginBottom: '20px' }}>
        <h3>Созданные структуры:</h3>
        <p>Портов: {structures.ports.length}</p>
        <p>Каналов: {structures.canals.length}</p>
        <p>Маршрутов: {structures.convoyRoutes.length}</p>
        <p>Секторов: {structures.airSectors.length}</p>
        <p>Не игровых гексов: {structures.nonGameHexes.length}</p>
        <p>Суши: {structures.landAreas.length}</p>
        {structures.englishChannel && <p>Ла-Манш: создан</p>}
        {structures.restrictedDD && <p>Ограничение: создано</p>}
      </div>

      {/* Экспорт/Импорт */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        <button 
          onClick={onExport}
          style={{
            width: '100%',
            padding: '10px',
            backgroundColor: '#4caf50',
            color: 'white',
            border: 'none',
            borderRadius: '5px',
            cursor: 'pointer',
          }}
        >
          Экспорт JSON
        </button>
        <button 
          onClick={handleImportClick}
          style={{
            width: '100%',
            padding: '10px',
            backgroundColor: '#2196f3',
            color: 'white',
            border: 'none',
            borderRadius: '5px',
            cursor: 'pointer',
          }}
        >
          Импорт JSON
        </button>
      </div>
    </div>
  );
};

export default StructurePanel;

