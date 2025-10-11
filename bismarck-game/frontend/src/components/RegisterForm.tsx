// Компонент формы регистрации

import React, { useState } from 'react';
import { useGameStore } from '../stores/gameStore';
import { authAPI } from '../services/api/gameAPI';
import { RegisterRequest, ViewType, NotificationType } from '../types/gameTypes';
import './RegisterForm.css';

const RegisterForm: React.FC = () => {
  const [formData, setFormData] = useState<RegisterRequest>({
    username: '',
    email: '',
    password: '',
  });
  const [confirmPassword, setConfirmPassword] = useState('');
  const [errors, setErrors] = useState<Partial<RegisterRequest & { confirmPassword: string }>>({});
  const [isLoading, setIsLoading] = useState(false);

  const { setLoading, setError, addNotification } = useGameStore();

  // Обработка изменений в форме
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    
    if (name === 'confirmPassword') {
      setConfirmPassword(value);
    } else {
      setFormData(prev => ({
        ...prev,
        [name]: value,
      }));
    }
    
    // Очищаем ошибку при изменении поля
    if (errors[name as keyof typeof errors]) {
      setErrors(prev => ({
        ...prev,
        [name]: undefined,
      }));
    }
  };

  // Валидация формы
  const validateForm = (): boolean => {
    const newErrors: Partial<RegisterRequest & { confirmPassword: string }> = {};

    // Валидация имени пользователя
    if (!formData.username.trim()) {
      newErrors.username = 'Имя пользователя обязательно';
    } else if (formData.username.length < 3) {
      newErrors.username = 'Имя пользователя должно содержать минимум 3 символа';
    } else if (!/^[a-zA-Z0-9_]+$/.test(formData.username)) {
      newErrors.username = 'Имя пользователя может содержать только буквы, цифры и подчеркивания';
    }

    // Валидация email
    if (!formData.email.trim()) {
      newErrors.email = 'Email обязателен';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Введите корректный email адрес';
    }

    // Валидация пароля
    if (!formData.password) {
      newErrors.password = 'Пароль обязателен';
    } else if (formData.password.length < 6) {
      newErrors.password = 'Пароль должен содержать минимум 6 символов';
    }

    // Валидация подтверждения пароля
    if (!confirmPassword) {
      newErrors.confirmPassword = 'Подтверждение пароля обязательно';
    } else if (formData.password !== confirmPassword) {
      newErrors.confirmPassword = 'Пароли не совпадают';
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
      const response = await authAPI.register(formData);
      
      if (response.success && response.data) {
        addNotification({
          type: NotificationType.Success,
          title: 'Регистрация успешна',
          message: `Аккаунт ${formData.username} успешно создан! Теперь вы можете войти в игру.`,
          read: false,
        });
        
        // Переходим к форме логина
        useGameStore.getState().setCurrentView(ViewType.Login);
      } else {
        setError(response.error || 'Ошибка регистрации');
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка регистрации',
          message: response.error || 'Не удалось создать аккаунт',
          read: false,
        });
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.error || 'Ошибка соединения с сервером';
      setError(errorMessage);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка регистрации',
        message: errorMessage,
        read: false,
      });
    } finally {
      setIsLoading(false);
      setLoading(false);
    }
  };

  // Переход к логину
  const handleLoginClick = () => {
    useGameStore.getState().setCurrentView(ViewType.Login);
  };

  return (
    <div className="register-form-container">
      <div className="register-form">
        <h2>Регистрация</h2>
        <p className="subtitle">Создайте аккаунт для игры</p>
        
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
            <label htmlFor="email">Email</label>
            <input
              type="email"
              id="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              className={errors.email ? 'error' : ''}
              placeholder="Введите email"
              disabled={isLoading}
            />
            {errors.email && (
              <span className="error-message">{errors.email}</span>
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

          <div className="form-group">
            <label htmlFor="confirmPassword">Подтверждение пароля</label>
            <input
              type="password"
              id="confirmPassword"
              name="confirmPassword"
              value={confirmPassword}
              onChange={handleChange}
              className={errors.confirmPassword ? 'error' : ''}
              placeholder="Подтвердите пароль"
              disabled={isLoading}
            />
            {errors.confirmPassword && (
              <span className="error-message">{errors.confirmPassword}</span>
            )}
          </div>

          <button
            type="submit"
            className="submit-button"
            disabled={isLoading}
          >
            {isLoading ? 'Регистрация...' : 'Зарегистрироваться'}
          </button>
        </form>

        <div className="form-footer">
          <p>
            Уже есть аккаунт?{' '}
            <button
              type="button"
              className="link-button"
              onClick={handleLoginClick}
              disabled={isLoading}
            >
              Войти
            </button>
          </p>
        </div>
      </div>
    </div>
  );
};

export default RegisterForm;
