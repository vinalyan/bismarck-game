import React, { useEffect } from 'react';
import { useGameStore } from './stores/gameStore';
import { ViewType } from './types/gameTypes';
import LoginForm from './components/LoginForm';
import RegisterForm from './components/RegisterForm';
import Lobby from './components/Lobby';
import Game from './components/Game';
import wsClient from './services/websocket/websocketClient';
import './App.css';

function App() {
  const { isAuthenticated, ui, setCurrentView, setUser, setAuthToken, setAuthenticated } = useGameStore();

  // Инициализация приложения
  useEffect(() => {
    // Проверяем сохраненные данные пользователя
    const savedToken = localStorage.getItem('authToken');
    const savedUser = localStorage.getItem('user');

    if (savedToken && savedUser) {
      try {
        const user = JSON.parse(savedUser);
        setUser(user);
        setAuthToken(savedToken);
        setAuthenticated(true);
        setCurrentView(ViewType.Lobby);
        
        // Подключаемся к WebSocket
        wsClient.connect(savedToken).catch((error) => {
          console.error('Failed to connect to WebSocket:', error);
        });
      } catch (error) {
        console.error('Error parsing saved user data:', error);
        localStorage.removeItem('authToken');
        localStorage.removeItem('user');
      }
    }
  }, [setUser, setAuthToken, setAuthenticated, setCurrentView]);

  // Очистка WebSocket при размонтировании
  useEffect(() => {
    return () => {
      wsClient.disconnect();
    };
  }, []);

  // Рендеринг компонентов в зависимости от состояния
  const renderCurrentView = () => {
    if (!isAuthenticated) {
      switch (ui.currentView) {
        case ViewType.Register:
          return <RegisterForm />;
        case ViewType.Login:
        default:
          return <LoginForm />;
      }
    }

    switch (ui.currentView) {
      case ViewType.Lobby:
        return <Lobby />;
      case ViewType.Game:
        return <Game />;
      case ViewType.Profile:
        return <div>Профиль (в разработке)</div>;
      default:
        return <Lobby />;
    }
  };

  return (
    <div className="App">
      {renderCurrentView()}
    </div>
  );
}

export default App;
