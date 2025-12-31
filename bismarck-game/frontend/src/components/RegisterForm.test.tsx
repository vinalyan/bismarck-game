import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RegisterForm from './RegisterForm';
import { useGameStore } from '../stores/gameStore';
import { authAPI } from '../services/api/gameAPI';
import { ViewType, NotificationType } from '../types/gameTypes';

// Мокируем store
jest.mock('../stores/gameStore');
const mockUseGameStore = useGameStore as jest.MockedFunction<typeof useGameStore>;

// Мокируем API
jest.mock('../services/api/gameAPI', () => ({
  authAPI: {
    register: jest.fn(),
  },
}));
const mockAuthAPI = authAPI as jest.Mocked<typeof authAPI>;

describe('RegisterForm', () => {
  const mockSetLoading = jest.fn();
  const mockSetError = jest.fn();
  const mockAddNotification = jest.fn();
  const mockSetCurrentView = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Настройка мока store
    mockUseGameStore.mockReturnValue({
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
    it('should render register form', () => {
      render(<RegisterForm />);
      
      expect(screen.getByText(/регистрация/i)).toBeInTheDocument();
      expect(screen.getByText(/создайте аккаунт для игры/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/имя пользователя/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/пароль/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/подтверждение пароля/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /зарегистрироваться/i })).toBeInTheDocument();
    });

    it('should render link to login form', () => {
      render(<RegisterForm />);
      
      expect(screen.getByText(/уже есть аккаунт/i)).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /войти/i })).toBeInTheDocument();
    });
  });

  describe('Form fields', () => {
    it('should have all required input fields', () => {
      render(<RegisterForm />);
      
      expect(screen.getByLabelText(/имя пользователя/i)).toHaveAttribute('type', 'text');
      expect(screen.getByLabelText(/email/i)).toHaveAttribute('type', 'email');
      expect(screen.getByLabelText(/пароль/i)).toHaveAttribute('type', 'password');
      expect(screen.getByLabelText(/подтверждение пароля/i)).toHaveAttribute('type', 'password');
    });
  });

  describe('Input handling', () => {
    it('should update username field when user types', () => {
      render(<RegisterForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i) as HTMLInputElement;
      userEvent.type(usernameInput, 'testuser');
      
      expect(usernameInput.value).toBe('testuser');
    });

    it('should update email field when user types', () => {
      render(<RegisterForm />);
      
      const emailInput = screen.getByLabelText(/email/i) as HTMLInputElement;
      userEvent.type(emailInput, 'test@example.com');
      
      expect(emailInput.value).toBe('test@example.com');
    });

    it('should update password field when user types', () => {
      render(<RegisterForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i) as HTMLInputElement;
      userEvent.type(passwordInput, 'password123');
      
      expect(passwordInput.value).toBe('password123');
    });

    it('should update confirm password field when user types', () => {
      render(<RegisterForm />);
      
      const confirmPasswordInput = screen.getByLabelText(/подтверждение пароля/i) as HTMLInputElement;
      userEvent.type(confirmPasswordInput, 'password123');
      
      expect(confirmPasswordInput.value).toBe('password123');
    });
  });

  describe('Validation - username', () => {
    it('should show error when username is empty', async () => {
      render(<RegisterForm />);
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/имя пользователя обязательно/i)).toBeInTheDocument();
      });
    });

    it('should show error when username is too short', async () => {
      render(<RegisterForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      userEvent.type(usernameInput, 'ab');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/имя пользователя должно содержать минимум 3 символа/i)).toBeInTheDocument();
      });
    });

    it('should show error when username contains invalid characters', async () => {
      render(<RegisterForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      userEvent.type(usernameInput, 'test-user!');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/имя пользователя может содержать только буквы, цифры и подчеркивания/i)).toBeInTheDocument();
      });
    });

    it('should accept valid username', async () => {
      render(<RegisterForm />);
      
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      userEvent.type(usernameInput, 'test_user123');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      // Should not show username validation error
      await waitFor(() => {
        expect(screen.queryByText(/имя пользователя обязательно/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/имя пользователя должно содержать минимум 3 символа/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/имя пользователя может содержать только/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Validation - email', () => {
    it('should show error when email is empty', async () => {
      render(<RegisterForm />);
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/email обязателен/i)).toBeInTheDocument();
      });
    });

    it('should show error when email format is invalid', async () => {
      render(<RegisterForm />);
      
      const emailInput = screen.getByLabelText(/email/i);
      userEvent.type(emailInput, 'invalid-email');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/введите корректный email адрес/i)).toBeInTheDocument();
      });
    });

    it('should accept valid email', async () => {
      render(<RegisterForm />);
      
      const emailInput = screen.getByLabelText(/email/i);
      userEvent.type(emailInput, 'test@example.com');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      // Should not show email validation errors (but will show other validation errors like password)
      await waitFor(() => {
        expect(screen.queryByText(/email обязателен/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/введите корректный email адрес/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Validation - password', () => {
    it('should show error when password is empty', async () => {
      render(<RegisterForm />);
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/пароль обязателен/i)).toBeInTheDocument();
      });
    });

    it('should show error when password is too short', async () => {
      render(<RegisterForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i);
      userEvent.type(passwordInput, '12345');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/пароль должен содержать минимум 6 символов/i)).toBeInTheDocument();
      });
    });
  });

  describe('Validation - confirm password', () => {
    it('should show error when confirm password is empty', async () => {
      render(<RegisterForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i);
      userEvent.type(passwordInput, 'password123');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/подтверждение пароля обязательно/i)).toBeInTheDocument();
      });
    });

    it('should show error when passwords do not match', async () => {
      render(<RegisterForm />);
      
      const passwordInput = screen.getByLabelText(/пароль/i);
      const confirmPasswordInput = screen.getByLabelText(/подтверждение пароля/i);
      
      userEvent.type(passwordInput, 'password123');
      userEvent.type(confirmPasswordInput, 'different');
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/пароли не совпадают/i)).toBeInTheDocument();
      });
    });
  });

  describe('Form submission - success', () => {
    const fillValidForm = () => {
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const emailInput = screen.getByLabelText(/email/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const confirmPasswordInput = screen.getByLabelText(/подтверждение пароля/i);
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(emailInput, 'test@example.com');
      userEvent.type(passwordInput, 'password123');
      userEvent.type(confirmPasswordInput, 'password123');
    };

    it('should call register API on form submit', async () => {
      mockAuthAPI.register.mockResolvedValue({
        success: true,
        data: {} as any,
      });

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockAuthAPI.register).toHaveBeenCalledWith({
          username: 'testuser',
          email: 'test@example.com',
          password: 'password123',
        });
      });
    });

    it('should add success notification on successful registration', async () => {
      mockAuthAPI.register.mockResolvedValue({
        success: true,
        data: {} as any,
      });

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Success,
          title: 'Регистрация успешна',
          message: 'Аккаунт testuser успешно создан! Теперь вы можете войти в игру.',
          read: false,
        });
      });
    });

    it('should navigate to login form on successful registration', async () => {
      mockAuthAPI.register.mockResolvedValue({
        success: true,
        data: {} as any,
      });

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetCurrentView).toHaveBeenCalledWith(ViewType.Login);
      });
    });
  });

  describe('Form submission - error', () => {
    const fillValidForm = () => {
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const emailInput = screen.getByLabelText(/email/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const confirmPasswordInput = screen.getByLabelText(/подтверждение пароля/i);
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(emailInput, 'test@example.com');
      userEvent.type(passwordInput, 'password123');
      userEvent.type(confirmPasswordInput, 'password123');
    };

    it('should handle API error response', async () => {
      mockAuthAPI.register.mockResolvedValue({
        success: false,
        error: 'Username already exists',
      });

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Username already exists');
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Error,
          title: 'Ошибка регистрации',
          message: 'Username already exists',
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
      mockAuthAPI.register.mockRejectedValue(networkError);

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Network error');
        expect(mockAddNotification).toHaveBeenCalledWith({
          type: NotificationType.Error,
          title: 'Ошибка регистрации',
          message: 'Network error',
          read: false,
        });
      });
    });

    it('should handle error without response data', async () => {
      mockAuthAPI.register.mockRejectedValue(new Error('Connection failed'));

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetError).toHaveBeenCalledWith('Ошибка соединения с сервером');
      });
    });
  });

  describe('Loading state', () => {
    const fillValidForm = () => {
      const usernameInput = screen.getByLabelText(/имя пользователя/i);
      const emailInput = screen.getByLabelText(/email/i);
      const passwordInput = screen.getByLabelText(/пароль/i);
      const confirmPasswordInput = screen.getByLabelText(/подтверждение пароля/i);
      
      userEvent.type(usernameInput, 'testuser');
      userEvent.type(emailInput, 'test@example.com');
      userEvent.type(passwordInput, 'password123');
      userEvent.type(confirmPasswordInput, 'password123');
    };

    it('should show loading text when submitting', async () => {
      mockAuthAPI.register.mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve({ success: true, data: {} as any }), 100))
      );

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByText(/регистрация\.\.\./i)).toBeInTheDocument();
      });
    });

    it('should disable inputs when loading', async () => {
      mockAuthAPI.register.mockImplementation(
        () => new Promise(resolve => setTimeout(() => resolve({ success: true, data: {} as any }), 100))
      );

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(screen.getByLabelText(/имя пользователя/i)).toBeDisabled();
        expect(screen.getByLabelText(/email/i)).toBeDisabled();
        expect(screen.getByLabelText(/пароль/i)).toBeDisabled();
        expect(screen.getByLabelText(/подтверждение пароля/i)).toBeDisabled();
        expect(submitButton).toBeDisabled();
      });
    });

    it('should call setLoading on submit', async () => {
      mockAuthAPI.register.mockResolvedValue({
        success: true,
        data: {} as any,
      });

      render(<RegisterForm />);
      
      fillValidForm();
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
      userEvent.click(submitButton);
      
      await waitFor(() => {
        expect(mockSetLoading).toHaveBeenCalledWith(true);
        expect(mockSetLoading).toHaveBeenCalledWith(false);
      });
    });
  });

  describe('Navigation', () => {
    it('should navigate to login form when login link is clicked', () => {
      render(<RegisterForm />);
      
      const loginLink = screen.getByRole('button', { name: /войти/i });
      userEvent.click(loginLink);
      
      expect(mockSetCurrentView).toHaveBeenCalledWith(ViewType.Login);
    });
  });

  describe('Error clearing', () => {
    it('should clear error when user starts typing in field', async () => {
      render(<RegisterForm />);
      
      const submitButton = screen.getByRole('button', { name: /зарегистрироваться/i });
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
  });
});

