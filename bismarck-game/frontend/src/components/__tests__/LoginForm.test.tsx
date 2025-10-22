import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from '../LoginForm';

// Mock the authAPI
jest.mock('../../services/api/authAPI', () => ({
  authAPI: {
    login: jest.fn()
  }
}));

// Mock the gameStore
const mockGameStore = {
  login: jest.fn(),
  ui: {
    currentView: 'login'
  }
};

jest.mock('../../stores/gameStore', () => ({
  useGameStore: () => mockGameStore
}));

import { authAPI } from '../../services/api/authAPI';
import { useGameStore } from '../../stores/gameStore';

const mockAuthAPI = authAPI as jest.Mocked<typeof authAPI>;
// Remove the mock function since we're using a direct mock object

describe('LoginForm', () => {
  const mockLogin = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockGameStore.login = mockLogin;
  });

  it('should render login form', () => {
    render(<LoginForm />);
    
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument();
  });

  it('should handle form submission with valid data', async () => {
    const user = userEvent.setup();
    const mockResponse = {
      user: {
        id: 'test-user-id',
        username: 'testuser',
        email: 'test@example.com'
      },
      token: 'jwt-token'
    };

    mockAuthAPI.login.mockResolvedValueOnce(mockResponse);

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(mockAuthAPI.login).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'password123'
      });
    });

    expect(mockLogin).toHaveBeenCalledWith(mockResponse.user, mockResponse.token);
  });

  it('should display error message on login failure', async () => {
    const user = userEvent.setup();
    const errorMessage = 'Invalid username or password';

    mockAuthAPI.login.mockRejectedValueOnce(new Error(errorMessage));

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'wrongpassword');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });
  });

  it('should validate required fields', async () => {
    const user = userEvent.setup();

    render(<LoginForm />);

    await user.click(screen.getByRole('button', { name: /login/i }));

    expect(screen.getByText(/username is required/i)).toBeInTheDocument();
    expect(screen.getByText(/password is required/i)).toBeInTheDocument();
  });

  it('should show loading state during submission', async () => {
    const user = userEvent.setup();

    // Mock a delayed response
    mockAuthAPI.login.mockImplementationOnce(
      () => new Promise(resolve => setTimeout(() => resolve({} as any), 100))
    );

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /login/i }));

    expect(screen.getByText(/logging in/i)).toBeInTheDocument();
  });

  it('should clear form after successful login', async () => {
    const user = userEvent.setup();
    const mockResponse = {
      user: {
        id: 'test-user-id',
        username: 'testuser',
        email: 'test@example.com'
      },
      token: 'jwt-token'
    };

    mockAuthAPI.login.mockResolvedValueOnce(mockResponse);

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalled();
    });

    // Form should be cleared after successful login
    expect(screen.getByLabelText(/username/i)).toHaveValue('');
    expect(screen.getByLabelText(/password/i)).toHaveValue('');
  });
});
