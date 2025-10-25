// Форма редактирования свойств структуры

import React, { useState, useEffect } from 'react';
import { StructureType, Structure } from '../types/editorTypes';

interface StructureFormProps {
  type: StructureType;
  hexIds: string[];
  onSubmit: (structure: Structure) => void;
  onCancel: () => void;
}

const StructureForm: React.FC<StructureFormProps> = ({ type, hexIds, onSubmit, onCancel }) => {
  const [name, setName] = useState('');
  const [portType, setPortType] = useState<'france' | 'norway'>('norway');
  const [canRefuel, setCanRefuel] = useState(true);
  const [canReloadTorpedoes, setCanReloadTorpedoes] = useState(true);
  const [allowedSide, setAllowedSide] = useState<'german' | 'allied' | 'both'>('german');
  const [direction, setDirection] = useState<'ew' | 'ns'>('ew');

  useEffect(() => {
    // Генерируем имя по умолчанию
    const defaultNames: Record<StructureType, string> = {
      port: 'Port',
      canal: 'Canal',
      convoy_route: 'Convoy Route',
      air_sector: 'Air Sector',
      english_channel: 'English Channel',
      restricted_dd: 'Restricted DD',
      non_game_hex: 'Non-Game Hex',
      land: 'Land Area',
    };
    setName(defaultNames[type]);
  }, [type]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    let structure: Structure;
    
    switch (type) {
      case 'port':
        structure = {
          type: 'port',
          hexIds,
          portType,
          name,
          canRefuel,
          canReloadTorpedoes,
        };
        break;
      case 'canal':
        structure = {
          type: 'canal',
          fromHex: hexIds[0],
          toHex: hexIds[1],
          allowedSide,
          name,
        };
        break;
      case 'convoy_route':
        structure = {
          type: 'convoy_route',
          hexIds,
          direction,
          name,
        };
        break;
      case 'air_sector':
        structure = {
          type: 'air_sector',
          hexIds,
          name,
        };
        break;
      case 'english_channel':
        structure = {
          type: 'english_channel',
          hexIds,
        };
        break;
      case 'restricted_dd':
        structure = {
          type: 'restricted_dd',
          hexIds,
        };
        break;
      case 'non_game_hex':
        structure = {
          type: 'non_game_hex',
          hexIds,
          name,
        };
        break;
      case 'land':
        structure = {
          type: 'land',
          hexIds,
          name,
        };
        break;
    }
    
    onSubmit(structure);
  };

  return (
    <form onSubmit={handleSubmit} style={{ padding: '15px', backgroundColor: '#fff', borderRadius: '5px' }}>
      <h3>Свойства структуры</h3>
      
      {/* Название */}
      <div style={{ marginBottom: '15px' }}>
        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
          Название:
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          style={{ width: '100%', padding: '8px', border: '1px solid #ddd', borderRadius: '4px' }}
          required
        />
      </div>

      {/* Специфичные поля для каждого типа */}
      {type === 'port' && (
        <>
          <div style={{ marginBottom: '15px' }}>
            <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
              Тип порта:
            </label>
            <select
              value={portType}
              onChange={(e) => setPortType(e.target.value as 'france' | 'norway')}
              style={{ width: '100%', padding: '8px', border: '1px solid #ddd', borderRadius: '4px' }}
            >
              <option value="norway">Норвегия</option>
              <option value="france">Франция</option>
            </select>
          </div>
          <div style={{ marginBottom: '15px' }}>
            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={canRefuel}
                onChange={(e) => setCanRefuel(e.target.checked)}
                style={{ marginRight: '8px' }}
              />
              Можно заправляться
            </label>
          </div>
          <div style={{ marginBottom: '15px' }}>
            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={canReloadTorpedoes}
                onChange={(e) => setCanReloadTorpedoes(e.target.checked)}
                style={{ marginRight: '8px' }}
              />
              Можно пополнять торпеды
            </label>
          </div>
        </>
      )}

      {type === 'canal' && (
        <div style={{ marginBottom: '15px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
            Сторона:
          </label>
          <select
            value={allowedSide}
            onChange={(e) => setAllowedSide(e.target.value as 'german' | 'allied' | 'both')}
            style={{ width: '100%', padding: '8px', border: '1px solid #ddd', borderRadius: '4px' }}
          >
            <option value="german">Немецкая</option>
            <option value="allied">Союзная</option>
            <option value="both">Обе</option>
          </select>
        </div>
      )}

      {type === 'convoy_route' && (
        <div style={{ marginBottom: '15px' }}>
          <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
            Направление:
          </label>
          <select
            value={direction}
            onChange={(e) => setDirection(e.target.value as 'ew' | 'ns')}
            style={{ width: '100%', padding: '8px', border: '1px solid #ddd', borderRadius: '4px' }}
          >
            <option value="ew">Восток-Запад</option>
            <option value="ns">Север-Юг</option>
          </select>
        </div>
      )}

      {/* Кнопки */}
      <div style={{ display: 'flex', gap: '10px' }}>
        <button
          type="submit"
          style={{
            flex: 1,
            padding: '10px',
            backgroundColor: '#4caf50',
            color: '#fff',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
        >
          Создать
        </button>
        <button
          type="button"
          onClick={onCancel}
          style={{
            flex: 1,
            padding: '10px',
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
    </form>
  );
};

export default StructureForm;

