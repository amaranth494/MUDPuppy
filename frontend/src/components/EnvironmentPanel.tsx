import { useState, useEffect, useCallback } from 'react';
import { Variable, VariableType } from '../types';
import { getEnvironment, putEnvironment } from '../services/api';

interface EnvironmentPanelProps {
  connectionId: string;
  isConnectedToThis: boolean;
  onVariablesChange?: (variables: Variable[]) => void;
}

export default function EnvironmentPanel({ connectionId, isConnectedToThis, onVariablesChange }: EnvironmentPanelProps) {
  const [variables, setVariables] = useState<Variable[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [variableErrors, setVariableErrors] = useState<Record<string, string>>({});
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  // Validate single variable
  const validateVariable = (variable: Variable, allVariables: Variable[]): string | null => {
    if (!variable.name || !variable.name.trim()) {
      return 'Name is required';
    }
    if (variable.name.includes('${') || variable.name.includes('}')) {
      return 'Name cannot contain ${} syntax';
    }
    // Check for unique name (excluding current variable)
    const duplicate = allVariables.find(v => v.id !== variable.id && v.name === variable.name);
    if (duplicate) {
      return 'Name must be unique';
    }
    return null;
  };

  // Validate all variables
  const validateVariables = (): boolean => {
    const errors: Record<string, string> = {};
    let hasErrors = false;
    
    variables.forEach(variable => {
      const error = validateVariable(variable, variables);
      if (error) {
        errors[variable.id] = error;
        hasErrors = true;
      }
    });
    
    setVariableErrors(errors);
    return !hasErrors;
  };

  // Load variables
  const loadVariables = useCallback(async () => {
    if (!connectionId) return;
    
    setIsLoading(true);
    setError(null);
    try {
      const data = await getEnvironment(connectionId);
      setVariables(data.items || []);
      if (onVariablesChange) {
        onVariablesChange(data.items || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load variables');
    } finally {
      setIsLoading(false);
    }
  }, [connectionId, onVariablesChange]);

  useEffect(() => {
    loadVariables();
  }, [loadVariables]);

  // Save variables
  const handleSave = async () => {
    if (!connectionId) return;
    
    // Validate variables first
    if (!validateVariables()) {
      setError('Please fix validation errors before saving');
      return;
    }
    
    // Check max limit
    if (variables.length > 100) {
      setError('Maximum 100 variables allowed');
      return;
    }
    
    setIsSaving(true);
    setError(null);
    setSuccessMessage(null);
    
    try {
      await putEnvironment(connectionId, variables);
      if (isConnectedToThis) {
        setSuccessMessage('Variables saved and updated live.');
      } else {
        setSuccessMessage('Variables saved successfully');
      }
      setVariableErrors({});
      await loadVariables();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save variables');
    } finally {
      setIsSaving(false);
    }
  };

  // Add variable
  const handleAddVariable = () => {
    // Check max limit
    if (variables.length >= 100) {
      setError('Maximum 100 variables allowed');
      return;
    }
    
    const newVariable: Variable = {
      id: crypto.randomUUID(),
      name: '',
      value: '',
      type: 'string',
    };
    setVariables([...variables, newVariable]);
  };

  // Update variable
  const handleUpdateVariable = (id: string, updates: Partial<Variable>) => {
    setVariables(variables.map(v => v.id === id ? { ...v, ...updates } : v));
    // Clear error when user types
    if (variableErrors[id]) {
      setVariableErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[id];
        return newErrors;
      });
    }
  };

  // Remove variable
  const handleRemoveVariable = (id: string) => {
    setVariables(variables.filter(v => v.id !== id));
    setDeleteConfirmId(null);
  };

  if (isLoading) {
    return (
      <div className="settings-section">
        <h3>Environment Variables</h3>
        <p className="section-description">
          Environment variables store reusable values used by automation rules. Use ${'{variable_name}'} syntax in aliases and trigger actions.
        </p>
        <div className="automation-loading">Loading variables...</div>
      </div>
    );
  }

  return (
    <div className="settings-section">
      <h3>Environment Variables</h3>
      <p className="section-description">
        Environment variables store reusable values used by automation rules. Use ${'{variable_name}'} syntax in aliases and trigger actions.
      </p>

      {/* Contextual Guidance */}
      <div className="contextual-help">
        <p className="help-text">
          <strong>Variables can be referenced using ${'{variable_name}'}.</strong>
        </p>
        <p className="help-example">
          Example: <code>cast ${'{target}'}</code> expands to <code>cast goblin</code> when target=goblin
        </p>
      </div>

      {/* Error/Success messages */}
      {error && (
        <div className="message message-error" style={{ marginBottom: '1rem' }}>
          {error}
          <button onClick={() => setError(null)} style={{ marginLeft: '1rem' }}>Dismiss</button>
        </div>
      )}
      
      {successMessage && (
        <div className="message message-success" style={{ marginBottom: '1rem' }}>
          {successMessage}
        </div>
      )}

      <div className="automation-list">
        {variables.map(variable => (
          <div key={variable.id} className="automation-row">
            <div className="automation-row-content">
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input
                    type="text"
                    className={`form-input ${variableErrors[variable.id]?.includes('Name') ? 'form-input-error' : ''}`}
                    placeholder="e.g., target"
                    value={variable.name}
                    onChange={(e) => handleUpdateVariable(variable.id, { name: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">
                    Value
                    <span className="tooltip-icon" title="The value to store. Use ${name} in aliases/triggers to reference this value.">?</span>
                  </label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="e.g., goblin"
                    value={variable.value}
                    onChange={(e) => handleUpdateVariable(variable.id, { value: e.target.value })}
                  />
                  <p className="form-hint">Use ${`{` + variable.name + `}`} in aliases or triggers to use this value</p>
                </div>
                <div className="form-group form-group-small">
                  <label className="form-label">Type</label>
                  <select
                    className="form-input"
                    value={variable.type || 'string'}
                    onChange={(e) => handleUpdateVariable(variable.id, { type: e.target.value as VariableType })}
                  >
                    <option value="string">String</option>
                    <option value="number">Number</option>
                    <option value="boolean">Boolean</option>
                    <option value="array">Array</option>
                  </select>
                </div>
              </div>
              {variableErrors[variable.id] && (
                <div className="form-error">{variableErrors[variable.id]}</div>
              )}
            </div>
            {deleteConfirmId === variable.id ? (
              <div className="delete-confirm">
                <span>Confirm?</span>
                <button
                  className="btn btn-small btn-danger"
                  onClick={() => handleRemoveVariable(variable.id)}
                >
                  Yes
                </button>
                <button
                  className="btn btn-small"
                  onClick={() => setDeleteConfirmId(null)}
                >
                  No
                </button>
              </div>
            ) : (
              <button
                className="btn btn-small btn-danger"
                onClick={() => setDeleteConfirmId(variable.id)}
              >
                Remove
              </button>
            )}
          </div>
        ))}
        {variables.length === 0 && (
          <div className="automation-empty">
            No variables configured
          </div>
        )}
      </div>

      <div className="settings-actions">
        <button
          className="btn btn-secondary"
          onClick={handleAddVariable}
        >
          Add Variable
        </button>
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={isSaving}
        >
          {isSaving ? 'Saving...' : 'Save Variables'}
        </button>
      </div>
    </div>
  );
}
