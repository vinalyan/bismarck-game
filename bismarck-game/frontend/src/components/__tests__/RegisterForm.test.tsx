import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import RegisterForm from '../RegisterForm';

// Mock the authAPI
jest.mock('../../services/api/authAPI', () => ({
  authAPI: {
    register: jest.fn()
  }
}));

// Mock the gameStore
const mockGameStore = {
  login: jest.fn(),
  ui: {
    currentView: 'register'
  }
};

jest.mock('../../stores/gameStore', () => ({
  useGameStore: () => mockGameStore
}));

import { authAPI } from '../../services/api/authAPI';
import { useGameStore } from '../../stores/gameStore';

const mockAuthAPI = authAPI as jest.Mocked<typeof authAPI>;
describe('RegisterForm', () => {
  const mockLogin = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockGameStore.login = mockLogin;
  });

  it('should render registration form', () => {
    render(<RegisterForm />);
    
    expect(screen.getByLabelText(/имя пользователя/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/пароль/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/подтверждение пароля/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /зарегистрироваться/i })).toBeInTheDocument();
  });

  it('should handle form submission with valid data', async () => {
    const user = userEvent.setup();
    const mockUser = {
      id: 'test-user-id',
      username: 'testuser',
      email: 'test@example.com',
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z'
    };

    mockAuthAPI.register.mockResolvedValueOnce(mockUser);

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'password123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    await waitFor(() => {
      expect(mockAuthAPI.register).toHaveBeenCalledWith({
        username: 'testuser',
        email: 'test@example.com',
        password: 'password123'
      });
    });

    expect(mockLogin).toHaveBeenCalledWith(mockUser, expect.any(String));
  });

  it('should display error message on registration failure', async () => {
    const user = userEvent.setup();
    const errorMessage = 'Username already exists';

    mockAuthAPI.register.mockRejectedValueOnce(new Error(errorMessage));

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'existinguser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'password123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });
  });

  it('should validate required fields', async () => {
    const user = userEvent.setup();

    render(<RegisterForm />);

    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    expect(screen.getByText(/username is required/i)).toBeInTheDocument();
    expect(screen.getByText(/email is required/i)).toBeInTheDocument();
    expect(screen.getByText(/password is required/i)).toBeInTheDocument();
  });

  it('should validate password length', async () => {
    const user = userEvent.setup();

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), '123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), '123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    expect(screen.getByText(/password must be at least 6 characters/i)).toBeInTheDocument();
  });

  it('should validate password confirmation', async () => {
    const user = userEvent.setup();

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'differentpassword');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument();
  });

  it('should validate email format', async () => {
    const user = userEvent.setup();

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'invalid-email');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'password123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    expect(screen.getByText(/invalid email format/i)).toBeInTheDocument();
  });

  it('should show loading state during submission', async () => {
    const user = userEvent.setup();

    // Mock a delayed response
    mockAuthAPI.register.mockImplementationOnce(
      () => new Promise(resolve => setTimeout(() => resolve({} as any), 100))
    );

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'password123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    expect(screen.getByText(/registering/i)).toBeInTheDocument();
  });

  it('should clear form after successful registration', async () => {
    const user = userEvent.setup();
    const mockUser = {
      id: 'test-user-id',
      username: 'testuser',
      email: 'test@example.com',
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z'
    };

    mockAuthAPI.register.mockResolvedValueOnce(mockUser);

    render(<RegisterForm />);

    await user.type(screen.getByLabelText(/имя пользователя/i), 'testuser');
    await user.type(screen.getByLabelText(/email/i), 'test@example.com');
    await user.type(screen.getByLabelText(/пароль/i), 'password123');
    await user.type(screen.getByLabelText(/подтверждение пароля/i), 'password123');
    await user.click(screen.getByRole('button', { name: /зарегистрироваться/i }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalled();
    });

    // Form should be cleared after successful registration
    expect(screen.getByLabelText(/username/i)).toHaveValue('');
    expect(screen.getByLabelText(/email/i)).toHaveValue('');
    expect(screen.getByLabelText(/password/i)).toHaveValue('');
    expect(screen.getByLabelText(/confirm password/i)).toHaveValue('');
  });
});
