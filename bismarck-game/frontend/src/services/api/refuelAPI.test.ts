import axios from 'axios';
import { refuelAPI, RefuelAllRequest } from './refuelAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('refuelAPI', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('refuelAll', () => {
    it('should call POST /api/refuel/all with correct data', async () => {
      const request: RefuelAllRequest = {
        game_id: 'game-1',
        fuel_amount: 10,
      };

      const mockResponse = {
        data: {
          success: true,
          data: {
            message: 'Refueled successfully',
            refueled_count: 5,
            total_units: 5,
            fuel_amount: 10,
          },
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAll(request);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        'http://localhost:8080/api/refuel/all',
        request
      );
      expect(result).toEqual(mockResponse.data);
      expect(result.success).toBe(true);
      expect(result.data.refueled_count).toBe(5);
    });

    it('should handle error response', async () => {
      const request: RefuelAllRequest = {
        game_id: 'game-1',
        fuel_amount: 10,
      };

      const mockResponse = {
        data: {
          success: false,
          error: 'Refuel failed',
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await refuelAPI.refuelAll(request);

      expect(result).toEqual(mockResponse.data);
      expect(result.success).toBe(false);
    });
  });
});

