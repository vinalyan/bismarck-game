import React from 'react';
import { render } from '@testing-library/react';
import RoutePath from './RoutePath';
import { HexCoordinate } from '../types/mapTypes';

describe('RoutePath', () => {
  const createHexCoordinate = (col: number, row: number, letter: string, number: number): HexCoordinate => ({
    col,
    row,
    letter,
    number,
  });

  describe('Rendering', () => {
    it('should not render when path has less than 2 coordinates', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      expect(container.firstChild).toBeNull();
    });

    it('should not render when path is empty', () => {
      const path: HexCoordinate[] = [];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      expect(container.firstChild).toBeNull();
    });

    it('should render when path has 2 or more coordinates', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgGroup = container.querySelector('g.route-path');
      expect(svgGroup).toBeInTheDocument();
    });
  });

  describe('Path rendering', () => {
    it('should create SVG path element', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
        createHexCoordinate(2, 0, 'A', 3),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toBeInTheDocument();
      expect(svgPath).toHaveAttribute('fill', 'none');
      expect(svgPath).toHaveAttribute('stroke-linecap', 'round');
      expect(svgPath).toHaveAttribute('stroke-linejoin', 'round');
      expect(svgPath).toHaveAttribute('opacity', '0.8');
    });

    it('should have path data attribute', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toHaveAttribute('d');
      expect(svgPath?.getAttribute('d')).toMatch(/^M /); // Path should start with M (Move to)
      expect(svgPath?.getAttribute('d')).toMatch(/ L /); // Should contain L (Line to)
    });

    it('should use correct stroke width', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toHaveAttribute('stroke-width', '4');
    });
  });

  describe('Player side colors', () => {
    it('should use german color (#ff6b6b) for german side', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toHaveAttribute('stroke', '#ff6b6b');
    });

    it('should use allied color (#4ecdc4) for allied side', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="allied" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toHaveAttribute('stroke', '#4ecdc4');
    });
  });

  describe('Path points', () => {
    it('should not render circles on first and last points', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
        createHexCoordinate(2, 0, 'A', 3),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const circles = container.querySelectorAll('circle');
      // Should have circles only for middle points (index 1), not for first (0) and last (2)
      expect(circles.length).toBe(1);
    });

    it('should render circles for middle points', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
        createHexCoordinate(2, 0, 'A', 3),
        createHexCoordinate(3, 0, 'A', 4),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const circles = container.querySelectorAll('circle');
      // Should have circles for points at index 1 and 2 (middle points)
      expect(circles.length).toBe(2);
    });

    it('should use correct color for circles based on player side', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
        createHexCoordinate(2, 0, 'A', 3),
      ];

      const { container: germanContainer } = render(
        <RoutePath path={path} playerSide="german" />
      );
      const germanCircles = germanContainer.querySelectorAll('circle');
      germanCircles.forEach(circle => {
        expect(circle).toHaveAttribute('fill', '#ff6b6b');
      });

      const { container: alliedContainer } = render(
        <RoutePath path={path} playerSide="allied" />
      );
      const alliedCircles = alliedContainer.querySelectorAll('circle');
      alliedCircles.forEach(circle => {
        expect(circle).toHaveAttribute('fill', '#4ecdc4');
      });
    });

    it('should set correct circle properties', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
        createHexCoordinate(2, 0, 'A', 3),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const circle = container.querySelector('circle');
      expect(circle).toHaveAttribute('r', '3');
      expect(circle).toHaveAttribute('opacity', '0.9');
      expect(circle).toHaveAttribute('cx');
      expect(circle).toHaveAttribute('cy');
    });
  });

  describe('Long paths', () => {
    it('should handle paths with many coordinates', () => {
      const path: HexCoordinate[] = Array.from({ length: 10 }, (_, i) =>
        createHexCoordinate(i, 0, 'A', i + 1)
      );

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toBeInTheDocument();
      
      // Should have circles for all middle points (indices 1-8, not 0 and 9)
      const circles = container.querySelectorAll('circle');
      expect(circles.length).toBe(8);
    });
  });

  describe('Edge cases', () => {
    it('should handle path with exactly 2 coordinates', () => {
      const path: HexCoordinate[] = [
        createHexCoordinate(0, 0, 'A', 1),
        createHexCoordinate(1, 0, 'A', 2),
      ];

      const { container } = render(<RoutePath path={path} playerSide="german" />);
      
      const svgPath = container.querySelector('path');
      expect(svgPath).toBeInTheDocument();
      
      // No circles should be rendered (only first and last points)
      const circles = container.querySelectorAll('circle');
      expect(circles.length).toBe(0);
    });
  });
});

