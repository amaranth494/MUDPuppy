import { useState, useEffect, useCallback } from 'react';
import { AISettingsResponse, PolicyResponse } from '../types';
import { getPolicy, acceptPolicy, getAISettings, putAISettings, deleteCapturedText } from '../services/api';

interface AIPlayerPanelProps {
  connectionId: string;
}

export default function AIPlayerPanel({ connectionId }: AIPlayerPanelProps) {
  const [policy, setPolicy] = useState<PolicyResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAccepting, setIsAccepting] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // AI settings editor fields — the two numeric fields bind to string state so
  // an empty input can round-trip as JSON null (blank is a first-class value,
  // never silently coerced to a number in the browser).
  const [conductRules, setConductRules] = useState('');
  const [approachGuidance, setApproachGuidance] = useState('');
  const [neverIssueList, setNeverIssueList] = useState('');
  const [modelName, setModelName] = useState('');
  const [callCapStr, setCallCapStr] = useState('');
  const [disengageThresholdStr, setDisengageThresholdStr] = useState('');

  // The captured-text danger zone (D-21) — its own two-step inline
  // confirmation, separate from the settings-save state above so a save in
  // flight can never be confused with a delete in flight.
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const applyAISettings = (data: AISettingsResponse) => {
    setConductRules(data.conduct_rules);
    setApproachGuidance(data.approach_guidance);
    setNeverIssueList(data.never_issue_list);
    setModelName(data.ai_settings.model_name);
    setCallCapStr(data.ai_settings.call_cap === null ? '' : String(data.ai_settings.call_cap));
    setDisengageThresholdStr(
      data.ai_settings.disengage_threshold === null ? '' : String(data.ai_settings.disengage_threshold)
    );
  };

  const loadPanel = useCallback(async () => {
    if (!connectionId) return;
    setIsLoading(true);
    setError(null);
    try {
      const policyData = await getPolicy(connectionId);
      setPolicy(policyData);
      if (policyData.accepted) {
        const settingsData = await getAISettings(connectionId);
        applyAISettings(settingsData);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load policy');
    } finally {
      setIsLoading(false);
    }
  }, [connectionId]);

  useEffect(() => {
    loadPanel();
  }, [loadPanel]);

  const handleAccept = async () => {
    if (!connectionId) return;
    setIsAccepting(true);
    setError(null);
    try {
      const policyData = await acceptPolicy(connectionId);
      setPolicy(policyData);
      const settingsData = await getAISettings(connectionId);
      applyAISettings(settingsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to record acceptance');
    } finally {
      setIsAccepting(false);
    }
  };

  const handleSave = async () => {
    if (!connectionId) return;
    setIsSaving(true);
    setError(null);
    setSuccessMessage(null);
    try {
      const body: AISettingsResponse = {
        conduct_rules: conductRules,
        approach_guidance: approachGuidance,
        never_issue_list: neverIssueList,
        ai_settings: {
          model_name: modelName,
          call_cap: callCapStr.trim() === '' ? null : Number(callCapStr),
          disengage_threshold: disengageThresholdStr.trim() === '' ? null : Number(disengageThresholdStr),
        },
      };
      const updated = await putAISettings(connectionId, body);
      applyAISettings(updated);
      setSuccessMessage('AI settings saved successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save AI settings');
    } finally {
      setIsSaving(false);
    }
  };

  // The captured-text danger zone's confirm step (D-21): the Yes button.
  // Failure shows the locked wording, not the raw error, per 04-UI-SPEC.md.
  const handleDeleteCapturedText = async () => {
    if (!connectionId) return;
    setIsDeleting(true);
    setError(null);
    setSuccessMessage(null);
    try {
      await deleteCapturedText(connectionId);
      setSuccessMessage('Captured text deleted.');
      setConfirmingDelete(false);
    } catch {
      setError('Failed to delete captured text — refresh the page to try again');
    } finally {
      setIsDeleting(false);
    }
  };

  // Inline formatting (bold only — the policy text's actual markdown surface),
  // themed with the app's custom properties rather than HelpPage.tsx's hex colors.
  const renderInline = (text: string) => {
    const parts = text.split(/(\*\*[^*]+\*\*)/g);
    return parts.map((part, idx) => {
      if (part.startsWith('**') && part.endsWith('**')) {
        return (
          <strong key={idx} style={{ fontWeight: 700 }}>
            {part.slice(2, -2)}
          </strong>
        );
      }
      return <span key={idx}>{part}</span>;
    });
  };

  // Paragraph/bullet-list splitting, adapted from HelpPage.tsx's renderContent
  // shape but re-themed to this file's custom properties.
  const renderContent = (content: string) => {
    const paragraphs = content.split('\n\n');
    return paragraphs.map((para, idx) => {
      if (para.includes('\n- ') || para.startsWith('- ')) {
        const items = para.split('\n').map((line) => line.replace(/^-\s*/, ''));
        return (
          <ul key={idx} style={{ marginBottom: '1rem', paddingLeft: '1.5rem', color: 'var(--color-text-dim)' }}>
            {items.map((item, i) => (
              <li key={i} style={{ marginBottom: '0.25rem' }}>
                {renderInline(item)}
              </li>
            ))}
          </ul>
        );
      }
      return (
        <p key={idx} style={{ marginBottom: '1rem', color: 'var(--color-text-dim)', lineHeight: 1.6 }}>
          {renderInline(para)}
        </p>
      );
    });
  };

  if (isLoading) {
    return (
      <div className="settings-section">
        <h3>AI Player</h3>
        <div className="automation-loading">Loading AI Player settings...</div>
      </div>
    );
  }

  return (
    <div className="settings-section">
      <h3>AI Player</h3>

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

      {policy && !policy.accepted && (
        <>
          <div className="policy-panel">
            <h4
              style={{
                fontSize: 'var(--font-size-lg)',
                fontWeight: 700,
                marginBottom: 'var(--spacing-xs)',
                color: 'var(--color-text)',
              }}
            >
              Safety and Abuse Policy
            </h4>
            <p
              style={{
                fontSize: 'var(--font-size-sm)',
                color: 'var(--color-text-dim)',
                marginBottom: 'var(--spacing-md)',
              }}
            >
              Version {policy.version}
            </p>
            {renderContent(policy.text)}
          </div>
          <div className="settings-actions">
            <button className="btn btn-primary" onClick={handleAccept} disabled={isAccepting}>
              {isAccepting ? 'Accepting...' : 'Accept Policy'}
            </button>
          </div>
        </>
      )}

      {policy && policy.accepted && (
        <>
          <p className="profile-hint ai-player-accepted-line">
            ✓ Accepted v{policy.accepted_version} on {policy.accepted_at ? policy.accepted_at.slice(0, 10) : ''}
          </p>

          <div className="form-group">
            <label className="form-label">Conduct Rules</label>
            <textarea
              className="form-input form-textarea"
              style={{ minHeight: '120px' }}
              value={conductRules}
              onChange={(e) => setConductRules(e.target.value)}
            />
            <p className="form-hint">Plain-text rules handed to the model as-is. Blank means none.</p>
          </div>

          <div className="form-group">
            <label className="form-label">Approach Guidance</label>
            <textarea
              className="form-input form-textarea"
              style={{ minHeight: '120px' }}
              value={approachGuidance}
              onChange={(e) => setApproachGuidance(e.target.value)}
            />
            <p className="form-hint">Free-text guidance on how the AI should play. Blank means none.</p>
          </div>

          <div className="form-group">
            <label className="form-label">Never-Issue List</label>
            <textarea
              className="form-input form-textarea"
              style={{ minHeight: '120px' }}
              value={neverIssueList}
              onChange={(e) => setNeverIssueList(e.target.value)}
            />
            <p className="form-hint">
              One entry per line. A command is blocked if it starts with an entry, matched
              whole-word and case-insensitive — so &quot;give&quot; blocks &quot;give sword to bob&quot;
              but not &quot;giveaway&quot;. Blank means none: nothing is blocked by this list, though
              the reviewer and the model&apos;s own judgment still apply.
            </p>
          </div>

          <div className="form-group">
            <label className="form-label">Model Name</label>
            <input
              type="text"
              className="form-input"
              placeholder="Server default"
              value={modelName}
              onChange={(e) => setModelName(e.target.value)}
            />
            <p className="form-hint">Blank uses the server's configured default model.</p>
          </div>

          <div className="form-group">
            <label className="form-label">Call Cap (per session)</label>
            <input
              type="number"
              className="form-input"
              placeholder="No cap"
              value={callCapStr}
              onChange={(e) => setCallCapStr(e.target.value)}
            />
            <p className="form-hint">Blank means no cap on calls per session.</p>
          </div>

          <div className="form-group">
            <label className="form-label">Disengage Threshold</label>
            <input
              type="number"
              className="form-input"
              placeholder="Default"
              value={disengageThresholdStr}
              onChange={(e) => setDisengageThresholdStr(e.target.value)}
            />
            <p className="form-hint">
              Blank uses the engine's built-in error handling: any AI failure shows an informative error,
              disengages, and regular play continues. It never crashes.
            </p>
          </div>

          <div className="settings-actions">
            <button className="btn btn-primary" onClick={handleSave} disabled={isSaving}>
              {isSaving ? 'Saving...' : 'Save AI Settings'}
            </button>
          </div>

          <div className="ai-player-danger-zone">
            <p className="form-hint">
              This permanently deletes captured game-text snapshots and session transcripts for this profile. Decision rows, reasoning, and outcomes stay on record. This cannot be undone.
            </p>
            {confirmingDelete ? (
              <div className="delete-confirm">
                <span>Delete captured text now?</span>
                <button className="btn btn-sm btn-danger" onClick={handleDeleteCapturedText} disabled={isDeleting}>
                  Yes
                </button>
                <button className="btn btn-sm" onClick={() => setConfirmingDelete(false)} disabled={isDeleting}>
                  No
                </button>
              </div>
            ) : (
              <button className="btn btn-sm btn-secondary" onClick={() => setConfirmingDelete(true)}>
                Delete Captured Text Now
              </button>
            )}
          </div>
        </>
      )}
    </div>
  );
}
