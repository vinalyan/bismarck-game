// Компонент тумана войны для управления видимостью юнитов

import React, { useState, useEffect } from 'react';
import { NavalUnit } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';
import { movementAPI, movementUtils, useVisibility } from '../services/api/movementAPI';
import './FogOfWar.css';

interface FogOfWarProps {
  gameId: string;
  playerId: string;
  playerSide: 'german' | 'allied';
  allUnits: NavalUnit[]; // Все юниты в игре (для отладки)
  visibleUnits: NavalUnit[]; // Видимые юниты для игрока
  onUnitClick?: (unit: NavalUnit) => void;
  onHexClick?: (hex: HexCoordinate) => void;
  fogOfWarEnabled: boolean;
}

const FogOfWar: React.FC<FogOfWarProps> = ({
  gameId,
  playerId,
  playerSide,
  allUnits,
  visibleUnits,
  onUnitClick,
  onHexClick,
  fogOfWarEnabled
}) => {
  const [visibilityData, setVisibilityData] = useState<{
    visibleUnits: any[];
    lastKnownPositions: any[];
  }>({
    visibleUnits: [],
    lastKnownPositions: []
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { getVisibleUnits, updateVisibility, loading: visibilityLoading, error: visibilityError } = useVisibility(gameId, playerId);

  // Загружаем данные видимости при изменении игры или игрока
  useEffect(() => {
    if (fogOfWarEnabled) {
      loadVisibilityData();
    }
  }, [gameId, playerId, fogOfWarEnabled]);

  const loadVisibilityData = async () => {
    try {
      setLoading(true);
      setError(null);
      await getVisibleUnits();
    } catch (err: any) {
      setError(err.message || 'Failed to load visibility data');
    } finally {
      setLoading(false);
    }
  };

  // Фильтруем юниты по видимости
  const getVisibleUnitsForPlayer = (): NavalUnit[] => {
    if (!fogOfWarEnabled) {
      return allUnits; // Показываем все юниты если туман войны отключен
    }

    return allUnits.filter(unit => {
      // Свои юниты всегда видимы
      if (unit.owner === playerSide) {
        return true;
      }

      // Юниты противника видимы только если они обнаружены или преследуются
      const visibilityState = visibilityData.visibleUnits.find(vu => vu.unit_id === unit.id);
      if (visibilityState) {
        return movementUtils.isUnitVisible(unit.owner, playerSide, visibilityState.visibility);
      }

      return false;
    });
  };

  // Получаем последние известные позиции невидимых юнитов
  const getLastKnownPositions = () => {
    if (!fogOfWarEnabled) {
      return [];
    }

    return visibilityData.lastKnownPositions.filter(lkp => {
      // Показываем только позиции юнитов противника
      return lkp.owner !== playerSide;
    });
  };

  // Обработка клика по юниту
  const handleUnitClick = (unit: NavalUnit) => {
    if (onUnitClick) {
      onUnitClick(unit);
    }
  };

  // Обработка клика по гексу
  const handleHexClick = (hex: HexCoordinate) => {
    if (onHexClick) {
      onHexClick(hex);
    }
  };

  // Обновление видимости юнита
  const handleUpdateVisibility = async (unitId: string, visibility: 'unknown' | 'sighted' | 'shadowed', hex?: string) => {
    try {
      await updateVisibility(unitId, visibility, hex);
      await loadVisibilityData(); // Перезагружаем данные
    } catch (err: any) {
      setError(err.message || 'Failed to update visibility');
    }
  };

  // Получаем информацию о видимости юнита
  const getUnitVisibility = (unit: NavalUnit) => {
    if (unit.owner === playerSide) {
      return 'sighted'; // Свои юниты считаются "обнаруженными"
    }

    const visibilityState = visibilityData.visibleUnits.find(vu => vu.unit_id === unit.id);
    return visibilityState ? visibilityState.visibility : 'unknown';
  };

  // Получаем маркер видимости
  const getVisibilityMarker = (unit: NavalUnit) => {
    const visibility = getUnitVisibility(unit);
    return movementUtils.getVisibilityMarker(visibility);
  };

  // Получаем текст видимости
  const getVisibilityText = (unit: NavalUnit) => {
    const visibility = getUnitVisibility(unit);
    return movementUtils.getVisibilityText(visibility);
  };

  const visibleUnitsForPlayer = getVisibleUnitsForPlayer();
  const lastKnownPositions = getLastKnownPositions();

  return (
    <div className="fog-of-war">
      {/* Панель управления туманом войны */}
      <div className="fog-of-war-controls">
        <div className="fog-of-war-header">
          <h3>Туман войны</h3>
          <div className="fog-of-war-status">
            <span className={`status-indicator ${fogOfWarEnabled ? 'enabled' : 'disabled'}`}>
              {fogOfWarEnabled ? 'Включен' : 'Отключен'}
            </span>
          </div>
        </div>

        {/* Статистика видимости */}
        <div className="visibility-stats">
          <div className="stat-item">
            <span className="stat-label">Видимые юниты:</span>
            <span className="stat-value">{visibleUnitsForPlayer.length}</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Последние позиции:</span>
            <span className="stat-value">{lastKnownPositions.length}</span>
          </div>
        </div>

        {/* Кнопки управления */}
        <div className="fog-of-war-actions">
          <button 
            className="refresh-button" 
            onClick={loadVisibilityData}
            disabled={loading || visibilityLoading}
          >
            Обновить
          </button>
        </div>
      </div>

      {/* Ошибки */}
      {(error || visibilityError) && (
        <div className="error-message">
          {error || visibilityError}
        </div>
      )}

      {/* Загрузка */}
      {(loading || visibilityLoading) && (
        <div className="loading-message">
          Загрузка данных видимости...
        </div>
      )}

      {/* Список видимых юнитов */}
      <div className="visible-units-list">
        <h4>Видимые юниты</h4>
        {visibleUnitsForPlayer.length === 0 ? (
          <p className="no-units">Нет видимых юнитов</p>
        ) : (
          <div className="units-grid">
            {visibleUnitsForPlayer.map(unit => (
              <div 
                key={unit.id} 
                className={`unit-card ${unit.owner === playerSide ? 'own-unit' : 'enemy-unit'}`}
                onClick={() => handleUnitClick(unit)}
              >
                <div className="unit-header">
                  <span className="unit-type">{unit.type}</span>
                  <span className="unit-id">{unit.id}</span>
                </div>
                <div className="unit-position">
                  <span className="position-label">Позиция:</span>
                  <span className="position-value">{unit.position}</span>
                </div>
                <div className="unit-visibility">
                  <span className="visibility-label">Статус:</span>
                  <span className={`visibility-value ${getUnitVisibility(unit)}`}>
                    {getVisibilityText(unit)}
                  </span>
                </div>
                {getVisibilityMarker(unit) && (
                  <div className="visibility-marker">
                    {getVisibilityMarker(unit)}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Последние известные позиции */}
      {lastKnownPositions.length > 0 && (
        <div className="last-known-positions">
          <h4>Последние известные позиции</h4>
          <div className="positions-list">
            {lastKnownPositions.map(position => (
              <div key={position.unit_id} className="position-item">
                <div className="position-header">
                  <span className="unit-type">{position.unit_type}</span>
                  <span className="unit-id">{position.unit_id}</span>
                </div>
                <div className="position-details">
                  <span className="position-label">Последняя позиция:</span>
                  <span className="position-value">{position.position}</span>
                </div>
                <div className="position-time">
                  <span className="time-label">Время:</span>
                  <span className="time-value">
                    {new Date(position.last_seen_at).toLocaleTimeString()}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Панель управления видимостью (для отладки) */}
      {process.env.NODE_ENV === 'development' && (
        <div className="visibility-debug">
          <h4>Управление видимостью (отладка)</h4>
          <div className="debug-actions">
            {allUnits.filter(unit => unit.owner !== playerSide).map(unit => (
              <div key={unit.id} className="debug-unit">
                <span className="unit-info">{unit.type} - {unit.id}</span>
                <div className="debug-buttons">
                  <button 
                    className="debug-button sighted"
                    onClick={() => handleUpdateVisibility(unit.id, 'sighted', unit.position)}
                  >
                    Обнаружен
                  </button>
                  <button 
                    className="debug-button shadowed"
                    onClick={() => handleUpdateVisibility(unit.id, 'shadowed', unit.position)}
                  >
                    Преследуется
                  </button>
                  <button 
                    className="debug-button unknown"
                    onClick={() => handleUpdateVisibility(unit.id, 'unknown')}
                  >
                    Скрыть
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default FogOfWar;
