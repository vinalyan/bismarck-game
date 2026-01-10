import React, { useState, useEffect } from 'react';
import { airAttackAPI, AirAttackTarget } from '../services/api/airAttackAPI';
import './AirAttackModal.css';

interface AirAttackModalProps {
  gameId: string;
  hexId: string;
  authToken: string;
  onExecute: (targetId: string, targetClass: string) => void;
  onCancel: () => void;
}

const AirAttackModal: React.FC<AirAttackModalProps> = ({
  gameId,
  hexId,
  authToken,
  onExecute,
  onCancel,
}) => {
  const [targets, setTargets] = useState<AirAttackTarget[]>([]);
  const [selectedTarget, setSelectedTarget] = useState<AirAttackTarget | null>(null);
  const [selectedTargetClass, setSelectedTargetClass] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackModal.tsx:26',
        message: 'AirAttackModal useEffect triggered',
        data: { gameId, hexId },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H4'
      })
    }).catch(() => {});
    // #endregion
    loadTargets();
  }, [gameId, hexId]);

  const loadTargets = async () => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackModal.tsx:40',
        message: 'AirAttackModal.loadTargets called',
        data: { gameId, hexId },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H4'
      })
    }).catch(() => {});
    // #endregion
    try {
      setLoading(true);
      setError(null);
      const response = await airAttackAPI.getTargets(gameId, hexId, authToken);
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackModal.tsx:52',
          message: 'AirAttackModal.getTargets response',
          data: { gameId, hexId, targetsCount: response?.targets?.length || 0, targets: response?.targets },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H4'
        })
      }).catch(() => {});
      // #endregion
      if (response) {
        setTargets(response.targets);
      } else {
        setError('Не удалось загрузить цели');
      }
    } catch (err: any) {
      // #region agent log
      fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          location: 'AirAttackModal.tsx:67',
          message: 'AirAttackModal.loadTargets error',
          data: { gameId, hexId, error: err.message || String(err) },
          timestamp: Date.now(),
          sessionId: 'debug-session',
          runId: 'run1',
          hypothesisId: 'H4'
        })
      }).catch(() => {});
      // #endregion
      setError(err.message || 'Ошибка загрузки целей');
    } finally {
      setLoading(false);
    }
  };

  const handleTargetSelect = (target: AirAttackTarget) => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackModal.tsx:77',
        message: 'Target selected in modal',
        data: { targetId: target.unit_id, targetClass: target.class, targetName: target.unit_name },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H4'
      })
    }).catch(() => {});
    // #endregion
    setSelectedTarget(target);
    setSelectedTargetClass(target.class);
  };

  const handleExecute = () => {
    // #region agent log
    fetch('http://127.0.0.1:7243/ingest/69ca24e2-ee3f-4810-9484-4f8bdf98479e', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        location: 'AirAttackModal.tsx:92',
        message: 'Modal handleExecute called',
        data: { selectedTargetId: selectedTarget?.unit_id, selectedTargetClass, hasSelectedTarget: !!selectedTarget },
        timestamp: Date.now(),
        sessionId: 'debug-session',
        runId: 'run1',
        hypothesisId: 'H4'
      })
    }).catch(() => {});
    // #endregion
    if (selectedTarget) {
      onExecute(selectedTarget.unit_id || '', selectedTargetClass);
    }
  };

  // Группируем цели по классам
  const targetsByClass = targets.reduce((acc, target) => {
    if (!acc[target.class]) {
      acc[target.class] = [];
    }
    acc[target.class].push(target);
    return acc;
  }, {} as Record<string, AirAttackTarget[]>);

  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content air-attack-modal" onClick={(e) => e.stopPropagation()}>
        <h3>Выбор цели для воздушной атаки</h3>
        <p className="hex-info">Гекс: {hexId}</p>

        {loading && <div className="loading">Загрузка целей...</div>}
        
        {error && <div className="error">{error}</div>}

        {!loading && !error && targets.length === 0 && (
          <div className="no-targets">В гексе нет доступных целей для воздушной атаки</div>
        )}

        {!loading && !error && targets.length > 0 && (
          <>
            <div className="targets-section">
              <h4>Выберите класс корабля для атаки:</h4>
              
              {Object.keys(targetsByClass).map((shipClass) => {
                const classTargets = targetsByClass[shipClass];
                const firstTarget = classTargets[0];
                
                return (
                  <div key={shipClass} className="target-class-group">
                    <label className={`target-radio ${selectedTargetClass === shipClass ? 'selected' : ''}`}>
                      <input
                        type="radio"
                        name="targetClass"
                        value={shipClass}
                        checked={selectedTargetClass === shipClass}
                        onChange={() => {
                          setSelectedTargetClass(shipClass);
                          // Выбираем первый корабль этого класса как пример
                          setSelectedTarget(firstTarget);
                        }}
                      />
                      <span className="target-class-name">{shipClass}</span>
                      <span className="target-count">({classTargets.length} кораблей)</span>
                    </label>

                    {selectedTargetClass === shipClass && (
                      <div className="target-ships-list">
                        <h5>Выберите конкретный корабль (защищающийся игрок выберет окончательную цель):</h5>
                        {classTargets.map((target) => (
                          <label
                            key={target.unit_id || target.task_force_id}
                            className={`target-ship-radio ${selectedTarget?.unit_id === target.unit_id ? 'selected' : ''}`}
                          >
                            <input
                              type="radio"
                              name="targetShip"
                              checked={selectedTarget?.unit_id === target.unit_id}
                              onChange={() => handleTargetSelect(target)}
                            />
                            <span className="target-name">
                              {target.unit_name || target.task_force_name}
                            </span>
                            <span className="target-info">
                              HULL: {target.current_hull}/{target.max_hull} | 
                              Видимость: {target.visibility}
                            </span>
                          </label>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

            <div className="modal-actions">
              <button
                className="btn-cancel"
                onClick={onCancel}
              >
                Отмена
              </button>
              <button
                className="btn-confirm"
                onClick={handleExecute}
                disabled={!selectedTarget}
              >
                Выполнить атаку
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default AirAttackModal;
