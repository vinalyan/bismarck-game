import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from './Button';

describe('Button', () => {
  describe('Rendering', () => {
    it('should render button with children', () => {
      render(<Button>Click me</Button>);
      expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument();
    });

    it('should render button with text content', () => {
      render(<Button>Test Button</Button>);
      expect(screen.getByText('Test Button')).toBeInTheDocument();
    });

    it('should render button with React node as children', () => {
      render(
        <Button>
          <span>Span content</span>
        </Button>
      );
      expect(screen.getByText('Span content')).toBeInTheDocument();
    });
  });

  describe('Variants', () => {
    it('should apply primary variant by default', () => {
      const { container } = render(<Button>Primary</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--primary');
    });

    it('should apply primary variant when specified', () => {
      const { container } = render(<Button variant="primary">Primary</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--primary');
    });

    it('should apply secondary variant', () => {
      const { container } = render(<Button variant="secondary">Secondary</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--secondary');
    });

    it('should apply danger variant', () => {
      const { container } = render(<Button variant="danger">Danger</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--danger');
    });

    it('should apply success variant', () => {
      const { container } = render(<Button variant="success">Success</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--success');
    });
  });

  describe('Sizes', () => {
    it('should apply medium size by default', () => {
      const { container } = render(<Button>Medium</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--md');
    });

    it('should apply small size', () => {
      const { container } = render(<Button size="sm">Small</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--sm');
    });

    it('should apply medium size when specified', () => {
      const { container } = render(<Button size="md">Medium</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--md');
    });

    it('should apply large size', () => {
      const { container } = render(<Button size="lg">Large</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--lg');
    });
  });

  describe('Disabled state', () => {
    it('should not be disabled by default', () => {
      render(<Button>Enabled</Button>);
      const button = screen.getByRole('button');
      expect(button).not.toBeDisabled();
    });

    it('should be disabled when disabled prop is true', () => {
      render(<Button disabled>Disabled</Button>);
      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });

    it('should apply disabled class when disabled', () => {
      const { container } = render(<Button disabled>Disabled</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--disabled');
    });

    it('should not apply disabled class when enabled', () => {
      const { container } = render(<Button>Enabled</Button>);
      const button = container.querySelector('button');
      expect(button).not.toHaveClass('btn--disabled');
    });
  });

  describe('Click handling', () => {
    it('should call onClick when clicked', () => {
      const handleClick = jest.fn();
      
      render(<Button onClick={handleClick}>Click me</Button>);
      
      const button = screen.getByRole('button');
      userEvent.click(button);
      
      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should not call onClick when disabled', () => {
      const handleClick = jest.fn();
      
      render(<Button onClick={handleClick} disabled>Disabled</Button>);
      
      const button = screen.getByRole('button');
      userEvent.click(button);
      
      expect(handleClick).not.toHaveBeenCalled();
    });

    it('should work without onClick handler', () => {
      render(<Button>No handler</Button>);
      const button = screen.getByRole('button');
      expect(button).toBeInTheDocument();
      // Should not throw when clicked
      expect(() => button.click()).not.toThrow();
    });
  });

  describe('Type attribute', () => {
    it('should have type="button" by default', () => {
      render(<Button>Default</Button>);
      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'button');
    });

    it('should apply type="submit" when specified', () => {
      render(<Button type="submit">Submit</Button>);
      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'submit');
    });

    it('should apply type="reset" when specified', () => {
      render(<Button type="reset">Reset</Button>);
      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'reset');
    });
  });

  describe('Custom className', () => {
    it('should apply custom className', () => {
      const { container } = render(<Button className="custom-class">Custom</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('custom-class');
    });

    it('should combine custom className with default classes', () => {
      const { container } = render(
        <Button className="custom-class" variant="secondary">
          Custom
        </Button>
      );
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn');
      expect(button).toHaveClass('btn--secondary');
      expect(button).toHaveClass('custom-class');
    });

    it('should work with empty className', () => {
      const { container } = render(<Button className="">Empty</Button>);
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn');
    });
  });

  describe('Combined props', () => {
    it('should apply all props correctly together', () => {
      const handleClick = jest.fn();
      const { container } = render(
        <Button
          variant="danger"
          size="lg"
          className="custom-class"
          onClick={handleClick}
          type="submit"
        >
          Combined
        </Button>
      );
      
      const button = container.querySelector('button');
      expect(button).toHaveClass('btn--danger');
      expect(button).toHaveClass('btn--lg');
      expect(button).toHaveClass('custom-class');
      expect(button).toHaveAttribute('type', 'submit');
      
      button?.click();
      expect(handleClick).toHaveBeenCalled();
    });
  });
});

