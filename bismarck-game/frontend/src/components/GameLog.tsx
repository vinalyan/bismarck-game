import React, { useState, useEffect, useRef } from 'react';
import { useGameStore } from '../stores/gameStore';
import { unitsAPI } from '../services/api/unitsAPI';
import './GameLog.css';

// Интерфейс для события игры (перенесен из gameEventAPI)
interface GameEvent {
  id: string;
  game_id: string;
  turn: number;
  phase: string;
  event_type: string;
  actor_name: string;
  description: string;
  data: any;
  created_at: string;
}

interface GameLogProps {
  gameId: string;
}

const GameLog: React.FC<GameLogProps> = ({ gameId }) => {
  const { currentGame, user, authToken } = useGameStore();
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
    
    // Отключен автоматический polling - используем WebSocket для обновлений
    // const interval = setInterval(loadEvents, 5000);
    
    // Слушаем WebSocket события для мгновенного обновления
    const handleGameEvent = async () => {
      // Удалены вызовы gameEventAPI.getGameEvents и phaseAPI.getCurrentPhase
      // События и текущая фаза теперь приходят через GameModel
      loadEvents();
    };
    
    // Слушаем события manual refresh для обновления лога
    const handleGameLogRefresh = () => {
      console.log('📢 GameLog received refresh event, reloading events...');
      loadEvents();
    };
    
    window.addEventListener('gameEventReceived', handleGameEvent);
    window.addEventListener('gameLogRefresh', handleGameLogRefresh);
    
    return () => {
      // clearInterval(interval);
      window.removeEventListener('gameEventReceived', handleGameEvent);
      window.removeEventListener('gameLogRefresh', handleGameLogRefresh);
    };
  }, [gameId]);

  const loadEvents = async () => {
    if (!gameId || !authToken) {
      console.warn('Missing gameId or authToken for loading events');
      return;
    }

    setLoading(true);
    setError(null);
    
    try {
      // Получаем события из GameModel через unitsAPI
      const response = await unitsAPI.getGameUnits(gameId, authToken);
      
      if (response.success && response.data.events) {
        // Преобразуем события из GameModel в формат GameEvent
        const gameEvents: GameEvent[] = response.data.events.map((event: any) => ({
          id: event.id || event.ID,
          game_id: event.game_id || event.GameID || gameId,
          turn: event.turn || event.Turn || 0,
          phase: event.phase || event.Phase || '',
          event_type: event.event_type || event.EventType || '',
          actor_name: event.actor_name || event.ActorName || '',
          description: event.description || event.Description || '',
          data: event.data || event.Data || {},
          created_at: event.created_at || event.CreatedAt || new Date().toISOString(),
        }));
        
        // Сортируем по дате создания (новые сверху)
        gameEvents.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
        
        setEvents(gameEvents);
      } else {
        setEvents([]);
      }
    } catch (err: any) {
      console.error('Error loading game events:', err);
      setError('Не удалось загрузить события игры');
      setEvents([]);
    } finally {
      setLoading(false);
    }
  };

  const getEventIcon = (eventType: string): string => {
    switch (eventType) {
      case 'movement': return '🚢';
      case 'phase_change': return '🔄';
      case 'turn_change': return '⏰';
      case 'air_attack': return '✈️💥';
      case 'combat': return '⚔️';
      case 'damage': return '💥';
      case 'sinking': return '🌊';
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
    
    if (event.event_type === 'air_attack' && event.data) {
      const { hex_id, action, target_name, target_class, hull_damage, new_hull, sunk } = event.data;
      
      // Обработка добавления/удаления маркера воздушной атаки
      if (action === 'added') {
        return `✈️ Маркер воздушной атаки добавлен в гекс ${hex_id}`;
      } else if (action === 'removed') {
        return `✈️ Маркер воздушной атаки удален из гекса ${hex_id}`;
      }
      
      // Обработка выполненной воздушной атаки
      if (target_name || target_class || sunk !== undefined) {
        // Формируем название корабля (имя и класс)
        let shipName = '';
        if (target_name && target_class) {
          shipName = `${target_name} (${target_class})`;
        } else if (target_name) {
          shipName = target_name;
        } else if (target_class) {
          shipName = target_class;
        } else {
          shipName = 'неизвестный корабль';
        }
        
        // Формируем информацию об атакующей стороне
        const attackerInfo = event.actor_name ? `[${event.actor_name}] ` : '';
        
        if (sunk) {
          return `✈️💥 ${attackerInfo}Воздушная атака: корабль ${shipName} потоплен в гексе ${hex_id}`;
        }
        
        const damageText = hull_damage === 1 ? '1 повреждение' : `${hull_damage || 1} повреждений`;
        return `✈️💥 ${attackerInfo}Воздушная атака на корабль ${shipName} в гексе ${hex_id}: нанесено ${damageText}, HULL: ${new_hull || 0}`;
      }
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
