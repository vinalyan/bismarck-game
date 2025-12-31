import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CreateTaskForceDialog from './CreateTaskForceDialog';

describe('CreateTaskForceDialog', () => {
  const mockUnits = [
    { id: 'unit-1', name: 'Bismarck', type: 'BB', detection_level: 'sighted', task_force_id: null },
    { id: 'unit-2', name: 'Prinz Eugen', type: 'CA', detection_level: 'sighted', task_force_id: null },
    { id: 'unit-3', name: 'Destroyer', type: 'DD', detection_level: 'shadowed', task_force_id: null },
    { id: 'unit-4', name: 'In TF', type: 'CL', detection_level: 'sighted', task_force_id: 'tf-1' },
  ];

  const mockTaskForces = [
    { id: 'tf-1', name: 'Task Force 1', position: 'A1', units: ['unit-4'] },
  ];

  const mockAllUnits = [...mockUnits];

  const mockOnConfirm = jest.fn();
  const mockOnAddToExisting = jest.fn();
  const mockOnRemoveFromTF = jest.fn();
  const mockOnCancel = jest.fn();

  const defaultProps = {
    hexId: 'A1',
    units: mockUnits,
    taskForces: [],
    allUnits: mockAllUnits,
    onConfirm: mockOnConfirm,
    onAddToExisting: mockOnAddToExisting,
    onRemoveFromTF: mockOnRemoveFromTF,
    onCancel: mockOnCancel,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render dialog with correct title when no existing TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      expect(screen.getByText(/Создать Task Force в гексе A1/i)).toBeInTheDocument();
    });

    it('should render dialog with correct title when existing TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      expect(screen.getByText(/Добавить юниты к Task Force в гексе A1/i)).toBeInTheDocument();
    });

    it('should render available units', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      expect(screen.getByText('Bismarck (BB)')).toBeInTheDocument();
      expect(screen.getByText('Prinz Eugen (CA)')).toBeInTheDocument();
    });

    it('should not render shadowed units', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      expect(screen.queryByText('Destroyer (DD)')).not.toBeInTheDocument();
    });

    it('should not render units already in TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      expect(screen.queryByText('In TF (CL)')).not.toBeInTheDocument();
    });

    it('should render existing Task Forces when present', () => {
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      expect(screen.getByText('Существующие Task Force:')).toBeInTheDocument();
      expect(screen.getByText('Task Force 1')).toBeInTheDocument();
    });
  });

  describe('Unit Selection', () => {
    it('should allow selecting units', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      
      expect(bismarckCheckbox).toBeChecked();
    });

    it('should allow deselecting units', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      expect(bismarckCheckbox).toBeChecked();
      
      userEvent.click(bismarckCheckbox);
      expect(bismarckCheckbox).not.toBeChecked();
    });

    it('should allow selecting multiple units', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      const prinzCheckbox = screen.getByLabelText(/Prinz Eugen \(CA\)/i);
      
      userEvent.click(bismarckCheckbox);
      userEvent.click(prinzCheckbox);
      
      expect(bismarckCheckbox).toBeChecked();
      expect(prinzCheckbox).toBeChecked();
    });
  });

  describe('Button States', () => {
    it('should disable confirm button when no units selected for new TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const confirmButton = screen.getByRole('button', { name: /Формировать TF/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should disable confirm button when only one unit selected for new TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      
      const confirmButton = screen.getByRole('button', { name: /Формировать TF/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should enable confirm button when 2+ units selected for new TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      const prinzCheckbox = screen.getByLabelText(/Prinz Eugen \(CA\)/i);
      
      userEvent.click(bismarckCheckbox);
      userEvent.click(prinzCheckbox);
      
      const confirmButton = screen.getByRole('button', { name: /Формировать TF/i });
      expect(confirmButton).not.toBeDisabled();
    });

    it('should disable confirm button when no units selected for existing TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const confirmButton = screen.getByRole('button', { name: /Добавить к TF/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should enable confirm button when 1+ unit selected for existing TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      
      const confirmButton = screen.getByRole('button', { name: /Добавить к TF/i });
      expect(confirmButton).not.toBeDisabled();
    });
  });

  describe('Confirm Action', () => {
    it('should call onConfirm with selected units when creating new TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      const prinzCheckbox = screen.getByLabelText(/Prinz Eugen \(CA\)/i);
      
      userEvent.click(bismarckCheckbox);
      userEvent.click(prinzCheckbox);
      
      const confirmButton = screen.getByRole('button', { name: /Формировать TF/i });
      userEvent.click(confirmButton);
      
      expect(mockOnConfirm).toHaveBeenCalledWith(['unit-1', 'unit-2']);
      expect(mockOnAddToExisting).not.toHaveBeenCalled();
    });

    it('should call onAddToExisting when adding to existing TF', () => {
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      
      const confirmButton = screen.getByRole('button', { name: /Добавить к TF/i });
      userEvent.click(confirmButton);
      
      expect(mockOnAddToExisting).toHaveBeenCalledWith('tf-1', 'unit-1');
      expect(mockOnConfirm).not.toHaveBeenCalled();
    });
  });

  describe('Cancel Action', () => {
    it('should call onCancel when cancel button clicked', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const cancelButton = screen.getByRole('button', { name: /Отмена/i });
      userEvent.click(cancelButton);
      
      expect(mockOnCancel).toHaveBeenCalled();
    });

    it('should call onCancel when overlay clicked', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const overlay = screen.getByText(/Создать Task Force/i).closest('.dialog-overlay');
      if (overlay) {
        userEvent.click(overlay);
        expect(mockOnCancel).toHaveBeenCalled();
      }
    });

    it('should not call onCancel when dialog content clicked', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const dialogContent = screen.getByText(/Создать Task Force/i).closest('.dialog-content');
      if (dialogContent) {
        userEvent.click(dialogContent);
        expect(mockOnCancel).not.toHaveBeenCalled();
      }
    });
  });

  describe('Remove from TF', () => {
    it('should call onRemoveFromTF when remove button clicked', async () => {
      mockOnRemoveFromTF.mockResolvedValue(undefined);
      
      render(<CreateTaskForceDialog {...defaultProps} taskForces={mockTaskForces} allUnits={mockAllUnits} />);
      
      const removeButtons = screen.getAllByTitle('Убрать из TF');
      expect(removeButtons.length).toBeGreaterThan(0);
      
      userEvent.click(removeButtons[0]);
      
      await waitFor(() => {
        expect(mockOnRemoveFromTF).toHaveBeenCalled();
      });
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty units list', () => {
      render(<CreateTaskForceDialog {...defaultProps} units={[]} />);
      
      expect(screen.getByText(/Создать Task Force/i)).toBeInTheDocument();
      const confirmButton = screen.getByRole('button', { name: /Формировать TF/i });
      expect(confirmButton).toBeDisabled();
    });

    it('should show correct button text with selection count', () => {
      render(<CreateTaskForceDialog {...defaultProps} />);
      
      const bismarckCheckbox = screen.getByLabelText(/Bismarck \(BB\)/i);
      userEvent.click(bismarckCheckbox);
      
      expect(screen.getByText(/Формировать TF \(1\/2\+\)/i)).toBeInTheDocument();
    });
  });
});

