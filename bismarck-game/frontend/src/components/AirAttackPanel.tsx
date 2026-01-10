import React, { useState, useEffect } from 'react';
import { airAttackAPI, AirAttackMarkers, AirAttackTargetsResponse } from '../services/api/airAttackAPI';
import AirAttackModal from './AirAttackModal';
import './AirAttackPanel.css';

interface AirAttackPanelProps {
  gameId: string;
  authToken: string;
  currentPhase: string;
  onRefresh?: () => void;
  onHasPendingAttacks?: (hasPending: boolean) => void;
}

const AirAttackPanel: React.FC<AirAttackPanelProps> = ({
  gameId,
  authToken,
  currentPhase,
  onRefresh,
  onHasPendingAttacks,
}) => {
  // #region agent log
  useEffect(() => {
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackPanel.tsx:18',
        message: 'AirAttackPanel rendered',
        data: { gameId, currentPhase, hasAuthToken: !!authToken, hasOnRefresh: !!onRefresh },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H2'
      })
    }).catch(() => {});
  }, [gameId, currentPhase, authToken, onRefresh]);
  // #endregion
  
  const [markers, setMarkers] = useState<AirAttackMarkers>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedHex, setSelectedHex] = useState<string | null>(null);
  const [showExecuteModal, setShowExecuteModal] = useState(false);
  const [targets, setTargets] = useState<AirAttackTargetsResponse | null>(null);

  // Загружаем маркеры при монтировании и обновлении фазы
  useEffect(() => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackPanel.tsx:27',
        message: 'useEffect triggered',
        data: { currentPhase, gameId },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H2'
      })
    }).catch(() => {});
    // #endregion
    if (currentPhase === 'air_attack' || currentPhase === 'movement') {
      loadMarkers();
    }
  }, [gameId, currentPhase, authToken]);

  // Отслеживаем изменения маркеров и уведомляем родителя о наличии невыполненных атак
  useEffect(() => {
    if (currentPhase === 'air_attack' && onHasPendingAttacks) {
      const hasPending = Object.keys(markers).length > 0 && 
        Object.values(markers).some((count: number) => count > 0);
      onHasPendingAttacks(hasPending);
    } else if (currentPhase !== 'air_attack' && onHasPendingAttacks) {
      // Сбрасываем состояние, если фаза не air_attack
      onHasPendingAttacks(false);
    }
  }, [markers, currentPhase, onHasPendingAttacks]);

  const loadMarkers = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await airAttackAPI.getMarkers(gameId, authToken);
      
      // Фильтруем маркеры: проверяем, есть ли в каждом гексе корабли
      // Если кораблей нет - игнорируем маркер (не показываем его в списке)
      const filteredMarkers: Record<string, number> = {};
      
      if (response && typeof response === 'object') {
        for (const [hexId, count] of Object.entries(response)) {
          if (count > 0) {
            // Проверяем, есть ли в гексе корабли
            try {
              const targetsResponse = await airAttackAPI.getTargets(gameId, hexId, authToken);
              // Если есть цели (корабли), добавляем маркер
              if (targetsResponse && targetsResponse.targets && Array.isArray(targetsResponse.targets) && targetsResponse.targets.length > 0) {
                filteredMarkers[hexId] = count;
              }
              // Если кораблей нет - игнорируем маркер (не добавляем в filteredMarkers)
            } catch (err) {
              // Если ошибка при проверке - игнорируем маркер (для безопасности)
              console.warn(`Failed to check targets in hex ${hexId}:`, err);
            }
          }
        }
      }
      
      setMarkers(filteredMarkers);
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки маркеров');
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteAttack = async (hexId: string) => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackPanel.tsx:66',
        message: 'handleExecuteAttack called',
        data: { gameId, hexId, currentPhase },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H3'
      })
    }).catch(() => {});
    // #endregion
    try {
      setLoading(true);
      setError(null);
      
      // Загружаем цели в гексе
      const targetsResponse = await airAttackAPI.getTargets(gameId, hexId, authToken);
      
      // Проверяем, что ответ получен и targets является массивом
      if (!targetsResponse || !targetsResponse.targets || !Array.isArray(targetsResponse.targets) || targetsResponse.targets.length === 0) {
        // #region agent log
        fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            location: 'AirAttackPanel.tsx:89',
            message: 'No targets found in hex',
            data: { gameId, hexId },
            timestamp: Date.now(),
            sessionId: 'debug-session',
            runId: 'run1',
            hypothesisId: 'H3'
          })
        }).catch(() => {});
        // #endregion
        setError('В гексе нет доступных целей для атаки');
        return;
      }

      setTargets(targetsResponse);
      setSelectedHex(hexId);
      setShowExecuteModal(true);
    } catch (err: any) {
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackPanel.tsx:110',
          message: 'handleExecuteAttack error',
          data: { gameId, hexId, error: err.message || String(err) },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H3'
        })
      }).catch(() => {});
      // #endregion
      setError(err.message || 'Ошибка загрузки целей');
    } finally {
      setLoading(false);
    }
  };

  const handleConfirmExecute = async (targetId: string, targetClass: string) => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackPanel.tsx:88',
        message: 'handleConfirmExecute called',
        data: { gameId, selectedHex, targetId, targetClass },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H4'
      })
    }).catch(() => {});
    // #endregion
    if (!selectedHex) {
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackPanel.tsx:96',
          message: 'No selectedHex - returning early',
          data: { gameId, targetId, targetClass },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H4'
        })
      }).catch(() => {});
      // #endregion
      return;
    }

    try {
      setLoading(true);
      setError(null);

      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackPanel.tsx:106',
          message: 'Calling executeAttack API',
          data: { gameId, selectedHex, targetId, targetClass },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H5'
        })
      }).catch(() => {});
      // #endregion
      const result = await airAttackAPI.executeAttack(
        gameId,
        selectedHex,
        targetId,
        targetClass,
        authToken
      );
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackPanel.tsx:117',
          message: 'executeAttack API response received',
          data: { gameId, selectedHex, targetId, success: result.success, error: result.error, data: result.data },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H5'
        })
      }).catch(() => {});
      // #endregion

      if (result.success) {
        // Обновляем маркеры после выполнения атаки
        await loadMarkers();
        
        // Уведомляем родительский компонент об обновлении всех данных (юниты, маркеры, события)
        if (onRefresh) {
          await onRefresh();
        }

        // Обновляем GameLog - отправляем событие обновления
        window.dispatchEvent(new CustomEvent('gameLogRefresh'));

        // Событие выполненной атаки автоматически логируется на сервере и появится в игровом логе
        if (result.data) {
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
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackPanel.tsx:181',
        message: 'Rendering air_attack phase panel',
        data: { gameId, currentPhase, markersCount: Object.keys(markers).length, markers, loading, error, showExecuteModal, selectedHex },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H2'
      })
    }).catch(() => {});
    // #endregion
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
                    onClick={() => {
                      // #region agent log
                      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                          location: 'AirAttackPanel.tsx:209',
                          message: 'Execute attack button clicked',
                          data: { gameId, hexId, count },
                          timestamp: Date.now(),
                          sessionId: 'debug-session',
                          runId: 'run1',
                          hypothesisId: 'H3'
                        })
                      }).catch(() => {});
                      // #endregion
                      handleExecuteAttack(hexId);
                    }}
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
          <>
            {/* #region agent log */}
            {(() => {
              fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  location: 'AirAttackPanel.tsx:253',
                  message: 'AirAttackModal should render',
                  data: { gameId, selectedHex, targetsCount: (targets && Array.isArray(targets.targets)) ? targets.targets.length : 0, showExecuteModal, hasTargets: !!targets },
                  timestamp: Date.now(),
                  sessionId: 'debug-session',
                  runId: 'run1',
                  hypothesisId: 'H4'
                })
              }).catch(() => {});
              return null;
            })()}
            {/* #endregion */}
            <AirAttackModal
              gameId={gameId}
              hexId={selectedHex}
              authToken={authToken}
              onExecute={handleConfirmExecute}
              onCancel={handleCancelExecute}
            />
          </>
        )}
      </>
    );
  }

  // Для других фаз не показываем панель
  return null;
};

export default AirAttackPanel;
