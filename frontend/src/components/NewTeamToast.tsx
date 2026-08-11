import { useEffect, useState } from 'react';
import { Users, X } from 'lucide-react';

interface NewTeamToastProps {
  teamNames: string[];
  onDismiss: () => void;
}

/**
 * Slides in from top-right when the user has been added to new teams.
 * Auto-dismisses after 6 s; also dismissable manually.
 */
export function NewTeamToast({ teamNames, onDismiss }: NewTeamToastProps) {
  const [visible, setVisible] = useState(false);

  // Animate in on mount
  useEffect(() => {
    const id = requestAnimationFrame(() => setVisible(true));
    return () => cancelAnimationFrame(id);
  }, []);

  // Auto-dismiss after 6 s
  useEffect(() => {
    const timer = setTimeout(() => handleDismiss(), 6000);
    return () => clearTimeout(timer);
  }, []);

  const handleDismiss = () => {
    setVisible(false);
    setTimeout(onDismiss, 350);
  };

  if (teamNames.length === 0) return null;

  return (
    <div
      style={{
        position: 'fixed',
        top: '88px',
        right: '24px',
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        transition: 'transform 0.35s cubic-bezier(0.16,1,0.3,1), opacity 0.35s ease',
        transform: visible ? 'translateX(0)' : 'translateX(calc(100% + 32px))',
        opacity: visible ? 1 : 0,
        pointerEvents: visible ? 'auto' : 'none',
      }}
    >
      {teamNames.map((name) => (
        <div
          key={name}
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '12px',
            padding: '14px 16px',
            minWidth: '300px',
            maxWidth: '360px',
            borderRadius: '16px',
            border: '1px solid rgba(16, 185, 129, 0.3)',
            background: 'color-mix(in oklch, var(--card) 85%, transparent)',
            backdropFilter: 'blur(20px)',
            boxShadow: '0 8px 32px -4px rgba(0,0,0,0.18), 0 0 0 1px rgba(16,185,129,0.1)',
          }}
        >
          <div
            style={{
              flexShrink: 0,
              width: '36px',
              height: '36px',
              borderRadius: '10px',
              background: 'rgba(16, 185, 129, 0.12)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Users style={{ width: '18px', height: '18px', color: '#10b981' }} />
          </div>

          <div style={{ flex: 1, minWidth: 0 }}>
            <p
              style={{
                margin: 0,
                fontSize: '13px',
                fontWeight: 600,
                color: 'var(--foreground)',
                lineHeight: 1.3,
              }}
            >
              Added to a new team
            </p>
            <p
              style={{
                margin: '3px 0 0',
                fontSize: '12px',
                color: 'var(--muted-foreground)',
                lineHeight: 1.4,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              You've been added to{' '}
              <span style={{ color: '#10b981', fontWeight: 600 }}>{name}</span>
            </p>
          </div>

          <button
            onClick={handleDismiss}
            style={{
              flexShrink: 0,
              background: 'transparent',
              border: 'none',
              padding: '2px',
              cursor: 'pointer',
              color: 'var(--muted-foreground)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: '6px',
              transition: 'color 0.15s',
            }}
            onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--foreground)')}
            onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--muted-foreground)')}
            aria-label="Dismiss notification"
          >
            <X style={{ width: '14px', height: '14px' }} />
          </button>
        </div>
      ))}
    </div>
  );
}
