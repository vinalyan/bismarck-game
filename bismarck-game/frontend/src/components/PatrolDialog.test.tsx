import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PatrolDialog from './PatrolDialog';

describe('PatrolDialog', () => {
  const mockUnits = [
    { id: 'unit-1', name: 'Bismarck', type: 'BB', is_patrolling: false },
    { id: 'unit-2', name: 'Prinz Eugen', type: 'CA', is_patrolling: true },
    { id: 'unit-3', name: 'Destroyer', type: 'DD', is_patrolling: false },
  ];

  const mockTaskForces = [
    { id: 'tf-1', name: 'Task Force 1', is_patrolling: false },
    { id: 'tf-2', name: 'Task Force 2', is_patrolling: true },
  ];

  const mockOnConfirm = jest.fn();
  const mockOnCancel = jest.fn();

  const defaultProps = {
    hexId: 'A1',
    units: mockUnits,
    taskForces: [],
    onConfirm: mockOnConfirm,
    onCancel: mockOnCancel,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render dialog with correct title', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      expect(screen.getByText(/Установить патруль в гексе A1/i)).toBeInTheDocument();
    });

    it('should render units without patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      expect(screen.getByText('Корабли без патруля:')).toBeInTheDocument();
      expect(screen.getByText('Bismarck (BB)')).toBeInTheDocument();
      expect(screen.getByText('Destroyer (DD)')).toBeInTheDocument();
    });

    it('should render units with patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      expect(screen.getByText('Корабли на патруле (можно снять):')).toBeInTheDocument();
      expect(screen.getByText('Prinz Eugen (CA) - на патруле')).toBeInTheDocument();
    });

    it('should render Task Forces without patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      expect(screen.getByText('Оперативные соединения без патруля:')).toBeInTheDocument();
      expect(screen.getByText('Task Force 1 (TF)')).toBeInTheDocument();
    });

    it('should render Task Forces with patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      expect(screen.getByText('Оперативные соединения на патруле (можно снять):')).toBeInTheDocument();
      expect(screen.getByText('Task Force 2 (TF) - на патруле')).toBeInTheDocument();
    });

    it('should show message when no units or task forces available', () => {
      render(<PatrolDialog {...defaultProps} units={[]} taskForces={[]} />);
      
      expect(screen.getByText(/Нет доступных кораблей или оперативных соединений/i)).toBeInTheDocument();
    });
  });

  describe('Unit Selection', () => {
    it('should allow selecting a unit without patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const bismarckRadio = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckRadio);
      
      expect(bismarckRadio).toBeChecked();
      expect(screen.getByText('Действие:')).toBeInTheDocument();
      // Текст "Установить патруль" встречается в описании и в кнопке
      expect(screen.getAllByText('Установить патруль').length).toBeGreaterThan(0);
    });

    it('should allow selecting a unit with patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const prinzRadio = screen.getByLabelText(/Prinz Eugen \(CA\) - на патруле/i);
      userEvent.click(prinzRadio);
      
      expect(prinzRadio).toBeChecked();
      expect(screen.getByText('Действие:')).toBeInTheDocument();
      // Текст "Снять патруль" встречается в описании и в кнопке
      expect(screen.getAllByText('Снять патруль').length).toBeGreaterThan(0);
    });

    it('should allow selecting a Task Force without patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const tf1Radio = screen.getByLabelText(/Task Force 1 \(TF\)/i);
      userEvent.click(tf1Radio);
      
      expect(tf1Radio).toBeChecked();
      expect(screen.getByText('Действие:')).toBeInTheDocument();
      // Текст "Установить патруль" встречается в описании и в кнопке
      expect(screen.getAllByText('Установить патруль').length).toBeGreaterThan(0);
    });

    it('should allow selecting a Task Force with patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const tf2Radio = screen.getByLabelText(/Task Force 2 \(TF\) - на патруле/i);
      userEvent.click(tf2Radio);
      
      expect(tf2Radio).toBeChecked();
      expect(screen.getByText('Действие:')).toBeInTheDocument();
      // Текст "Снять патруль" встречается в описании и в кнопке
      expect(screen.getAllByText('Снять патруль').length).toBeGreaterThan(0);
    });
  });

  describe('Button States', () => {
    it('should disable confirm button when no selection', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const confirmButton = screen.getByRole('button', { name: /Установить патруль/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should enable confirm button when unit selected', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const bismarckRadio = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckRadio);
      
      const confirmButton = screen.getByRole('button', { name: /Установить патруль/i });
      expect(confirmButton).not.toBeDisabled();
    });

    it('should show "Установить патруль" when setting patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const bismarckRadio = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckRadio);
      
      expect(screen.getByRole('button', { name: /Установить патруль/i })).toBeInTheDocument();
    });

    it('should show "Снять патруль" when removing patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const prinzRadio = screen.getByLabelText(/Prinz Eugen \(CA\) - на патруле/i);
      userEvent.click(prinzRadio);
      
      expect(screen.getByRole('button', { name: /Снять патруль/i })).toBeInTheDocument();
    });
  });

  describe('Confirm Action', () => {
    it('should call onConfirm with unit id when confirming unit patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const bismarckRadio = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckRadio);
      
      const confirmButton = screen.getByRole('button', { name: /Установить патруль/i });
      userEvent.click(confirmButton);
      
      expect(mockOnConfirm).toHaveBeenCalledWith('unit-1', true, false);
    });

    it('should call onConfirm with unit id and false when removing unit patrol', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const prinzRadio = screen.getByLabelText(/Prinz Eugen \(CA\) - на патруле/i);
      userEvent.click(prinzRadio);
      
      const confirmButton = screen.getByRole('button', { name: /Снять патруль/i });
      userEvent.click(confirmButton);
      
      expect(mockOnConfirm).toHaveBeenCalledWith('unit-2', false, false);
    });

    it('should call onConfirm with Task Force id when confirming TF patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const tf1Radio = screen.getByLabelText(/Task Force 1 \(TF\)/i);
      userEvent.click(tf1Radio);
      
      const confirmButton = screen.getByRole('button', { name: /Установить патруль/i });
      userEvent.click(confirmButton);
      
      expect(mockOnConfirm).toHaveBeenCalledWith('tf-1', true, true);
    });

    it('should call onConfirm with Task Force id and false when removing TF patrol', () => {
      render(<PatrolDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const tf2Radio = screen.getByLabelText(/Task Force 2 \(TF\) - на патруле/i);
      userEvent.click(tf2Radio);
      
      const confirmButton = screen.getByRole('button', { name: /Снять патруль/i });
      userEvent.click(confirmButton);
      
      expect(mockOnConfirm).toHaveBeenCalledWith('tf-2', false, true);
    });
  });

  describe('Cancel Action', () => {
    it('should call onCancel when cancel button clicked', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      const cancelButton = screen.getByRole('button', { name: /Отмена/i });
      userEvent.click(cancelButton);
      
      expect(mockOnCancel).toHaveBeenCalled();
    });

    it('should call onCancel when overlay clicked', () => {
      const { container } = render(<PatrolDialog {...defaultProps} />);
      
      const overlay = container.querySelector('.dialog-overlay');
      if (overlay) {
        userEvent.click(overlay);
        expect(mockOnCancel).toHaveBeenCalled();
      }
    });

    it('should not call onCancel when dialog content clicked', () => {
      const { container } = render(<PatrolDialog {...defaultProps} />);
      
      const dialogContent = container.querySelector('.dialog-content');
      if (dialogContent) {
        userEvent.click(dialogContent);
        expect(mockOnCancel).not.toHaveBeenCalled();
      }
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty units and task forces', () => {
      render(<PatrolDialog {...defaultProps} units={[]} taskForces={[]} />);
      
      expect(screen.getByText(/Установить патруль в гексе A1/i)).toBeInTheDocument();
      const confirmButton = screen.getByRole('button', { name: /Установить патруль/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should handle only units without task forces', () => {
      render(<PatrolDialog {...defaultProps} />);
      
      expect(screen.queryByText(/Оперативные соединения/i)).not.toBeInTheDocument();
      expect(screen.getByText(/Корабли без патруля:/i)).toBeInTheDocument();
    });

    it('should handle only task forces without units', () => {
      render(<PatrolDialog {...defaultProps} units={[]} taskForces={mockTaskForces} />);
      
      expect(screen.queryByText(/Корабли/i)).not.toBeInTheDocument();
      expect(screen.getByText(/Оперативные соединения без патруля:/i)).toBeInTheDocument();
    });
  });
});

