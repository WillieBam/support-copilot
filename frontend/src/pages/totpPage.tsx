import type React from 'react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useFirebaseTotpAuth } from '../service/auth/useFirebaseTotpAuth'
import { useTotpPageState } from '../service/auth/useTotpPageState'
import './pages.css'

type TotpPageProps = {
  auth: ReturnType<typeof useFirebaseTotpAuth>
}

export const TotpPage: React.FC<TotpPageProps> = ({ auth }) => {
  const state = useTotpPageState(auth)
  const [success, setSuccess] = useState(false)
  const navigate = useNavigate()


  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await state.handleSubmit(e)

    // If verification succeeded, auth.authError is cleared and token is present
    if (!auth.authError && auth.isSignedIn) {
      setSuccess(true)
      setTimeout(() => {
        navigate('/chat', { replace: true })
      }, 1200)
    }
  }

  return (
    <div className="totp-page-container">
      <div className="totp-card">
        {/* Subtle decorative glow */}
        <div className="totp-glow-emerald" />
        <div className="totp-glow-orange" />

        <div className="totp-header">
          <span className="totp-eyebrow">Support Copilot</span>
          <h1 className="totp-title">Enter Totp</h1>
        </div>

        {success ? (
          <div className="totp-success-banner">
            ✓ login successfully
          </div>
        ) : (
          <form onSubmit={handleFormSubmit} className="totp-form">
            <div className="totp-form-group">
              {/* <label htmlFor="totp-input" className="totp-label">
                Enter your 6-digit TOTP code
              </label> */}
              <div className="totp-input-row">
                <input
                  id="totp-input"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  maxLength={6}
                  value={state.totpCode}
                  onChange={(e) => state.handleCodeChange(e.target.value)}
                  placeholder="123456"
                  disabled={state.isBusy}
                  className="totp-input"
                  required
                />
                <button
                  type="submit"
                  disabled={state.isBusy || !state.totpCode.trim()}
                  className="totp-btn-verify"
                >
                  {state.isBusy ? 'Verifying...' : 'Verify'}
                </button>
              </div>
              {state.codeError && (
                <p className="totp-error">{state.codeError}</p>
              )}
              {state.submitError && (
                <p className="totp-error">{state.submitError}</p>
              )}
            </div>
          </form>
        )}

        <div className="totp-footer-actions">
          {state.authStatus && !state.submitError && !success && (
            <p className="totp-status">{state.authStatus}</p>
          )}

          <button
            type="button"
            onClick={() => void state.handleCancel()}
            disabled={state.isBusy}
            className="totp-btn-cancel"
          >
            Cancel & Sign Out
          </button>
        </div>
      </div>
    </div>
  )
}

export default TotpPage