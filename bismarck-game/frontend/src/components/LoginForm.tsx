// Компонент формы логина

import React, { useState } from 'react';
import { useGameStore } from '../stores/gameStore';
import { authAPI } from '../services/api/gameAPI';
import { LoginRequest, ViewType, NotificationType } from '../types/gameTypes';
import './LoginForm.css';

const LoginForm: React.FC = () => {
  const [formData, setFormData] = useState<LoginRequest>({
    username: '',
    password: '',
  });
  const [errors, setErrors] = useState<Partial<LoginRequest>>({});
  const [isLoading, setIsLoading] = useState(false);

  const { login, setLoading, setError, addNotification } = useGameStore();

  // Обработка изменений в форме
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value,
    }));
    
    // Очищаем ошибку при изменении поля
    if (errors[name as keyof LoginRequest]) {
      setErrors(prev => ({
        ...prev,
        [name]: undefined,
      }));
    }
  };

  // Валидация формы
  const validateForm = (): boolean => {
    const newErrors: Partial<LoginRequest> = {};

    if (!formData.username.trim()) {
      newErrors.username = 'Имя пользователя обязательно';
    }

    if (!formData.password) {
      newErrors.password = 'Пароль обязателен';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Обработка отправки формы
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    setIsLoading(true);
    setLoading(true);
    setError(null);

    try {
      const response = await authAPI.login(formData);
      
      if (response.success && response.data) {
        const { user, token } = response.data;
        login(user, token);
        
        addNotification({
          type: NotificationType.Success,
          title: 'Успешный вход',
          message: `Добро пожаловать, ${user.username}!`,
          read: false,
        });
      } else {
        setError(response.error || 'Ошибка входа');
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка входа',
          message: response.error || 'Неверные учетные данные',
          read: false,
        });
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || 'Ошибка соединения с сервером';
      setError(errorMessage);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка входа',
        message: errorMessage,
        read: false,
      });
    } finally {
      setIsLoading(false);
      setLoading(false);
    }
  };

  // Переход к регистрации
  const handleRegisterClick = () => {
    useGameStore.getState().setCurrentView(ViewType.Register);
  };

  return (
    <div className="login-form-container">
      <div className="login-form">
        <h2>Вход в игру</h2>
        <p className="subtitle">Погоня за Бисмарком</p>
        
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Имя пользователя</label>
            <input
              type="text"
              id="username"
              name="username"
              value={formData.username}
              onChange={handleChange}
              className={errors.username ? 'error' : ''}
              placeholder="Введите имя пользователя"
              disabled={isLoading}
            />
            {errors.username && (
              <span className="error-message">{errors.username}</span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              type="password"
              id="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              className={errors.password ? 'error' : ''}
              placeholder="Введите пароль"
              disabled={isLoading}
            />
            {errors.password && (
              <span className="error-message">{errors.password}</span>
            )}
          </div>

          <button
            type="submit"
            className="submit-button"
            disabled={isLoading}
          >
            {isLoading ? 'Вход...' : 'Войти'}
          </button>
        </form>

        <div className="form-footer">
          <p>
            Нет аккаунта?{' '}
            <button
              type="button"
              className="link-button"
              onClick={handleRegisterClick}
              disabled={isLoading}
            >
              Зарегистрироваться
            </button>
          </p>
        </div>
      </div>
    </div>
  );
};

export default LoginForm;
