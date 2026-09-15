import { useSession } from '../context/SessionContext';

/**
 * AutopilotBadge Component - states the switch position (D-10).
 *
 * Reads autopilotState straight from SessionContext, which is kept fresh by
 * SessionBadge.tsx's existing 15s interval / visibility-change / on-mount refresh of the
 * same context, plus the live websocket push wired in SessionContext.tsx. This component
 * runs no interval and no visibility listener of its own -- a second poll here would be
 * duplicated work.
 *
 * Derives nothing: it does not read the connection's own status field and holds no local
 * component state about the switch. The server's value is the only input, which is what
 * makes the badge trustworthy after a refresh.
 */
export default function AutopilotBadge() {
  const { autopilotState } = useSession();

  const getStateSuffix = () => {
    switch (autopilotState) {
      case 'on':
        return 'on';
      case 'waiting':
        return 'waiting';
      case 'off':
        return 'off';
      default:
        return 'off';
    }
  };

  const getLabel = () => {
    switch (autopilotState) {
      case 'on':
        return 'On';
      case 'waiting':
        return 'Waiting';
      case 'off':
        return 'Off';
      default:
        return 'Off';
    }
  };

  const stateSuffix = getStateSuffix();
  const label = getLabel();

  return (
    <div className={`autopilot-badge state-${stateSuffix}`} title={`Autopilot: ${label}`}>
      <span className="autopilot-badge-dot" />
      <span className="autopilot-badge-text">Autopilot: {label}</span>
    </div>
  );
}
