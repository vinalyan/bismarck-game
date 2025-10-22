import axios from 'axios';
import { authAPI } from '../authAPI';

// Get the mocked axios instance
const mockedAxios = axios.create() as jest.Mocked<typeof axios.create>;

describe('authAPI', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('register', () => {
    it('should register user successfully', async () => {
      const mockUser = {
        id: 'test-user-id',
        username: 'testuser',
        email: 'test@example.com',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      };

      mockedAxios.post.mockResolvedValueOnce({
        data: mockUser,
        status: 201
      });

      const result = await authAPI.register({
        username: 'testuser',
        email: 'test@example.com',
        password: 'password123'
      });

      expect(mockedAxios.post).toHaveBeenCalledWith('/auth/register', {
        username: 'testuser',
        email: 'test@example.com',
        password: 'password123'
      });
      expect(result).toEqual(mockUser);
    });

    it('should handle registration error', async () => {
      const errorMessage = 'Username already exists';
      mockedAxios.post.mockRejectedValueOnce(new Error(errorMessage));

      await expect(authAPI.register({
        username: 'existinguser',
        email: 'test@example.com',
        password: 'password123'
      })).rejects.toThrow(errorMessage);
    });
  });

  describe('login', () => {
    it('should login user successfully', async () => {
      const mockResponse = {
        user: {
          id: 'test-user-id',
          username: 'testuser',
          email: 'test@example.com'
        },
        token: 'jwt-token-here'
      };

      mockedAxios.post.mockResolvedValueOnce({
        data: mockResponse,
        status: 200
      });

      const result = await authAPI.login({
        username: 'testuser',
        password: 'password123'
      });

      expect(mockedAxios.post).toHaveBeenCalledWith('/auth/login', {
        username: 'testuser',
        password: 'password123'
      });
      expect(result).toEqual(mockResponse);
    });

    it('should handle login error', async () => {
      const errorMessage = 'Invalid username or password';
      mockedAxios.post.mockRejectedValueOnce(new Error(errorMessage));

      await expect(authAPI.login({
        username: 'testuser',
        password: 'wrongpassword'
      })).rejects.toThrow(errorMessage);
    });
  });

  describe('logout', () => {
    it('should logout user successfully', async () => {
      mockedAxios.post.mockResolvedValueOnce({
        data: { message: 'Logged out successfully' },
        status: 200
      });

      await authAPI.logout();

      expect(mockedAxios.post).toHaveBeenCalledWith('/auth/logout');
    });

    it('should handle logout error', async () => {
      const errorMessage = 'Logout failed';
      mockedAxios.post.mockRejectedValueOnce(new Error(errorMessage));

      await expect(authAPI.logout()).rejects.toThrow(errorMessage);
    });
  });

  describe('getCurrentUser', () => {
    it('should get current user successfully', async () => {
      const mockUser = {
        id: 'test-user-id',
        username: 'testuser',
        email: 'test@example.com',
        role: 'player',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      };

      mockedAxios.get.mockResolvedValueOnce({
        data: mockUser,
        status: 200
      });

      const result = await authAPI.getCurrentUser();

      expect(mockedAxios.get).toHaveBeenCalledWith('/auth/me');
      expect(result).toEqual(mockUser);
    });

    it('should handle get current user error', async () => {
      const errorMessage = 'Unauthorized';
      mockedAxios.get.mockRejectedValueOnce(new Error(errorMessage));

      await expect(authAPI.getCurrentUser()).rejects.toThrow(errorMessage);
    });
  });
});
