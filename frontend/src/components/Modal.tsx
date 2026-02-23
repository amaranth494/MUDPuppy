import { useEffect, useRef, useCallback, ReactNode } from 'react';

interface ModalProps {
  /** Whether the modal is open */
  isOpen: boolean;
  /** Callback when modal should close */
  onClose: () => void;
  /** Modal title for accessibility */
  title?: string;
  /** Whether clicking the backdrop should close the modal (default: false) */
  closeOnBackdropClick?: boolean;
  /** Child content to render inside the modal */
  children: ReactNode;
}

/**
 * Reusable Modal component for SP03PH03
 * 
 * Features:
 * - Full-height overlay
 * - Close button
 * - ESC key support to close
 * - Focus management (trap focus, restore on close)
 * - Accessibility (role="dialog", aria-modal="true")
 * 
 * Note: Terminal rendering and text selection continue to work while modal is open.
 * Input lock is handled separately via SessionContext.
 */
export default function Modal({
  isOpen,
  onClose,
  title,
  closeOnBackdropClick = false,
  children,
}: ModalProps) {
  const modalRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  // Save the currently focused element before modal opens
  useEffect(() => {
    if (isOpen) {
      // Save reference to previously focused element
      previousFocusRef.current = document.activeElement as HTMLElement;
      
      // Focus the modal container
      setTimeout(() => {
        if (modalRef.current) {
          modalRef.current.focus();
        }
      }, 0);
    }
  }, [isOpen]);

  // Handle ESC key to close modal
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, onClose]);

  // Handle backdrop click
  const handleBackdropClick = useCallback(() => {
    if (closeOnBackdropClick) {
      onClose();
    }
  }, [closeOnBackdropClick, onClose]);

  // Restore focus to previously focused element on close
  useEffect(() => {
    if (!isOpen && previousFocusRef.current) {
      previousFocusRef.current.focus();
      previousFocusRef.current = null;
    }
  }, [isOpen]);

  // Trap focus inside modal
  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (!modalRef.current) return;

    // Get all focusable elements inside modal
    const focusableElements = modalRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    // Handle Tab key
    if (event.key === 'Tab') {
      if (event.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstElement) {
          event.preventDefault();
          lastElement?.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          event.preventDefault();
          firstElement?.focus();
        }
      }
    }
  }, []);

  if (!isOpen) return null;

  return (
    <div 
      className="modal-overlay"
      onClick={handleBackdropClick}
      role="presentation"
    >
      <div
        ref={modalRef}
        className="modal-container"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'modal-title' : undefined}
        tabIndex={-1}
        onKeyDown={handleKeyDown}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          className="modal-close"
          onClick={onClose}
          aria-label="Close modal"
          type="button"
        >
          ✕
        </button>

        {/* Modal title (hidden visually but available for screen readers) */}
        {title && (
          <h2 id="modal-title" className="modal-title">
            {title}
          </h2>
        )}

        {/* Modal content */}
        <div className="modal-content">
          {children}
        </div>
      </div>
    </div>
  );
}
