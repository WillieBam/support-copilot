import type React from 'react'
import { Link } from 'react-router-dom'
import { useFirebaseTotpAuth } from '../service/auth/useFirebaseTotpAuth'
import { useLoginState } from '../service/auth/useLoginState'
import './pages.css'

type LoginPageProps = {
  auth: ReturnType<typeof useFirebaseTotpAuth>
}

export const LoginPage: React.FC<LoginPageProps> = ({ auth }) => {
  const state = useLoginState(auth)

  if (state.isSignedIn && !state.isEmailVerified && state.authMode === 'firebase') {
    return (
      <div className="login-page-container">
        <div className="login-card">
          {/* Subtle decorative glow */}
          <div className="login-glow-emerald" />
          <div className="login-glow-orange" />

          <div className="login-header">
            <span className="login-eyebrow">Support Copilot</span>
            <h1 className="login-title">Verify Email</h1>
            <p className="login-desc">
              We sent a verification email to:
            </p>
            <div className="my-2 inline-flex items-center justify-center px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 font-semibold text-xs tracking-wide mx-auto">
              {state.email || 'Your Registered Email'}
            </div>
            <p className="login-subdesc">
              Please check your inbox and spam folder, then verify your email before continuing.
            </p>
          </div>

          <div className="login-actions">
            <button
              type="button"
              onClick={() => void state.checkVerificationStatus()}
              disabled={state.isBusy}
              className="login-btn-emerald"
            >
              {state.isBusy ? 'Checking...' : 'Check Status'}
            </button>

            <button
              type="button"
              onClick={() => void state.resendVerification()}
              disabled={state.isBusy || state.resendCooldown > 0}
              className="login-btn-outline disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {state.resendCooldown > 0
                ? `Resend Email (${state.resendCooldown}s)`
                : 'Resend Verification Email'}
            </button>

            <button
              type="button"
              onClick={() => void state.signOut()}
              disabled={state.isBusy}
              className="login-btn-danger"
            >
              Sign Out
            </button>
          </div>

          <p className="text-[11px] text-center text-muted-foreground/80 italic mt-1">
            💡 Returning to this tab will auto-check your email verification status.
          </p>

          {state.submitError && (
            <p className="login-error">{state.submitError}</p>
          )}
          {state.authStatus && (
            <p className="login-status">{state.authStatus}</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="login-page-container">
      <div className="login-card">
        {/* Subtle decorative glow */}
        <div className="login-glow-emerald" />
        <div className="login-glow-orange" />

        <div className="login-header-simple">
          <span className="login-eyebrow">Support Copilot</span>
          <h1 className="login-title">Login</h1>
        </div>

        {/* Auth Mode Switcher */}
        <div className="flex bg-muted/40 p-1 rounded-xl border border-border/60 relative z-10">
          <button
            type="button"
            onClick={() => state.handleModeChange('backend')}
            className={`flex-1 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
              state.authMode === 'backend'
                ? 'bg-card text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Direct Account
          </button>
          <button
            type="button"
            onClick={() => state.handleModeChange('firebase')}
            className={`flex-1 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
              state.authMode === 'firebase'
                ? 'bg-card text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            Firebase Auth
          </button>
        </div>

        <form onSubmit={(e) => void state.handleSubmit(e)} className="login-form">
          <div className="login-form-group">
            <label className="login-label">
              {state.authMode === 'backend' ? 'Username or Email' : 'Email'}
            </label>
            <input
              type={state.authMode === 'backend' ? 'text' : 'email'}
              value={state.identifier}
              onChange={(e) => state.handleIdentifierChange(e.target.value)}
              placeholder={state.authMode === 'backend' ? 'johndoe or name@example.com' : 'name@example.com'}
              disabled={state.isBusy}
              className="login-input"
              required
            />
            {state.identifierError && (
              <p className="login-input-error">{state.identifierError}</p>
            )}
          </div>

          <div className="login-form-group">
            <label className="login-label">Password</label>
            <input
              type="password"
              value={state.password}
              onChange={(e) => state.handlePasswordChange(e.target.value)}
              placeholder="••••••••"
              disabled={state.isBusy}
              className="login-input"
              required
            />
            {state.passwordError && (
              <p className="login-input-error">{state.passwordError}</p>
            )}
          </div>

          <div className="login-submit-row">
            <button
              type="submit"
              disabled={state.isBusy}
              className="login-btn-submit"
            >
              {state.isBusy ? 'Signing In...' : state.authMode === 'backend' ? 'Sign In' : 'Sign In with Firebase'}
            </button>
          </div>
        </form>

        {state.submitError && (
          <p className="login-error">{state.submitError}</p>
        )}
        {state.authStatus && !state.submitError && (
          <p className="login-status">{state.authStatus}</p>
        )}

        <div className="login-footer">
          First Time Here?{' '}
          <Link to="/register" className="login-link">
            Create an Account
          </Link>
        </div>
      </div>
    </div>
  )
}

export default LoginPage