import Modal from './Modal';
import { useSession } from '../context/SessionContext';

interface ReconnectModalProps {
  isOpen: boolean;
  onInputLockChange?: (locked: boolean) => void;
}

/**
 * Reconnection modal for MVP
 * 
 * Shows when user tries to connect but already has an active session.
 * Offers options to reconnect (disconnect old session first) or disconnect (cancel).
 */
export default function ReconnectModal({ isOpen, onInputLockChange }: ReconnectModalProps) {
  const { 
    pendingReconnectData, 
    clearPendingReconnect, 
    forceReconnect,
    host 
  } = useSession();

  const mudName = host || pendingReconnectData?.host || 'the MUD server';

  const handleReconnect = async () => {
    await forceReconnect();
  };

  const handleDisconnect = () => {
    clearPendingReconnect();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleDisconnect}
      title="Active Connection Detected"
      onInputLockChange={onInputLockChange}
    >
      <div style={{ textAlign: 'center', padding: '20px' }}>
        <p style={{ marginBottom: '20px', fontSize: '16px' }}>
          A previous active connection was detected. Would you like to reconnect to {mudName}?
        </p>
        <div style={{ display: 'flex', gap: '15px', justifyContent: 'center' }}>
          <button
            onClick={handleReconnect}
            style={{
              padding: '12px 24px',
              fontSize: '16px',
              backgroundColor: '#4CAF50',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            RECONNECT
          </button>
          <button
            onClick={handleDisconnect}
            style={{
              padding: '12px 24px',
              fontSize: '16px',
              backgroundColor: '#f44336',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            DISCONNECT
          </button>
        </div>
      </div>
    </Modal>
  );
}
