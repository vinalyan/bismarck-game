import React, { useState, useEffect } from 'react';
import { GameTurn, PhaseRecord, GamePhase, PhaseStatus, getPhaseConfig, getPhaseStatusText, getPhaseStatusColor, getPhaseSequence } from '../types/phaseTypes';
import { phaseAPI } from '../services/api/phaseAPI';
import './PhasePanel.css';

interface PhasePanelProps {
  gameId: string;
  currentTurn?: GameTurn;
  onPhaseChange?: (phase: GamePhase) => void;
  currentUserId?: string;
  currentGame?: {
    player1_id: string;
    player2_id?: string;
    status: string;
  };
}

const PhasePanel: React.FC<PhasePanelProps> = ({ gameId, currentTurn, onPhaseChange, currentUserId, currentGame }) => {
  const [phaseRecords, setPhaseRecords] = useState<PhaseRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Загружаем записи о фазах при изменении хода
  useEffect(() => {
    if (currentTurn) {
      loadPhaseRecords(currentTurn.turn_number);
    }
  }, [currentTurn]);

  const loadPhaseRecords = async (turnNumber: number) => {
    try {
      setLoading(true);
      setError(null);
      const records = await phaseAPI.getPhaseRecords(gameId, turnNumber);
      setPhaseRecords(records);
    } catch (err) {
      setError('Ошибка загрузки фаз');
      console.error('Error loading phase records:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleStartPhase = async (phase: GamePhase) => {
    if (!currentTurn) return;
    
    try {
      setLoading(true);
      await phaseAPI.startPhase({
        game_id: gameId,
        turn: currentTurn.turn_number,
        phase: phase,
      });
      
      // Перезагружаем записи о фазах
      await loadPhaseRecords(currentTurn.turn_number);
      
      if (onPhaseChange) {
        onPhaseChange(phase);
      }
    } catch (err) {
      setError('Ошибка начала фазы');
      console.error('Error starting phase:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCompletePhase = async (phase: GamePhase) => {
    if (!currentTurn) return;
    
    try {
      setLoading(true);
      await phaseAPI.completePhase({
        game_id: gameId,
        turn: currentTurn.turn_number,
        phase: phase,
      });
      
      // Перезагружаем записи о фазах
      await loadPhaseRecords(currentTurn.turn_number);
    } catch (err) {
      setError('Ошибка завершения фазы');
      console.error('Error completing phase:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleNextPhase = async () => {
    try {
      setLoading(true);
      await phaseAPI.nextPhase({ game_id: gameId });
      
      // Перезагружаем записи о фазах
      if (currentTurn) {
        await loadPhaseRecords(currentTurn.turn_number);
      }
    } catch (err) {
      setError('Ошибка перехода к следующей фазе');
      console.error('Error advancing to next phase:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleStartTurn = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Начинаем первый ход
      const newTurn = await phaseAPI.startTurn({ game_id: gameId });
      
      if (onPhaseChange) {
        onPhaseChange('setup' as GamePhase);
      }
      
      // Перезагружаем записи о фазах для нового хода
      if (newTurn && newTurn.turn_number) {
        await loadPhaseRecords(newTurn.turn_number);
      } else {
        // Если newTurn не содержит turn_number, загружаем ход 1
        await loadPhaseRecords(1);
      }
    } catch (err) {
      setError('Ошибка начала хода');
      console.error('Error starting turn:', err);
    } finally {
      setLoading(false);
    }
  };

  const getPhaseRecord = (phase: GamePhase): PhaseRecord | undefined => {
    return phaseRecords.find(record => record.phase === phase);
  };

  const canStartPhase = (phase: GamePhase): boolean => {
    const record = getPhaseRecord(phase);
    return record?.status === 'pending';
  };

  const canCompletePhase = (phase: GamePhase): boolean => {
    const record = getPhaseRecord(phase);
    return record?.status === 'active';
  };

  const isCurrentPhase = (phase: GamePhase): boolean => {
    return currentTurn?.current_phase === phase;
  };

  // Определяем, может ли текущий игрок начать ход
  const canStartTurn = (): boolean => {
    if (!currentUserId || !currentGame) return false;
    
    // Только немецкий игрок (Player1) может начать первый ход
    const isPlayer1 = currentGame.player1_id === currentUserId;
    const isGameReady = currentGame.status === 'active' && !!currentGame.player2_id;
    const hasNoActiveTurn = !currentTurn;
    
    // Проверяем, что это первый ход (нет завершенных ходов)
    const isFirstTurn = !currentGame.current_turn || currentGame.current_turn === 0;
    
    console.log('PhasePanel canStartTurn check:', {
      currentUserId,
      player1Id: currentGame.player1_id,
      isPlayer1,
      isGameReady,
      hasNoActiveTurn,
      isFirstTurn,
      currentTurn: currentGame.current_turn,
      canStart: isPlayer1 && isGameReady && hasNoActiveTurn && isFirstTurn
    });
    
    return isPlayer1 && isGameReady && hasNoActiveTurn && isFirstTurn;
  };

  if (!currentTurn) {
    return (
      <div className="phase-panel">
        <h3>Фазы игры</h3>
        {canStartTurn() ? (
          <div className="start-turn-section">
            <p>Игра готова к началу. Нажмите кнопку, чтобы начать первый ход.</p>
            <button
              className="action-button start-turn-button"
              onClick={handleStartTurn}
              disabled={loading}
            >
              Начать ход 1
            </button>
          </div>
        ) : (
          <p>Ожидание начала хода...</p>
        )}
      </div>
    );
  }

  const phases = getPhaseSequence(currentTurn.turn_number);

  return (
    <div className="phase-panel">
      <div className="phase-panel-header">
        <h3>Фазы игры</h3>
        <div className="turn-info">
          <span className="turn-number">Ход {currentTurn.turn_number}</span>
          <span className="current-phase">
            {getPhaseConfig(currentTurn.current_phase)?.name || currentTurn.current_phase}
          </span>
          {currentTurn.start_time && (
            <span className="turn-start-time">
              Начат: {new Date(currentTurn.start_time).toLocaleTimeString()}
            </span>
          )}
          <span className="turn-status">
            Статус: {currentTurn.status === 'active' ? 'Активен' : currentTurn.status}
          </span>
        </div>
      </div>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      <div className="phases-list">
        {phases.map((phase) => {
          const config = getPhaseConfig(phase);
          const record = getPhaseRecord(phase);
          const status = record?.status || 'pending';
          const isCurrent = isCurrentPhase(phase);
          const canStart = canStartPhase(phase);
          const canComplete = canCompletePhase(phase);

          return (
            <div
              key={phase}
              className={`phase-item ${isCurrent ? 'current' : ''} ${status}`}
            >
              <div className="phase-header">
                <div className="phase-info">
                  <h4 className="phase-name">{config.name}</h4>
                  <p className="phase-description">{config.description}</p>
                </div>
                <div className="phase-status">
                  <span
                    className="status-badge"
                    style={{ backgroundColor: getPhaseStatusColor(status) }}
                  >
                    {getPhaseStatusText(status)}
                  </span>
                </div>
              </div>

              <div className="phase-actions">
                {canStart && (
                  <button
                    className="action-button start-button"
                    onClick={() => handleStartPhase(phase)}
                    disabled={loading}
                  >
                    Начать
                  </button>
                )}
                {canComplete && (
                  <button
                    className="action-button complete-button"
                    onClick={() => handleCompletePhase(phase)}
                    disabled={loading}
                  >
                    Завершить
                  </button>
                )}
              </div>

              {record && (record.start_time || record.end_time) && (
                <div className="phase-timing">
                  {record.start_time && (
                    <span className="start-time">
                      Начало: {new Date(record.start_time).toLocaleTimeString()}
                    </span>
                  )}
                  {record.end_time && (
                    <span className="end-time">
                      Конец: {new Date(record.end_time).toLocaleTimeString()}
                    </span>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="phase-panel-footer">
        <button
          className="next-phase-button"
          onClick={handleNextPhase}
          disabled={loading}
        >
          Следующая фаза
        </button>
      </div>
    </div>
  );
};

export default PhasePanel;
