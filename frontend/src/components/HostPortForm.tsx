import { useState, useEffect } from 'react';

interface HostPortFormProps {
  host: string;
  port: number;
  onHostChange: (host: string) => void;
  onPortChange: (port: number) => void;
  disabled?: boolean;
  autoFocus?: boolean;
  onSubmit?: () => void;
}

export default function HostPortForm({
  host,
  port,
  onHostChange,
  onPortChange,
  disabled = false,
  autoFocus = false,
  onSubmit,
}: HostPortFormProps) {
  const [localHost, setLocalHost] = useState(host);
  const [localPort, setLocalPort] = useState(port);

  // Sync with props
  useEffect(() => {
    setLocalHost(host);
  }, [host]);

  useEffect(() => {
    setLocalPort(port);
  }, [port]);

  const handleHostChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setLocalHost(value);
    onHostChange(value);
  };

  const handlePortChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value) || 23;
    setLocalPort(value);
    onPortChange(value);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && onSubmit) {
      e.preventDefault();
      onSubmit();
    }
  };

  return (
    <div className="host-port-form">
      <div className="form-group">
        <label className="form-label" htmlFor="host-port-host">
          Host
        </label>
        <input
          id="host-port-host"
          type="text"
          className="form-input"
          placeholder="e.g., example.com"
          value={localHost}
          onChange={handleHostChange}
          disabled={disabled}
          autoFocus={autoFocus}
          onKeyDown={handleKeyDown}
        />
      </div>

      <div className="form-group">
        <label className="form-label" htmlFor="host-port-port">
          Port
        </label>
        <input
          id="host-port-port"
          type="number"
          className="form-input"
          placeholder="23"
          value={localPort}
          onChange={handlePortChange}
          disabled={disabled}
          onKeyDown={handleKeyDown}
        />
      </div>
    </div>
  );
}
