import React from 'react';
import { render, screen } from '@testing-library/react';
import Tooltip from './Tooltip';

describe('Tooltip', () => {
  const mockContent = {
    hexId: 'A1',
    hexType: 'water',
    features: [],
  };

  describe('Visibility', () => {
    it('should not render when visible is false', () => {
      const { container } = render(
        <Tooltip visible={false} x={100} y={200} content={mockContent} />
      );
      
      expect(container.firstChild).toBeNull();
    });

    it('should render when visible is true', () => {
      render(<Tooltip visible={true} x={100} y={200} content={mockContent} />);
      
      expect(screen.getByText(/гекс a1/i)).toBeInTheDocument();
    });
  });

  describe('Positioning', () => {
    it('should position tooltip correctly', () => {
      const { container } = render(
        <Tooltip visible={true} x={100} y={200} content={mockContent} />
      );
      
      const tooltip = container.querySelector('.tooltip');
      expect(tooltip).toHaveStyle({
        position: 'fixed',
        left: '137.8px', // 100 + 37.8
        top: '237.8px',  // 200 + 37.8
        zIndex: '1000',
      });
    });

    it('should handle different x and y coordinates', () => {
      const { container } = render(
        <Tooltip visible={true} x={50} y={75} content={mockContent} />
      );
      
      const tooltip = container.querySelector('.tooltip');
      expect(tooltip).toHaveStyle({
        left: '87.8px', // 50 + 37.8
        top: '112.8px', // 75 + 37.8
      });
    });
  });

  describe('Content rendering', () => {
    it('should render hexId in header', () => {
      render(<Tooltip visible={true} x={100} y={200} content={mockContent} />);
      
      expect(screen.getByText(/гекс a1/i)).toBeInTheDocument();
    });

    it('should render hex type', () => {
      render(<Tooltip visible={true} x={100} y={200} content={mockContent} />);
      
      expect(screen.getByText(/тип:/i)).toBeInTheDocument();
      expect(screen.getByText(/море/i)).toBeInTheDocument(); // water -> Море
    });

    it('should not render features section when features array is empty', () => {
      render(<Tooltip visible={true} x={100} y={200} content={mockContent} />);
      
      expect(screen.queryByText(/особенности:/i)).not.toBeInTheDocument();
    });

    it('should render features when provided', () => {
      const contentWithFeatures = {
        hexId: 'B2',
        hexType: 'water',
        features: ['fog', 'restricted_dd'],
      };

      render(<Tooltip visible={true} x={100} y={200} content={contentWithFeatures} />);
      
      expect(screen.getByText(/особенности:/i)).toBeInTheDocument();
      expect(screen.getByText(/туман/i)).toBeInTheDocument(); // fog -> Туман
      expect(screen.getByText(/зона действия немецких dd/i)).toBeInTheDocument(); // restricted_dd -> Зона действия немецких DD
    });
  });

  describe('Hex type display names', () => {
    const hexTypes = [
      { type: 'water', expected: 'Море' },
      { type: 'land', expected: 'Суша' },
      { type: 'non_game', expected: 'Неигровой' },
      { type: 'port', expected: 'Порт' },
      { type: 'ice', expected: 'Лёд' },
      { type: 'fog', expected: 'Туман' },
      { type: 'unknown', expected: 'Неизвестно' },
    ];

    hexTypes.forEach(({ type, expected }) => {
      it(`should display correct name for hex type "${type}"`, () => {
        const content = {
          hexId: 'A1',
          hexType: type,
          features: [],
        };

        render(<Tooltip visible={true} x={100} y={200} content={content} />);
        expect(screen.getByText(expected)).toBeInTheDocument();
      });
    });
  });

  describe('Feature display names', () => {
    const features = [
      { feature: 'fog', expected: 'Туман' },
      { feature: 'port', expected: 'Порт' },
      { feature: 'airport', expected: 'Аэропорт' },
      { feature: 'air_sector', expected: 'Зона действия авиации' },
      { feature: 'restricted_dd', expected: 'Зона действия немецких DD' },
      { feature: 'ice', expected: 'Ледяное поле' },
      { feature: 'unknown_feature', expected: 'unknown_feature' }, // fallback
    ];

    features.forEach(({ feature, expected }) => {
      it(`should display correct name for feature "${feature}"`, () => {
        const content = {
          hexId: 'A1',
          hexType: 'water',
          features: [feature],
        };

        render(<Tooltip visible={true} x={100} y={200} content={content} />);
        expect(screen.getByText(expected)).toBeInTheDocument();
      });
    });
  });

  describe('Multiple features', () => {
    it('should render all features in a list', () => {
      const content = {
        hexId: 'C3',
        hexType: 'water',
        features: ['fog', 'port', 'airport'],
      };

      render(<Tooltip visible={true} x={100} y={200} content={content} />);
      
      expect(screen.getByText(/туман/i)).toBeInTheDocument();
      // Используем точное совпадение для "Порт" чтобы избежать конфликта с "Аэропорт"
      expect(screen.getByText('Порт')).toBeInTheDocument();
      expect(screen.getByText(/аэропорт/i)).toBeInTheDocument();
    });
  });

  describe('Structure', () => {
    it('should have correct CSS classes', () => {
      const { container } = render(
        <Tooltip visible={true} x={100} y={200} content={mockContent} />
      );
      
      expect(container.querySelector('.tooltip')).toBeInTheDocument();
      expect(container.querySelector('.tooltip-content')).toBeInTheDocument();
      expect(container.querySelector('.tooltip-header')).toBeInTheDocument();
      expect(container.querySelector('.tooltip-body')).toBeInTheDocument();
      expect(container.querySelector('.tooltip-type')).toBeInTheDocument();
    });

    it('should have tooltip-type class with hex type', () => {
      const { container } = render(
        <Tooltip visible={true} x={100} y={200} content={mockContent} />
      );
      
      const typeValue = container.querySelector('.tooltip-type-water');
      expect(typeValue).toBeInTheDocument();
    });
  });
});

