// Компонент лобби для списка игр

import React, { useState, useEffect } from 'react';
import { useGameStore } from '../stores/gameStore';
import { gameAPI } from '../services/api/gameAPI';
import { CreateGameRequest, GameResponse, GameStatus, ViewType, GameMode, Difficulty, VictoryCondition, NotificationType } from '../types/gameTypes';
import './Lobby.css';

const Lobby: React.FC = () => {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createFormData, setCreateFormData] = useState<CreateGameRequest>({
    name: '',
    settings: {
      maxPlayers: 2,
      turnDuration: 30,
      gameMode: GameMode.Classic,
      difficulty: Difficulty.Normal,
      weatherEnabled: true,
      fogOfWar: true,
      randomEvents: true,
      victoryConditions: [VictoryCondition.Operational],
    },
  });
  const [isCreating, setIsCreating] = useState(false);

  const {
    user,
    games,
    setGames,
    addGame,
    updateGame,
    setLoading,
    setError,
    addNotification,
    joinGame,
    logout,
    setCurrentView,
  } = useGameStore();

  // Загрузка списка игр при монтировании компонента
  useEffect(() => {
    loadGames();
  }, []);

  // Загрузка списка игр
  const loadGames = async () => {
    setLoading(true);
    try {
      const response = await gameAPI.getGames();
      if (response.success && response.data) {
        setGames(response.data);
      } else {
        setError(response.error || 'Ошибка загрузки игр');
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || 'Ошибка соединения с сервером';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // Создание новой игры
  const handleCreateGame = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!createFormData.name.trim()) {
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Введите название игры',
        read: false,
      });
      return;
    }

    setIsCreating(true);
    setLoading(true);

    try {
      const response = await gameAPI.createGame(createFormData);
      if (response.success && response.data) {
        addNotification({
          type: NotificationType.Success,
          title: 'Игра создана',
          message: `Игра "${createFormData.name}" успешно создана`,
          read: false,
        });
        
        // Добавляем игру в список
        const gameResponse: GameResponse = {
          ...response.data,
          player1: user!,
        };
        addGame(gameResponse);
        
        // Сбрасываем форму
        setCreateFormData({
          name: '',
          settings: {
            maxPlayers: 2,
            turnDuration: 30,
            gameMode: GameMode.Classic,
            difficulty: Difficulty.Normal,
            weatherEnabled: true,
            fogOfWar: true,
            randomEvents: true,
            victoryConditions: [VictoryCondition.Operational],
          },
        });
        setShowCreateForm(false);
      } else {
        setError(response.error || 'Ошибка создания игры');
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || 'Ошибка соединения с сервером';
      setError(errorMessage);
    } finally {
      setIsCreating(false);
      setLoading(false);
    }
  };

  // Присоединение к игре
  const handleJoinGame = async (gameId: string) => {
    // Проверяем, что пользователь не пытается присоединиться к своей игре
    const game = games.find(g => g.id === gameId);
    if (game && user && (game.player1?.id === user.id || game.player1_id === user.id)) {
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Вы не можете присоединиться к своей собственной игре',
        read: false,
      });
      return;
    }

    setLoading(true);
    try {
      const response = await gameAPI.joinGame({ gameId });
      if (response.success && response.data) {
        addNotification({
          type: NotificationType.Success,
          title: 'Присоединение к игре',
          message: 'Вы успешно присоединились к игре',
          read: false,
        });
        
        // Обновляем игру в списке данными из ответа API
        updateGame(gameId, response.data);
        
        // Переходим к игре (пока просто обновляем список игр)
        loadGames();
      } else {
        const errorMessage = response.error || 'Ошибка присоединения к игре';
        setError(errorMessage);
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка присоединения',
          message: errorMessage,
          read: false,
        });
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || 'Ошибка соединения с сервером';
      setError(errorMessage);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка соединения',
        message: errorMessage,
        read: false,
      });
    } finally {
      setLoading(false);
    }
  };

  // Выход из аккаунта
  const handleLogout = () => {
    logout();
    addNotification({
      type: NotificationType.Info,
      title: 'Выход',
      message: 'Вы вышли из аккаунта',
      read: false,
    });
  };

  // Получение статуса игры для отображения
  const getGameStatusText = (status: GameStatus): string => {
    switch (status) {
      case GameStatus.Waiting:
        return 'Ожидание игроков';
      case GameStatus.InProgress:
        return 'В процессе';
      case GameStatus.Completed:
        return 'Завершена';
      case GameStatus.Cancelled:
        return 'Отменена';
      default:
        return 'Неизвестно';
    }
  };

  // Получение класса статуса для стилизации
  const getGameStatusClass = (status: GameStatus): string => {
    switch (status) {
      case GameStatus.Waiting:
        return 'status-waiting';
      case GameStatus.InProgress:
        return 'status-in-progress';
      case GameStatus.Completed:
        return 'status-completed';
      case GameStatus.Cancelled:
        return 'status-cancelled';
      default:
        return 'status-unknown';
    }
  };

  return (
    <div className="lobby-container">
      <div className="lobby-header">
        <h1>Лобби</h1>
        <div className="user-info">
          <span>Добро пожаловать, {user?.username}!</span>
          <button onClick={handleLogout} className="logout-button">
            Выйти
          </button>
        </div>
      </div>

      <div className="lobby-content">
        <div className="games-section">
          <div className="section-header">
            <h2>Доступные игры</h2>
            <button
              onClick={() => setShowCreateForm(!showCreateForm)}
              className="create-game-button"
            >
              {showCreateForm ? 'Отмена' : 'Создать игру'}
            </button>
          </div>

          {showCreateForm && (
            <div className="create-game-form">
              <form onSubmit={handleCreateGame}>
                <div className="form-group">
                  <label htmlFor="gameName">Название игры</label>
                  <input
                    type="text"
                    id="gameName"
                    value={createFormData.name}
                    onChange={(e) => setCreateFormData(prev => ({
                      ...prev,
                      name: e.target.value,
                    }))}
                    placeholder="Введите название игры"
                    disabled={isCreating}
                  />
                </div>

                <div className="form-group">
                  <label htmlFor="maxPlayers">Максимум игроков</label>
                  <select
                    id="maxPlayers"
                    value={createFormData.settings.maxPlayers}
                    onChange={(e) => setCreateFormData(prev => ({
                      ...prev,
                      settings: {
                        ...prev.settings,
                        maxPlayers: parseInt(e.target.value),
                      },
                    }))}
                    disabled={isCreating}
                  >
                    <option value={2}>2 игрока</option>
                    <option value={4}>4 игрока</option>
                  </select>
                </div>

                <div className="form-group">
                  <label htmlFor="turnDuration">Длительность хода (сек)</label>
                  <select
                    id="turnDuration"
                    value={createFormData.settings.turnDuration}
                    onChange={(e) => setCreateFormData(prev => ({
                      ...prev,
                      settings: {
                        ...prev.settings,
                        turnDuration: parseInt(e.target.value),
                      },
                    }))}
                    disabled={isCreating}
                  >
                    <option value={30}>30 секунд</option>
                    <option value={60}>1 минута</option>
                    <option value={120}>2 минуты</option>
                  </select>
                </div>

                <button
                  type="submit"
                  className="submit-button"
                  disabled={isCreating}
                >
                  {isCreating ? 'Создание...' : 'Создать игру'}
                </button>
              </form>
            </div>
          )}

          <div className="games-list">
            {games.length === 0 ? (
              <div className="no-games">
                <p>Нет доступных игр</p>
                <p>Создайте новую игру или подождите, пока кто-то создаст</p>
              </div>
            ) : (
              games.map((game) => (
                <div key={game.id} className="game-card">
                  <div className="game-info">
                    <h3>{game.name}</h3>
                    <p className="game-creator">Создатель: {game.player1_username || game.player1?.username || game.player1_id}</p>
                    <p className="game-players">
                      Игроки: {game.player1_username || game.player1?.username || game.player1_id}
                      {game.player2_id && `, ${game.player2_username || game.player2?.username || game.player2_id}`}
                    </p>
                    <p className="game-settings">
                      Режим: {game.settings?.gameMode || 'Классический'}, 
                      Сложность: {game.settings?.difficulty || 'Стандартная'}
                    </p>
                  </div>
                  
                  <div className="game-status">
                    <span className={`status-badge ${getGameStatusClass(game.status)}`}>
                      {getGameStatusText(game.status)}
                    </span>
                  </div>

                  <div className="game-actions">
                    {game.status === GameStatus.Waiting && !game.player2_id && (
                      <button
                        onClick={() => handleJoinGame(game.id)}
                        className="join-button"
                        disabled={game.player1?.id === user?.id || game.player1_id === user?.id}
                      >
                        {(game.player1?.id === user?.id || game.player1_id === user?.id) ? 'Ваша игра' : 'Присоединиться'}
                      </button>
                    )}
                    
                    {game.status === GameStatus.InProgress && (
                      <button
                        onClick={() => setCurrentView(ViewType.Game)}
                        className="view-game-button"
                      >
                        Продолжить игру
                      </button>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Lobby;
