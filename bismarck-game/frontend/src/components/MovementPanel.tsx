// Компонент панели движения для выбора и подтверждения движения юнитов

import React, { useState, useEffect } from 'react';
import { NavalUnit } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';
import { movementAPI, movementUtils, useMovement } from '../services/api/movementAPI';
import './MovementPanel.css';

interface MovementPanelProps {
  selectedUnit: NavalUnit | null;
  gameId: string;
  playerId: string;
  onMove: (unitId: string, toHex: string) => void;
  onCancel: () => void;
  onHexSelect?: (hex: HexCoordinate) => void;
}

const MovementPanel: React.FC<MovementPanelProps> = ({
  selectedUnit,
  gameId,
  playerId,
  onMove,
  onCancel,
  onHexSelect
}) => {
  const [availableMoves, setAvailableMoves] = useState<string[]>([]);
  const [fuelCosts, setFuelCosts] = useState<Record<string, number>>({});
  const [selectedHex, setSelectedHex] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showConfirmation, setShowConfirmation] = useState(false);

  const { getAvailableMoves, moveUnit, loading: movementLoading, error: movementError } = useMovement(gameId, playerId);

  // Загружаем доступные ходы при выборе юнита
  useEffect(() => {
    if (selectedUnit) {
      loadAvailableMoves();
    } else {
      setAvailableMoves([]);
      setFuelCosts({});
      setSelectedHex(null);
      setShowConfirmation(false);
    }
  }, [selectedUnit]);

  const loadAvailableMoves = async () => {
    if (!selectedUnit) return;

    try {
      setLoading(true);
      setError(null);
      
      const response = await movementAPI.getAvailableMoves(gameId, selectedUnit.id);
      setAvailableMoves(response.available_hexes);
      setFuelCosts(response.fuel_costs);
    } catch (err: any) {
      setError(err.message || 'Failed to load available moves');
    } finally {
      setLoading(false);
    }
  };

  const handleHexClick = (hex: string) => {
    if (!availableMoves.includes(hex)) {
      return;
    }

    setSelectedHex(hex);
    setShowConfirmation(true);

    // Уведомляем родительский компонент о выборе гекса
    if (onHexSelect) {
      // Преобразуем строку гекса в HexCoordinate
      const coordinate = parseHexString(hex);
      if (coordinate) {
        onHexSelect(coordinate);
      }
    }
  };

  const handleConfirmMove = async () => {
    if (!selectedUnit || !selectedHex) return;

    try {
      setLoading(true);
      setError(null);

      const result = await movementAPI.moveUnit(gameId, selectedUnit.id, {
        unit_id: selectedUnit.id,
        to_hex: selectedHex
      });

      if (result.success) {
        onMove(selectedUnit.id, selectedHex);
        setShowConfirmation(false);
        setSelectedHex(null);
        // Перезагружаем доступные ходы
        await loadAvailableMoves();
      } else {
        setError(result.message || 'Movement failed');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to move unit');
    } finally {
      setLoading(false);
    }
  };

  const handleCancelMove = () => {
    setShowConfirmation(false);
    setSelectedHex(null);
    onCancel();
  };

  const parseHexString = (hexString: string): HexCoordinate | null => {
    // Парсим строку типа "K15" в HexCoordinate
    const match = hexString.match(/^([A-Z]+)(\d+)$/);
    if (!match) return null;

    const letter = match[1];
    const number = parseInt(match[2]);

    // Упрощенное преобразование в координаты
    const row = letter.length === 1 ? letter.charCodeAt(0) - 65 : (letter.charCodeAt(0) - 65) * 26 + (letter.charCodeAt(1) - 65);
    const col = number - 1;

    return {
      letter,
      number,
      row,
      col
    };
  };

  const getFuelCost = (hex: string): number => {
    return fuelCosts[hex] || 0;
  };

  const getSpeedClass = (): string => {
    if (!selectedUnit) return 'M';
    return movementUtils.getSpeedClass(selectedUnit.type);
  };

  const getMaxDistance = (): number => {
    return movementUtils.getMaxMovementDistance(getSpeedClass());
  };

  if (!selectedUnit) {
    return (
      <div className="movement-panel">
        <div className="movement-panel-header">
          <h3>Движение юнитов</h3>
        </div>
        <div className="movement-panel-content">
          <p>Выберите юнит для движения</p>
        </div>
      </div>
    );
  }

  return (
    <div className="movement-panel">
      <div className="movement-panel-header">
        <h3>Движение юнита</h3>
        <button className="close-button" onClick={onCancel}>×</button>
      </div>

      <div className="movement-panel-content">
        {/* Информация о юните */}
        <div className="unit-info">
          <h4>{selectedUnit.type} - {selectedUnit.id}</h4>
          <div className="unit-details">
            <div className="detail-item">
              <span>Текущая позиция:</span>
              <span className="detail-value">{selectedUnit.position}</span>
            </div>
            <div className="detail-item">
              <span>Класс скорости:</span>
              <span className="detail-value">{getSpeedClass()}</span>
            </div>
            <div className="detail-item">
              <span>Макс. расстояние:</span>
              <span className="detail-value">{getMaxDistance()} гекс</span>
            </div>
            <div className="detail-item">
              <span>Топливо:</span>
              <span className="detail-value">{selectedUnit.maxFuel || 0} FP</span>
            </div>
          </div>
        </div>

        {/* Ошибки */}
        {(error || movementError) && (
          <div className="error-message">
            {error || movementError}
          </div>
        )}

        {/* Загрузка */}
        {(loading || movementLoading) && (
          <div className="loading-message">
            Загрузка...
          </div>
        )}

        {/* Подтверждение движения */}
        {showConfirmation && selectedHex && (
          <div className="confirmation-dialog">
            <h4>Подтверждение движения</h4>
            <p>
              Переместить {selectedUnit.type} из {selectedUnit.position} в {selectedHex}?
            </p>
            <div className="movement-cost">
              <span>Стоимость топлива: {getFuelCost(selectedHex)} FP</span>
            </div>
            <div className="confirmation-buttons">
              <button 
                className="confirm-button" 
                onClick={handleConfirmMove}
                disabled={loading || movementLoading}
              >
                Подтвердить
              </button>
              <button 
                className="cancel-button" 
                onClick={handleCancelMove}
                disabled={loading || movementLoading}
              >
                Отмена
              </button>
            </div>
          </div>
        )}

        {/* Доступные ходы */}
        {!showConfirmation && (
          <div className="available-moves">
            <h4>Доступные ходы</h4>
            {availableMoves.length === 0 ? (
              <p>Нет доступных ходов</p>
            ) : (
              <div className="moves-grid">
                {availableMoves.map(hex => (
                  <button
                    key={hex}
                    className={`move-hex ${selectedHex === hex ? 'selected' : ''}`}
                    onClick={() => handleHexClick(hex)}
                    title={`Переместиться в ${hex} (${getFuelCost(hex)} FP)`}
                  >
                    <span className="hex-coordinate">{hex}</span>
                    <span className="fuel-cost">{getFuelCost(hex)} FP</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Кнопки управления */}
        <div className="movement-controls">
          <button 
            className="refresh-button" 
            onClick={loadAvailableMoves}
            disabled={loading || movementLoading}
          >
            Обновить ходы
          </button>
          <button 
            className="cancel-button" 
            onClick={onCancel}
            disabled={loading || movementLoading}
          >
            Отмена
          </button>
        </div>
      </div>
    </div>
  );
};

export default MovementPanel;
