import React, { useState, useEffect } from 'react';
import { GameTurn, PhaseRecord, GamePhase, PhaseStatus, getPhaseStatusText, getPhaseStatusColor, getPhaseSequence, PHASE_NAMES, PHASE_DESCRIPTIONS } from '../types/phaseTypes';
import { Game } from '../types/gameTypes';
import { phaseAPI } from '../services/api/phaseAPI';
import AirAttackPanel from './AirAttackPanel';
import './PhasePanel.css';

interface PhasePanelProps {
  gameId: string;
  currentTurn?: GameTurn;
  onPhaseChange?: (phase: GamePhase) => void;
  currentUserId?: string;
  currentGame?: Game;
  authToken?: string;
  onRefresh?: () => void;
}

const PhasePanel: React.FC<PhasePanelProps> = ({ gameId, currentTurn, onPhaseChange, currentUserId, currentGame, authToken, onRefresh }) => {
  const [phaseRecords, setPhaseRecords] = useState<PhaseRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Загружаем записи о фазах при изменении хода
  useEffect(() => {
    if (currentTurn && currentTurn.turn_number) {
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
      
      // Удален вызов phaseAPI.getCurrentPhase - информация о текущей фазе теперь приходит через GameModel
      // Родительский компонент должен обновить currentTurn из GameModel
      
      // Уведомляем родительский компонент об обновлении хода
      // Информация о текущей фазе будет обновлена через GameModel
      window.dispatchEvent(new CustomEvent('turnUpdated'));
      
      // Перезагружаем записи о фазах (используем текущий ход, если он есть)
      if (currentTurn && currentTurn.turn_number) {
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
      
      console.log('🔄 Starting new turn...');
      
      // Начинаем первый ход
      const newTurn = await phaseAPI.startTurn({ game_id: gameId });
      
      console.log('✅ New turn started:', {
        turn_number: newTurn.turn_number,
        current_phase: newTurn.current_phase,
        status: newTurn.status
      });
      
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
      
      // Удален вызов phaseAPI.getCurrentPhase - информация о текущей фазе теперь приходит через GameModel
      // Уведомляем родительский компонент об обновлении хода
      // Родительский компонент должен обновить currentTurn из GameModel
      window.dispatchEvent(new CustomEvent('turnUpdated'));
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
    
    // Проверяем, что игра еще не начата (turn: 0, phase: "setup")
    // Используем currentTurn напрямую, если он установлен, иначе используем currentGame
    // Кнопка должна появляться когда turn: 0 и phase: "setup"
    const turnNumber = currentTurn?.turn_number ?? currentGame.current_turn ?? 0;
    const phase = currentTurn?.current_phase ?? currentGame.current_phase ?? 'setup';
    
    const isGameNotStarted = turnNumber === 0 && phase === 'setup';
    
    return isPlayer1 && isGameReady && isGameNotStarted;
  };

  // Проверяем, нужно ли показать кнопку "Начать ход 1"
  // Кнопка должна появляться когда turn: 0 и phase: "setup" (игра еще не начата)
  const turnNumber = currentTurn?.turn_number ?? currentGame?.current_turn ?? 0;
  const phase = currentTurn?.current_phase ?? currentGame?.current_phase ?? 'setup';
  const shouldShowStartTurnButton = turnNumber === 0 && phase === 'setup';

  if (shouldShowStartTurnButton && canStartTurn()) {
    return (
      <div className="phase-panel">
        <h3>Фазы игры</h3>
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
      </div>
    );
  }

  if (!currentTurn) {
    return (
      <div className="phase-panel">
        <h3>Фазы игры</h3>
        <p>Ожидание начала хода...</p>
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
            {PHASE_NAMES[currentTurn.current_phase] || currentTurn.current_phase}
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
                  <h4 className="phase-name">{PHASE_NAMES[phase]}</h4>
                  <p className="phase-description">{PHASE_DESCRIPTIONS[phase]}</p>
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

              {/* Показываем панель управления воздушными атаками для фаз Movement и Air Attack */}
              {isCurrent && (phase === 'movement' || phase === 'air_attack') && authToken && (
                <AirAttackPanel
                  gameId={gameId}
                  authToken={authToken}
                  currentPhase={phase}
                  onRefresh={onRefresh}
                />
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
