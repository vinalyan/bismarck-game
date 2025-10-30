import React, { useState } from 'react';
import './CreateTaskForceDialog.css';

interface CreateTaskForceDialogProps {
  hexId: string;
  units: any[]; // units currently in this hex (may exclude TF members if their position is cleared)
  taskForces: any[];
  allUnits?: any[]; // all game units to resolve TF members by id
  onConfirm: (selectedUnitIds: string[]) => void;
  onAddToExisting: (taskForceId: string, unitId: string) => void;
  onRemoveFromTF: (taskForceId: string, unitId: string) => Promise<void> | void;
  onCancel: () => void;
}

const CreateTaskForceDialog: React.FC<CreateTaskForceDialogProps> = ({
  hexId,
  units,
  taskForces,
  allUnits,
  onConfirm,
  onAddToExisting,
  onRemoveFromTF,
  onCancel,
}) => {
  const [selectedUnits, setSelectedUnits] = useState<string[]>([]);
  // Убираем локальную мутацию позиций/состава — теперь это целиком делает бэкенд

  const availableUnits = units.filter(
    (unit) => unit.detection_level !== 'shadowed' && !unit.task_force_id
  );

  // Проверяем, есть ли в гексе существующие Task Force
  const existingTaskForces = taskForces.filter(tf => tf.position === hexId);
  const hasExistingTF = existingTaskForces.length > 0;

  console.log('CreateTaskForceDialog - hexId:', hexId, 'units:', units.length, 'availableUnits:', availableUnits.length, 'selectedUnits:', selectedUnits.length, 'hasExistingTF:', hasExistingTF);
  console.log('🔍 All units in hex:', units.map(u => ({ id: u.id, name: u.name, detection_level: u.detection_level, task_force_id: u.task_force_id })));
  console.log('🔍 Available units:', availableUnits.map(u => ({ id: u.id, name: u.name, detection_level: u.detection_level, task_force_id: u.task_force_id })));
  console.log('🔍 Existing Task Forces:', existingTaskForces.map(tf => ({ id: tf.id, name: tf.name })));

  const toggleUnitSelection = (unitId: string) => {
    console.log('🔄 toggleUnitSelection called with unitId:', unitId);
    setSelectedUnits((prev) => {
      const newSelection = prev.includes(unitId)
        ? prev.filter((id) => id !== unitId)
        : [...prev, unitId];
      console.log('✅ New selection:', newSelection);
      return newSelection;
    });
  };

  const handleConfirm = () => {
    console.log('🚢 handleConfirm called with:', { selectedUnits, selectedUnitsLength: selectedUnits.length, hasExistingTF });
    
    if (hasExistingTF && selectedUnits.length === 1) {
      // Если есть существующий TF и выбран только один юнит - добавляем к TF
      const taskForceId = existingTaskForces[0].id; // Берем первый TF в гексе
      const unitId = selectedUnits[0];
      console.log('✅ Adding unit to existing TF:', { taskForceId, unitId });
      onAddToExisting(taskForceId, unitId);
    } else if (selectedUnits.length >= 2) {
      // Если выбрано 2+ юнита - создаем новый TF
      console.log('✅ Creating new TF with:', { selectedUnits });
      onConfirm(selectedUnits);
    } else {
      console.log('❌ Not enough units selected:', selectedUnits.length);
    }
  };

  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <h3>
          {hasExistingTF 
            ? `Добавить юниты к Task Force в гексе ${hexId}` 
            : `Создать Task Force в гексе ${hexId}`
          }
        </h3>
        
        {hasExistingTF && (
          <div className="dialog-section">
            <h4>Существующие Task Force:</h4>
            <div className="existing-tfs">
              {existingTaskForces.map(tf => {
                // Resolve TF members: prefer allUnits (most reliable), fallback to local available units by task_force_id (will be empty for TF members)
                const tfUnits = (allUnits || []).filter(u => (u.task_force_id === tf.id) || (Array.isArray(tf.units) && tf.units.includes(u.id)));
                return (
                  <div key={tf.id} className="tf-info">
                    <strong>{tf.name}</strong>
                    {tfUnits.length > 0 && (
                      <ul className="tf-units-list">
                        {tfUnits.map(u => (
                          <li key={u.id} className="tf-unit-item">
                            {`${u.type} - ${u.name}`}
                            <button
                              className="tf-remove-btn"
                              title="Убрать из TF"
                              onClick={async () => {
                                try {
                                  await onRemoveFromTF(tf.id, u.id);
                                } catch (err) {
                                  console.error('Failed to remove unit from TF', err);
                                }
                              }}
                            >
                              -
                            </button>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}
        
        <div className="dialog-section">
          <h4>
            {hasExistingTF 
              ? "Выберите юниты для добавления (1+ юнит):" 
              : "Выберите юниты (минимум 2):"
            }
          </h4>
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


        <div className="dialog-actions">
          <button 
            onClick={() => {
              console.log('🚢 Action button clicked!', { selectedUnits, hasExistingTF });
              handleConfirm();
            }} 
            disabled={hasExistingTF ? selectedUnits.length < 1 : selectedUnits.length < 2}
            style={{ 
              backgroundColor: (hasExistingTF ? selectedUnits.length < 1 : selectedUnits.length < 2) ? '#ccc' : '#007bff',
              color: 'white',
              padding: '8px 16px',
              border: 'none',
              borderRadius: '4px',
              cursor: (hasExistingTF ? selectedUnits.length < 1 : selectedUnits.length < 2) ? 'not-allowed' : 'pointer'
            }}
          >
            {hasExistingTF 
              ? `Добавить к TF (${selectedUnits.length}/1+)` 
              : `Сформировать TF (${selectedUnits.length}/2+)`
            }
          </button>
          <button onClick={onCancel}>Отмена</button>
        </div>
      </div>
    </div>
  );
};

export default CreateTaskForceDialog;
