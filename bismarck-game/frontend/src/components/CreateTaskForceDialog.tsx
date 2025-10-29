import React, { useState } from 'react';
import './CreateTaskForceDialog.css';

interface CreateTaskForceDialogProps {
  hexId: string;
  units: any[];
  taskForces: any[];
  onConfirm: (selectedUnitIds: string[], formation: string) => void;
  onCancel: () => void;
}

const CreateTaskForceDialog: React.FC<CreateTaskForceDialogProps> = ({
  hexId,
  units,
  taskForces,
  onConfirm,
  onCancel,
}) => {
  const [selectedUnits, setSelectedUnits] = useState<string[]>([]);
  const [formation, setFormation] = useState<string>('line');

  const availableUnits = units.filter(
    (unit) => unit.detection_level !== 'shadowed' && !unit.task_force_id
  );

  const toggleUnitSelection = (unitId: string) => {
    setSelectedUnits((prev) =>
      prev.includes(unitId)
        ? prev.filter((id) => id !== unitId)
        : [...prev, unitId]
    );
  };

  const handleConfirm = () => {
    if (selectedUnits.length >= 2) {
      onConfirm(selectedUnits, formation);
    }
  };

  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <h3>Создать Task Force в гексе {hexId}</h3>
        
        <div className="dialog-section">
          <h4>Выберите юниты (минимум 2):</h4>
          <div className="units-list">
            {availableUnits.map((unit) => (
              <label key={unit.id} className="unit-checkbox">
                <input
                  type="checkbox"
                  checked={selectedUnits.includes(unit.id)}
                  onChange={() => toggleUnitSelection(unit.id)}
                />
                <span>{unit.name} ({unit.type})</span>
                {unit.detection_level === 'shadowed' && (
                  <span className="unit-disabled-hint"> - Нельзя (shadowed)</span>
                )}
              </label>
            ))}
          </div>
        </div>

        <div className="dialog-section">
          <h4>Формация:</h4>
          <select value={formation} onChange={(e) => setFormation(e.target.value)}>
            <option value="line">Line (Линия)</option>
            <option value="diamond">Diamond (Ромб)</option>
            <option value="wedge">Wedge (Клин)</option>
            <option value="scattered">Scattered (Рассеянная)</option>
          </select>
        </div>

        <div className="dialog-actions">
          <button onClick={handleConfirm} disabled={selectedUnits.length < 2}>
            Сформировать TF
          </button>
          <button onClick={onCancel}>Отмена</button>
        </div>
      </div>
    </div>
  );
};

export default CreateTaskForceDialog;
