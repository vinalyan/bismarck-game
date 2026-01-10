import React, { useState, useEffect } from 'react';
import { GameTurn, GamePhase, PHASE_NAMES, PHASE_DESCRIPTIONS } from '../types/phaseTypes';
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasPendingAttacks, setHasPendingAttacks] = useState(false);

  // Сбрасываем состояние невыполненных атак при смене фазы
  useEffect(() => {
    if (currentTurn?.current_phase !== 'air_attack') {
      setHasPendingAttacks(false);
    }
  }, [currentTurn?.current_phase]);


  const handleNextPhase = async () => {
    try {
      setLoading(true);
      await phaseAPI.nextPhase({ game_id: gameId });
      
      // Удален вызов phaseAPI.getCurrentPhase - информация о текущей фазе теперь приходит через GameModel
      // Родительский компонент должен обновить currentTurn из GameModel
      
      // Уведомляем родительский компонент об обновлении хода
      // Информация о текущей фазе будет обновлена через GameModel
      window.dispatchEvent(new CustomEvent('turnUpdated'));
      
      // Обновляем данные игры через родительский компонент
      if (onRefresh) {
        onRefresh();
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
      
      // Уведомляем родительский компонент об обновлении хода
      // Родительский компонент должен обновить currentTurn из GameModel
      window.dispatchEvent(new CustomEvent('turnUpdated'));
      
      // Обновляем данные игры через родительский компонент
      if (onRefresh) {
        onRefresh();
      }
    } catch (err) {
      setError('Ошибка начала хода');
      console.error('Error starting turn:', err);
    } finally {
      setLoading(false);
    }
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
  const shouldShowStartTurnButton = (currentTurn?.turn_number ?? currentGame?.current_turn ?? 0) === 0 && 
                                     (currentTurn?.current_phase ?? currentGame?.current_phase ?? 'setup') === 'setup';

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
        <div className="phase-panel-header">
          <h3>Фазы игры</h3>
        </div>
        <p>Ожидание начала хода...</p>
      </div>
    );
  }

  const currentPhase = currentTurn.current_phase;
  const isMovementOrAirAttack = currentPhase === 'movement' || currentPhase === 'air_attack';

  return (
    <div className="phase-panel">
      <div className="phase-panel-header">
        <h3>Фазы игры</h3>
        <div className="turn-info">
          <span className="turn-number">Ход {currentTurn.turn_number}</span>
          <span className="current-phase">
            {PHASE_NAMES[currentPhase] || currentPhase}
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

      {/* Показываем описание текущей фазы */}
      {PHASE_DESCRIPTIONS[currentPhase] && (
        <div className="current-phase-description">
          <p>{PHASE_DESCRIPTIONS[currentPhase]}</p>
        </div>
      )}

      {/* Показываем панель управления воздушными атаками для фаз Movement и Air Attack */}
      {isMovementOrAirAttack && authToken && (
        <AirAttackPanel
          gameId={gameId}
          authToken={authToken}
          currentPhase={currentPhase}
          onRefresh={onRefresh}
          onHasPendingAttacks={(hasPending) => {
            // Обновляем состояние только для фазы air_attack
            if (currentPhase === 'air_attack') {
              setHasPendingAttacks(hasPending);
            } else {
              setHasPendingAttacks(false);
            }
          }}
        />
      )}

      <div className="phase-panel-footer">
        <button
          className="next-phase-button"
          onClick={handleNextPhase}
          disabled={loading || (currentPhase === 'air_attack' && hasPendingAttacks)}
          title={currentPhase === 'air_attack' && hasPendingAttacks ? 'Необходимо выполнить все атаки перед переходом к следующей фазе' : ''}
        >
          Следующая фаза
        </button>
      </div>
    </div>
  );
};

export default PhasePanel;
