import React, { useState } from 'react';
import './PatrolDialog.css';

interface PatrolDialogProps {
  hexId: string;
  units: any[]; // Морские юниты в гексе (уже отфильтрованы по стороне, без ТФ)
  onConfirm: (unitId: string, isPatrolling: boolean) => void;
  onCancel: () => void;
}

const PatrolDialog: React.FC<PatrolDialogProps> = ({
  hexId,
  units,
  onConfirm,
  onCancel,
}) => {
  const [selectedUnitId, setSelectedUnitId] = useState<string | null>(null);
  const [isSettingPatrol, setIsSettingPatrol] = useState(true); // true = установить, false = снять

  // Фильтруем юниты, которые уже на патруле
  const unitsWithoutPatrol = units.filter(unit => !unit.is_patrolling);
  const unitsWithPatrol = units.filter(unit => unit.is_patrolling);

  const handleConfirm = () => {
    if (selectedUnitId) {
      onConfirm(selectedUnitId, isSettingPatrol);
    }
  };

  // Если выбран юнит, который уже на патруле - автоматически переключаем на снятие патруля
  const handleUnitSelection = (unitId: string) => {
    setSelectedUnitId(unitId);
    const unit = units.find(u => u.id === unitId);
    setIsSettingPatrol(!unit?.is_patrolling);
  };

  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <h3>Установить патруль в гексе {hexId}</h3>
        
        <div className="dialog-section">
          <h4>Выберите корабль:</h4>
          <div className="units-list">
            {unitsWithoutPatrol.length > 0 && (
              <>
                <h5 style={{ marginTop: '10px', marginBottom: '5px', fontSize: '0.9em', color: '#666' }}>
                  Корабли без патруля:
                </h5>
                {unitsWithoutPatrol.map((unit) => (
                  <label 
                    key={unit.id} 
                    className={`unit-radio ${selectedUnitId === unit.id ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="unit"
                      value={unit.id}
                      checked={selectedUnitId === unit.id}
                      onChange={() => handleUnitSelection(unit.id)}
                    />
                    <span>{unit.name} ({unit.type})</span>
                  </label>
                ))}
              </>
            )}
            
            {unitsWithPatrol.length > 0 && (
              <>
                <h5 style={{ marginTop: '15px', marginBottom: '5px', fontSize: '0.9em', color: '#666' }}>
                  Корабли на патруле (можно снять):
                </h5>
                {unitsWithPatrol.map((unit) => (
                  <label 
                    key={unit.id} 
                    className={`unit-radio ${selectedUnitId === unit.id ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="unit"
                      value={unit.id}
                      checked={selectedUnitId === unit.id}
                      onChange={() => handleUnitSelection(unit.id)}
                    />
                    <span>{unit.name} ({unit.type}) - на патруле</span>
                  </label>
                ))}
              </>
            )}
            
            {units.length === 0 && (
              <p style={{ color: '#999', fontStyle: 'italic' }}>
                Нет доступных кораблей для патруля в этом гексе
              </p>
            )}
          </div>
        </div>

        {selectedUnitId && (
          <div className="dialog-section" style={{ backgroundColor: '#f8f9fa', padding: '10px', borderRadius: '4px' }}>
            <p style={{ margin: 0, color: '#000' }}>
              <strong>Действие:</strong> {isSettingPatrol ? 'Установить патруль' : 'Снять патруль'}
            </p>
          </div>
        )}

        <div className="dialog-actions">
          <button 
            onClick={handleConfirm}
            disabled={!selectedUnitId}
            style={{ 
              backgroundColor: selectedUnitId ? '#007bff' : '#ccc',
              color: 'white',
              padding: '8px 16px',
              border: 'none',
              borderRadius: '4px',
              cursor: selectedUnitId ? 'pointer' : 'not-allowed'
            }}
          >
            {isSettingPatrol ? 'Установить патруль' : 'Снять патруль'}
          </button>
          <button onClick={onCancel}>Отмена</button>
        </div>
      </div>
    </div>
  );
};

export default PatrolDialog;

