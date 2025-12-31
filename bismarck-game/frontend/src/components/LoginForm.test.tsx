import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from './LoginForm';
import { useGameStore } from '../stores/gameStore';
import { authAPI } from '../services/api/gameAPI';
import { ViewType, NotificationType } from '../types/gameTypes';

// Мокируем store
jest.mock('../stores/gameStore');
const mockUseGameStore = useGameStore as jest.MockedFunction<typeof useGameStore>;

// Мокируем API
jest.mock('../services/api/gameAPI', () => ({
  authAPI: {
    login: jest.fn(),
  },
}));
const mockAuthAPI = authAPI as jest.Mocked<typeof authAPI>;

describe('LoginForm', () => {
  const mockLogin = jest.fn();
  const mockSetLoading = jest.fn();
  const mockSetError = jest.fn();
  const mockAddNotification = jest.fn();
  const mockSetCurrentView = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Настройка мока store
    mockUseGameStore.mockReturnValue({
      login: mockLogin,
      setLoading: mockSetLoading,
      setError: mockSetError,
      addNotification: mockAddNotification,
    } as any);

    // Настройка getState для setCurrentView
    (useGameStore.getState as jest.Mock) = jest.fn(() => ({
      setCurrentView: mockSetCurrentView,
    }));
  });

  describe('Rendering', () => {
    it('should render login form', () => {
      render(<LoginForm />);
      
      expect(screen.getByText(/вход в игру/i)).toBeInTheDocument();
      expect(screen.getByText(/погоня за бисмарком/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/имя пользователя/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/пароль/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /войти/i })).toBeInTheDocument();
    });

    it('should render link to register form', () => {
      render(<LoginForm />);
      
      expect(screen.getByText(/нет аккаунта/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /зарегистрироваться/i })).toBeInTheDocument();
    });
  });

  describe('Form fields', () => {
    it('should have username input', () => {
      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      expect(usernameInput).toHaveAttribute('type', 'text');
      expect(usernameInput).toHaveAttribute('name', 'username');
    });

    it('should have password input', () => {
      render(<LoginForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i);
      expect(passwordInput).toHaveAttribute('type', 'password');
      expect(passwordInput).toHaveAttribute('name', 'password');
    });
  });

  describe('Input handling', () => {
    it('should update username field when user types', () => {
      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i) as HTMLInputElement;
      userEvent.type(usernameInput, 'testuser');
      
      expect(usernameInput.value).toBe('testuser');
    });

    it('should update password field when user types', () => {
      render(<LoginForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i) as HTMLInputElement;
      userEvent.type(passwordInput, 'password123');
      
      expect(passwordInput.value).toBe('password123');
    });
  });

  describe('Validation', () => {
    it('should show error when username is empty', async () => {
      render(<LoginForm />);
      
      const submitButton = screen.getByRole('button', { name: /войти/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/имя пользователя обязательно/i)).toBeInTheDocument();
      });
    });

    it('should show error when password is empty', async () => {
      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      userEvent.type(usernameInput, 'testuser');
      
      const submitButton = screen.getByRole('button', { name: /войти/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/пароль обязателен/i)).toBeInTheDocument();
      });
    });

    it('should clear error when user starts typing', async () => {
      render(<LoginForm />);
      
      const submitButton = screen.getByRole('button', { name: /войти/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/имя пользователя обязательно/i)).toBeInTheDocument();
      });
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      userEvent.type(usernameInput, 't');
      
      await waitFor(() => {
        expect(screen.queryByText(/имя пользователя обязательно/i)).not.toBeInTheDocument();
      });
    });

    it('should apply error class to input when there is an error', async () => {
      render(<LoginForm />);
      
      const submitButton = screen.getByRole('button', { name: /войти/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        const usernameInput = screen.getByLabelText(/имя пользователя/i);
        expect(usernameInput).toHaveClass('error');
      });
    });
  });

  describe('Form submission - success', () => {
    it('should call login API on form submit', async () => {
      const mockUser = {
        id: '1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'player' as any,
        stats: {
          gamesPlayed: 0,
          gamesWon: 0,
          gamesLost: 0,
          totalScore: 0,
          averageScore: 0,
          winRate: 0,
          favoriteFaction: '',
          totalPlayTime: 0,
        },
        isActive: true,
        createdAt: '2023-01-01',
        updatedAt: '2023-01-01',
      };

      mockAuthAPI.login.mockResolvedValue({
        success: true,
        data: {
          user: mockUser,
          token: 'test-token',
        },
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockAuthAPI.login).toHaveBeenCalledWith({
          username: 'testuser',
          password: 'password123',
        });
      });
    });

    it('should call login action from store on successful login', async () => {
      const mockUser = {
        id: '1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'player' as any,
        stats: {
          gamesPlayed: 0,
          gamesWon: 0,
          gamesLost: 0,
          totalScore: 0,
          averageScore: 0,
          winRate: 0,
          favoriteFaction: '',
          totalPlayTime: 0,
        },
        isActive: true,
        createdAt: '2023-01-01',
        updatedAt: '2023-01-01',
      };

      mockAuthAPI.login.mockResolvedValue({
        success: true,
        data: {
          user: mockUser,
          token: 'test-token',
        },
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith(mockUser, 'test-token');
      });
    });

    it('should add success notification on successful login', async () => {
      const mockUser = {
        id: '1',
        username: 'testuser',
        email: 'test@example.com',
        role: 'player' as any,
        stats: {
          gamesPlayed: 0,
          gamesWon: 0,
          gamesLost: 0,
          totalScore: 0,
          averageScore: 0,
          winRate: 0,
          favoriteFaction: '',
          totalPlayTime: 0,
        },
        isActive: true,
        createdAt: '2023-01-01',
        updatedAt: '2023-01-01',
      };

      mockAuthAPI.login.mockResolvedValue({
        success: true,
        data: {
          user: mockUser,
          token: 'test-token',
        },
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Success,
          title: 'Успешный вход',
          message: 'Добро пожаловать, testuser!',
          read: false,
        });
      });
    });
  });

  describe('Form submission - error', () => {
    it('should handle API error response', async () => {
      mockAuthAPI.login.mockResolvedValue({
        success: false,
        error: 'Неверные учетные данные',
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'wrongpassword');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Неверные учетные данные');
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Error,
          title: 'Ошибка входа',
          message: 'Неверные учетные данные',
          read: false,
        });
      });
    });

    it('should handle network error', async () => {
      const networkError = {
        response: {
          data: {
            error: 'Network error',
          },
        },
      };
      mockAuthAPI.login.mockRejectedValue(networkError);

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Network error');
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Error,
          title: 'Ошибка входа',
          message: 'Network error',
          read: false,
        });
      });
    });

    it('should handle error without response data', async () => {
      mockAuthAPI.login.mockRejectedValue(new Error('Connection failed'));

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Ошибка соединения с сервером');
      });
    });
  });

  describe('Loading state', () => {
    it('should show loading text when submitting', async () => {
      mockAuthAPI.login.mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve({ success: true, data: { user: {} as any, token: 'token' } }), 100))
      );

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/вход\.\.\./i)).toBeInTheDocument();
      });
    });

    it('should disable inputs when loading', async () => {
      mockAuthAPI.login.mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve({ success: true, data: { user: {} as any, token: 'token' } }), 100))
      );

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(usernameInput).toBeDisabled();
        expect(passwordInput).toBeDisabled();
        expect(submitButton).toBeDisabled();
      });
    });

    it('should call setLoading on submit', async () => {
      mockAuthAPI.login.mockResolvedValue({
        success: true,
        data: {
          user: {} as any,
          token: 'test-token',
        },
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetLoading).toHaveBeenCalledWith(true);
        expect(mockSetLoading).toHaveBeenCalledWith(false);
      });
    });
  });

  describe('Navigation', () => {
    it('should navigate to register form when register link is clicked', () => {
      render(<LoginForm />);
      
      const registerLink = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(registerLink);
      
      expect(mockSetCurrentView).toHaveBeenCalledWith(ViewType.Register);
    });

    it('should clear error before submitting valid form', async () => {
      mockAuthAPI.login.mockResolvedValue({
        success: true,
        data: {
          user: {} as any,
          token: 'test-token',
        },
      });

      render(<LoginForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const submitButton = screen.getByRole('button', { name: /войти/i });
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(passwordInput, 'password123');
      userEvent.click(submitButton);
      
      await waitFor(() => {
        // setError(null) должен быть вызван в начале handleSubmit перед API запросом
        expect(mockSetError).toHaveBeenCalledWith(null);
      });
    });
  });
});

