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

        <form onSubmit={(e) => void state.handleSubmit(e)} className="register-form">
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
              {state.isBusy ? 'Creating...' : 'Register'}
            </button>
          </div>
        </form>

        {state.submitError && (
          <p className="register-error">{state.submitError}</p>
        )}
        {state.authStatus && !state.submitError && (
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