import React, { useState } from 'react';
import './PatrolDialog.css';

interface PatrolDialogProps {
  hexId: string;
  units: any[]; // Морские юниты в гексе (уже отфильтрованы по стороне, без ТФ)
  taskForces?: any[]; // Task Forces в гексе
  onConfirm: (id: string, isPatrolling: boolean, isTaskForce?: boolean) => void;
  onCancel: () => void;
}

const PatrolDialog: React.FC<PatrolDialogProps> = ({
  hexId,
  units,
  taskForces = [],
  onConfirm,
  onCancel,
}) => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isTaskForce, setIsTaskForce] = useState<boolean>(false);
  const [isSettingPatrol, setIsSettingPatrol] = useState(true); // true = установить, false = снять

  // Фильтруем юниты, которые уже на патруле
  const unitsWithoutPatrol = units.filter(unit => !unit.is_patrolling);
  const unitsWithPatrol = units.filter(unit => unit.is_patrolling);
  
  // Фильтруем Task Forces, которые уже на патруле
  const taskForcesWithoutPatrol = taskForces.filter(tf => !tf.is_patrolling);
  const taskForcesWithPatrol = taskForces.filter(tf => tf.is_patrolling);

  const handleConfirm = () => {
    if (selectedId) {
      onConfirm(selectedId, isSettingPatrol, isTaskForce);
    }
  };

  // Если выбран юнит или ТФ, который уже на патруле - автоматически переключаем на снятие патруля
  const handleSelection = (id: string, isTF: boolean) => {
    setSelectedId(id);
    setIsTaskForce(isTF);
    if (isTF) {
      const tf = taskForces.find(t => t.id === id);
      setIsSettingPatrol(!tf?.is_patrolling);
    } else {
      const unit = units.find(u => u.id === id);
      setIsSettingPatrol(!unit?.is_patrolling);
    }
  };

  return (
    <div className="dialog-overlay" onClick={onCancel}>
      <div className="dialog-content" onClick={(e) => e.stopPropagation()}>
        <h3>Установить патруль в гексе {hexId}</h3>
        
        <div className="dialog-section">
          <h4>Выберите корабль или оперативное соединение:</h4>
          <div className="units-list">
            {taskForcesWithoutPatrol.length > 0 && (
              <>
                <h5 style={{ marginTop: '10px', marginBottom: '5px', fontSize: '0.9em', color: '#666' }}>
                  Оперативные соединения без патруля:
                </h5>
                {taskForcesWithoutPatrol.map((tf) => (
                  <label 
                    key={tf.id} 
                    className={`unit-radio ${selectedId === tf.id ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="patrolTarget"
                      value={tf.id}
                      checked={selectedId === tf.id && isTaskForce}
                      onChange={() => handleSelection(tf.id, true)}
                    />
                    <span>{tf.name} (TF)</span>
                  </label>
                ))}
              </>
            )}
            
            {taskForcesWithPatrol.length > 0 && (
              <>
                <h5 style={{ marginTop: '15px', marginBottom: '5px', fontSize: '0.9em', color: '#666' }}>
                  Оперативные соединения на патруле (можно снять):
                </h5>
                {taskForcesWithPatrol.map((tf) => (
                  <label 
                    key={tf.id} 
                    className={`unit-radio ${selectedId === tf.id ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="patrolTarget"
                      value={tf.id}
                      checked={selectedId === tf.id && isTaskForce}
                      onChange={() => handleSelection(tf.id, true)}
                    />
                    <span>{tf.name} (TF) - на патруле</span>
                  </label>
                ))}
              </>
            )}
            
            {unitsWithoutPatrol.length > 0 && (
              <>
                <h5 style={{ marginTop: taskForces.length > 0 ? '15px' : '10px', marginBottom: '5px', fontSize: '0.9em', color: '#666' }}>
                  Корабли без патруля:
                </h5>
                {unitsWithoutPatrol.map((unit) => (
                  <label 
                    key={unit.id} 
                    className={`unit-radio ${selectedId === unit.id && !isTaskForce ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="patrolTarget"
                      value={unit.id}
                      checked={selectedId === unit.id && !isTaskForce}
                      onChange={() => handleSelection(unit.id, false)}
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
                    className={`unit-radio ${selectedId === unit.id && !isTaskForce ? 'selected' : ''}`}
                  >
                    <input
                      type="radio"
                      name="patrolTarget"
                      value={unit.id}
                      checked={selectedId === unit.id && !isTaskForce}
                      onChange={() => handleSelection(unit.id, false)}
                    />
                    <span>{unit.name} ({unit.type}) - на патруле</span>
                  </label>
                ))}
              </>
            )}
            
            {units.length === 0 && taskForces.length === 0 && (
              <p style={{ color: '#999', fontStyle: 'italic' }}>
                Нет доступных кораблей или оперативных соединений для патруля в этом гексе
              </p>
            )}
          </div>
        </div>

        {selectedId && (
          <div className="dialog-section" style={{ backgroundColor: '#f8f9fa', padding: '10px', borderRadius: '4px' }}>
            <p style={{ margin: 0, color: '#000' }}>
              <strong>Действие:</strong> {isSettingPatrol ? 'Установить патруль' : 'Снять патруль'}
            </p>
          </div>
        )}

        <div className="dialog-actions">
          <button 
            onClick={handleConfirm}
            disabled={!selectedId}
            style={{ 
              backgroundColor: selectedId ? '#007bff' : '#ccc',
              color: 'white',
              padding: '8px 16px',
              border: 'none',
              borderRadius: '4px',
              cursor: selectedId ? 'pointer' : 'not-allowed'
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

