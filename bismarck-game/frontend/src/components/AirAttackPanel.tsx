import React, { useState, useEffect } from 'react';
import { airAttackAPI, AirAttackMarkers, AirAttackTargetsResponse } from '../services/api/airAttackAPI';
import AirAttackModal from './AirAttackModal';
import './AirAttackPanel.css';

interface AirAttackPanelProps {
  gameId: string;
  authToken: string;
  currentPhase: string;
  onRefresh?: () => void;
}

const AirAttackPanel: React.FC<AirAttackPanelProps> = ({
  gameId,
  authToken,
  currentPhase,
  onRefresh,
}) => {
  const [markers, setMarkers] = useState<AirAttackMarkers>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedHex, setSelectedHex] = useState<string | null>(null);
  const [showExecuteModal, setShowExecuteModal] = useState(false);
  const [targets, setTargets] = useState<AirAttackTargetsResponse | null>(null);

  // Загружаем маркеры при монтировании и обновлении фазы
  useEffect(() => {
    if (currentPhase === 'air_attack' || currentPhase === 'movement') {
      loadMarkers();
    }
  }, [gameId, currentPhase, authToken]);

  const loadMarkers = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await airAttackAPI.getMarkers(gameId, authToken);
      setMarkers(response || {});
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки маркеров');
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteAttack = async (hexId: string) => {
    try {
      setLoading(true);
      setError(null);
      
      // Загружаем цели в гексе
      const targetsResponse = await airAttackAPI.getTargets(gameId, hexId, authToken);
      if (!targetsResponse || targetsResponse.targets.length === 0) {
        setError('В гексе нет доступных целей для атаки');
        return;
      }

      setTargets(targetsResponse);
      setSelectedHex(hexId);
      setShowExecuteModal(true);
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки целей');
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmExecute = async (targetId: string, targetClass: string) => {
    if (!selectedHex) return;

    try {
      setLoading(true);
      setError(null);

      const result = await airAttackAPI.executeAttack(
        gameId,
        selectedHex,
        targetId,
        targetClass,
        authToken
      );

      if (result.success) {
        // Обновляем маркеры после выполнения атаки
        await loadMarkers();
        
        // Уведомляем родительский компонент об обновлении всех данных (юниты, маркеры, события)
        if (onRefresh) {
          await onRefresh();
        }

        // Обновляем GameLog - отправляем событие обновления
        window.dispatchEvent(new CustomEvent('gameLogRefresh'));

        // Показываем подробное сообщение об успехе
        if (result.data) {
          const message = result.data.sunk
            ? `✈️💥 Воздушная атака: ${result.data.target_name || 'Корабль'} потоплен в гексе ${selectedHex}!`
            : `✈️💥 Воздушная атака выполнена на ${result.data.target_name || 'корабль'} в гексе ${selectedHex}: нанесено 1 повреждение, новый HULL: ${result.data.new_hull}`;
          alert(message);
          
          console.log('✅ Air attack executed successfully:', {
            hexId: selectedHex,
            targetId,
            targetClass,
            result: result.data
          });
        }

        setShowExecuteModal(false);
        setSelectedHex(null);
        setTargets(null);
      } else {
        setError(result.error || 'Ошибка выполнения атаки');
        console.error('❌ Air attack failed:', result.error);
      }
    } catch (err: any) {
      setError(err.message || 'Ошибка выполнения атаки');
    } finally {
      setLoading(false);
    }
  };

  const handleCancelExecute = () => {
    setShowExecuteModal(false);
    setSelectedHex(null);
    setTargets(null);
  };

  // В фазе Movement показываем только список маркеров (добавление происходит через клик на гекс)
  if (currentPhase === 'movement') {
    const markerEntries = Object.entries(markers);
    
    return (
      <div className="air-attack-panel">
        <h4>Маркеры воздушной атаки</h4>
        {loading && <div className="loading">Загрузка...</div>}
        {error && <div className="error">{error}</div>}
        
        {!loading && !error && markerEntries.length === 0 && (
          <div className="no-markers">
            Нет маркеров воздушной атаки. Кликните на гекс с shadowed вражеским юнитом для добавления маркера.
          </div>
        )}

        {!loading && !error && markerEntries.length > 0 && (
          <div className="markers-list">
            {markerEntries.map(([hexId, count]) => (
              <div key={hexId} className="marker-item">
                <span className="hex-id">{hexId}</span>
                <span className="marker-count">×{count}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  // В фазе Air Attack показываем маркеры и кнопки для выполнения атак
  if (currentPhase === 'air_attack') {
    const markerEntries = Object.entries(markers);

    return (
      <>
        <div className="air-attack-panel">
          <h4>Воздушные атаки</h4>
          {loading && <div className="loading">Загрузка...</div>}
          {error && <div className="error">{error}</div>}

          {!loading && !error && markerEntries.length === 0 && (
            <div className="no-markers">
              Нет маркеров воздушной атаки для выполнения.
            </div>
          )}

          {!loading && !error && markerEntries.length > 0 && (
            <div className="attacks-list">
              {markerEntries.map(([hexId, count]) => (
                <div key={hexId} className="attack-item">
                  <div className="attack-info">
                    <span className="hex-id">{hexId}</span>
                    <span className="marker-count">×{count}</span>
                  </div>
                  <button
                    className="btn-execute"
                    onClick={() => handleExecuteAttack(hexId)}
                    disabled={loading}
                  >
                    Выполнить атаку
                  </button>
                </div>
              ))}
            </div>
          )}

          <div className="panel-info">
            <p className="info-text">
              Приоритет: Союзники выполняют атаки первыми, затем немцы.
              После выполнения атаки маркер удаляется автоматически.
            </p>
          </div>
        </div>

        {showExecuteModal && selectedHex && targets && (
          <AirAttackModal
            gameId={gameId}
            hexId={selectedHex}
            authToken={authToken}
            onExecute={handleConfirmExecute}
            onCancel={handleCancelExecute}
          />
        )}
      </>
    );
  }

  // Для других фаз не показываем панель
  return null;
};

export default AirAttackPanel;
