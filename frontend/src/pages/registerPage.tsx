import type React from 'react'
import { Link } from 'react-router-dom'
import { useFirebaseTotpAuth } from '../service/auth/useFirebaseTotpAuth'
import { useRegisterState } from '../service/auth/useRegisterState'
import './pages.css'

type RegisterPageProps = {
  auth: ReturnType<typeof useFirebaseTotpAuth>
}

export const RegisterPage: React.FC<RegisterPageProps> = ({ auth }) => {
  const state = useRegisterState(auth)

  return (
    <div className="register-page-container">
      <div className="register-card">
        {/* Subtle decorative glow */}
        <div className="register-glow-emerald" />
        <div className="register-glow-orange" />

        <div className="register-header">
          <span className="register-eyebrow">Support Copilot</span>
          <h1 className="register-title">Create Account</h1>
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

        {state.registrationSuccess ? (
          <div className="flex flex-col gap-4 text-center relative z-10 py-4">
            <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-emerald-500/10 text-emerald-400 mx-auto text-xl font-bold">
              ✓
            </div>
            <div>
              <h2 className="text-base font-semibold text-foreground">Account Created!</h2>
              <p className="text-xs text-muted-foreground mt-1">
                Your account ({state.username || state.email}) has been registered successfully.
              </p>
            </div>
            <Link
              to="/login"
              className="register-btn-emerald inline-block py-2.5 text-center text-sm font-semibold rounded-[20px] mt-2"
            >
              Proceed to Login
            </Link>
          </div>
        ) : (
          <form onSubmit={(e) => void state.handleSubmit(e)} className="register-form">
            {state.authMode === 'backend' && (
              <div className="register-form-group">
                <label className="register-label">Username</label>
                <input
                  type="text"
                  value={state.username}
                  onChange={(e) => state.handleUsernameChange(e.target.value)}
                  placeholder="johndoe"
                  disabled={state.isBusy}
                  className="register-input"
                  required
                />
                {state.usernameError && (
                  <p className="register-input-error">{state.usernameError}</p>
                )}
              </div>
            )}

            <div className="register-form-group">
              <label className="register-label">Email</label>
              <input
                type="email"
                value={state.email}
                onChange={(e) => state.handleEmailChange(e.target.value)}
                placeholder="name@example.com"
                disabled={state.isBusy}
                className="register-input"
                required
              />
              {state.emailError && (
                <p className="register-input-error">{state.emailError}</p>
              )}
            </div>

            <div className="register-form-group">
              <label className="register-label">Password</label>
              <input
                type="password"
                value={state.password}
                onChange={(e) => state.handlePasswordChange(e.target.value)}
                placeholder="••••••••"
                disabled={state.isBusy}
                className="register-input"
                required
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                Must be 6–8 characters long with at least 1 special character (e.g. !@#$%^&*)
              </p>
              {state.passwordError && (
                <p className="register-input-error">{state.passwordError}</p>
              )}
            </div>

            <div className="register-submit-row">
              <button
                type="submit"
                disabled={state.isBusy}
                className="register-btn-submit"
              >
                {state.isBusy ? 'Creating...' : state.authMode === 'backend' ? 'Register Account' : 'Register with Firebase'}
              </button>
            </div>
          </form>
        )}

        {state.submitError && (
          <p className="register-error">{state.submitError}</p>
        )}
        {state.authStatus && !state.submitError && !state.registrationSuccess && (
          <p className="register-status">{state.authStatus}</p>
        )}

        <div className="register-footer">
          Already have an account?{' '}
          <Link to="/login" className="register-link">
            Login here
          </Link>
        </div>
      </div>
    </div>
  )
}

export default RegisterPage