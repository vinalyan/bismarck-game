import React, { useState, useEffect, useRef } from 'react';
import { gameEventAPI, GameEvent } from '../services/api/gameEventAPI';
import { useGameStore } from '../stores/gameStore';
import './GameLog.css';

interface GameLogProps {
  gameId: string;
}

const GameLog: React.FC<GameLogProps> = ({ gameId }) => {
  const { currentGame, user } = useGameStore();
  const [events, setEvents] = useState<GameEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const logContentRef = useRef<HTMLDivElement>(null);

  // Определяем сторону игрока
  const getPlayerSide = (): string => {
    if (!currentGame || !user) return 'unknown';
    
    if (currentGame.player1_id === user.id) {
      return 'german'; // Player1 всегда немцы
    }
    if (currentGame.player2_id === user.id) {
      return 'allied'; // Player2 всегда союзники
    }
    
    return 'unknown';
  };

  useEffect(() => {
    loadEvents();
    
    // Обновляем события каждые 5 секунд
    const interval = setInterval(loadEvents, 5000);
    
    // Слушаем WebSocket события для мгновенного обновления
    const handleGameEvent = () => {
      loadEvents();
    };
    
    window.addEventListener('gameEventReceived', handleGameEvent);
    
    return () => {
      clearInterval(interval);
      window.removeEventListener('gameEventReceived', handleGameEvent);
    };
  }, [gameId]);

  const loadEvents = async () => {
    setLoading(true);
    setError(null);
    
    const playerSide = getPlayerSide();
    if (playerSide === 'unknown') {
      setError('Unable to determine player side');
      setLoading(false);
      return;
    }
    
    try {
      const response = await gameEventAPI.getGameEvents(gameId, playerSide, 15);
      
      if (response.success && response.data && response.data.events) {
        // События уже приходят в правильном порядке (новые сверху)
        setEvents(response.data.events);
        // Автопрокрутка к новым событиям (они уже сверху)
        setTimeout(() => {
          if (logContentRef.current) {
            logContentRef.current.scrollTop = 0;
          }
        }, 100);
      } else {
        setError(response.error || 'Failed to load events');
        setEvents([]); // Убеждаемся, что events всегда массив
      }
    } catch (error) {
      setError('Failed to load events');
      setEvents([]); // Убеждаемся, что events всегда массив
    } finally {
      setLoading(false);
    }
  };

  const getEventIcon = (eventType: string): string => {
    switch (eventType) {
      case 'movement': return '🚢';
      case 'phase_change': return '🔄';
      case 'turn_change': return '⏰';
      default: return '📝';
    }
  };

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('ru-RU', { 
      hour: '2-digit', 
      minute: '2-digit',
      second: '2-digit'
    });
  };

  const formatEventDescription = (event: GameEvent): string => {
    if (event.event_type === 'movement' && event.data) {
      const { from_hex, to_hex, fuel_cost, hexes_moved } = event.data;
      return `${event.actor_name} переместился из ${from_hex} в ${to_hex} (${hexes_moved} гексов, ${fuel_cost} топлива)`;
    }
    return event.description;
  };

  return (
    <div className="game-log">
      <div className="game-log-header">
        <h3>Лог игры</h3>
        <button 
          onClick={loadEvents} 
          className="refresh-btn"
          disabled={loading}
        >
          {loading ? '⏳' : '🔄'}
        </button>
      </div>

      <div className="game-log-content" ref={logContentRef}>
        {error && (
          <div className="error-message">
            {error}
          </div>
        )}

        {loading && events.length === 0 ? (
          <div className="loading">Загрузка событий...</div>
        ) : (
          <div className="events-list">
            {events && events.length > 0 ? events.map((event) => (
              <div key={event.id} className={`log-event event-${event.event_type}`}>
                <div className="event-header">
                  <span className="event-icon">
                    {getEventIcon(event.event_type)}
                  </span>
                  <span className="event-time">
                    {formatTimestamp(event.created_at)}
                  </span>
                  <span className="event-turn">
                    Ход {event.turn}, {event.phase}
                  </span>
                </div>
                <div className="event-description">
                  {formatEventDescription(event)}
                </div>
              </div>
            )) : (
              <div className="no-events">
                Пока нет событий в игре
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default GameLog;
