import type React from 'react'
import { useFirebaseTotpAuth } from '../service/auth/useFirebaseTotpAuth'
import { useSetupTotpState } from '../service/auth/useSetupTotpState'
import TotpQrCode from '../components/TotpQrCode'
import './pages.css'

type SetupTotpProps = {
  auth: ReturnType<typeof useFirebaseTotpAuth>
}

export const SetupTotp: React.FC<SetupTotpProps> = ({ auth }) => {
  const state = useSetupTotpState(auth)

  return (
    <div className="setup-page-container">
      <div className="setup-card">
        {/* Subtle decorative glow */}
        <div className="setup-glow-emerald" />
        <div className="setup-glow-orange" />

        <div className="setup-header">
          <span className="setup-eyebrow">Support Copilot</span>
          <h1 className="setup-title">Setup TOTP</h1>
        </div>

        {state.needsTotpEnrollment ? (
          <div className="setup-body">
            <p className="setup-text">
              Scan this QR code in your authenticator app (e.g. Google Authenticator or 1Password) to begin:
            </p>

            <div className="setup-qr-container">
              <TotpQrCode otpauthUrl={state.enrollOtpAuthUrl} />
            </div>

            <div className="setup-manual-group">
              <span className="setup-manual-label">Or Setup Key Manually:</span>
              <input
                type="text"
                value={state.enrollOtpAuthUrl}
                readOnly
                className="setup-manual-input"
                onClick={(e) => (e.target as HTMLInputElement).select()}
              />
            </div>

            <form onSubmit={(e) => void state.handleSubmit(e)} className="setup-form">
              <div className="setup-form-group">
                <label htmlFor="totp-code" className="setup-label">
                  Enter TOTP
                </label>
                <div className="setup-input-row">
                  <input
                    id="totp-code"
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    maxLength={6}
                    value={state.enrollCode}
                    onChange={(e) => state.handleCodeChange(e.target.value)}
                    placeholder="123456"
                    disabled={state.isBusy}
                    className="setup-input"
                    required
                  />
                  <button
                    type="submit"
                    disabled={state.isBusy || !state.enrollCode.trim()}
                    className="setup-btn-submit"
                  >
                    {state.isBusy ? 'Verifying...' : 'Submit'}
                  </button>
                </div>
                {state.codeError && (
                  <p className="setup-error-msg">{state.codeError}</p>
                )}
              </div>
            </form>
          </div>
        ) : !state.isEmailVerified ? (
          <div className="setup-body text-center space-y-4">
            <p className="setup-text">
              Please verify your email address to enable TOTP setup. Once verified, click <strong>"Check Status"</strong> below.
            </p>
            <div className="flex flex-col sm:flex-row gap-3 justify-center pt-2">
              <button
                type="button"
                onClick={() => void state.checkVerificationStatus()}
                disabled={state.isBusy}
                className="setup-btn-submit"
              >
                {state.isBusy ? 'Checking...' : 'Check Status'}
              </button>
              <button
                type="button"
                onClick={() => void state.resendVerification()}
                disabled={state.isBusy}
                className="px-4 py-2 rounded-xl text-xs font-semibold bg-muted hover:bg-muted/80 text-foreground border border-border transition-colors cursor-pointer"
              >
                Resend Email
              </button>
            </div>
          </div>
        ) : (
          <div className="setup-initializing-container">
            <div className="setup-initializing-text">Initializing TOTP generator...</div>
          </div>
        )}

        <div className="setup-footer-actions">
          {state.submitError && (
            <p className="setup-error">{state.submitError}</p>
          )}
          {state.authStatus && !state.submitError && (
            <p className="setup-status">{state.authStatus}</p>
          )}

          <button
            type="button"
            onClick={() => void state.handleCancel()}
            disabled={state.isBusy}
            className="setup-btn-cancel"
          >
            Cancel & Sign Out
          </button>
        </div>
      </div>
    </div>
  )
}

export default SetupTotp