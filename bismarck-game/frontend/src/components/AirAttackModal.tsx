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
    loadTargets();
  }, [gameId, hexId]);

  const loadTargets = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await airAttackAPI.getTargets(gameId, hexId, authToken);
      if (response) {
        setTargets(response.targets);
      } else {
        setError('Не удалось загрузить цели');
      }
    } catch (err: any) {
      setError(err.message || 'Ошибка загрузки целей');
    } finally {
      setLoading(false);
    }
  };

  const handleTargetSelect = (target: AirAttackTarget) => {
    setSelectedTarget(target);
    setSelectedTargetClass(target.class);
  };

  const handleExecute = () => {
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
