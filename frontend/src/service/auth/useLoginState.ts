import { useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useLoginState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [authMode, setAuthMode] = useState<'backend' | 'firebase'>('backend')
  const [identifierError, setIdentifierError] = useState('')
  const [passwordError, setPasswordError] = useState('')

  const handleIdentifierChange = (val: string) => {
    setIdentifier(val)
    auth.setLoginIdentifier(val)
    auth.setLoginEmail(val)
    if (identifierError) setIdentifierError('')
  }

  const handlePasswordChange = (val: string) => {
    setPassword(val)
    auth.setPassword(val)
    if (passwordError) setPasswordError('')
  }

  const handleModeChange = (mode: 'backend' | 'firebase') => {
    setAuthMode(mode)
    auth.setAuthProvider(mode)
    setIdentifierError('')
    setPasswordError('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIdentifierError('')
    setPasswordError('')

    let hasError = false
    const trimmedId = identifier.trim()

    if (!trimmedId) {
      setIdentifierError(authMode === 'backend' ? 'Username or email is required' : 'Email is required')
      hasError = true
    } else if (authMode === 'firebase' && !/\S+@\S+\.\S+/.test(trimmedId)) {
      setIdentifierError('Invalid email format')
      hasError = true
    }

    if (!password) {
      setPasswordError('Password is required')
      hasError = true
    } else if (password.length < 6) {
      setPasswordError('Password must be at least 6 characters')
      hasError = true
    }

    if (hasError) return

    if (authMode === 'backend') {
      await auth.signInBackend(trimmedId, password)
    } else {
      await auth.signIn()
    }
  }

  return {
    identifier,
    password,
    authMode,
    identifierError,
    passwordError,
    email: identifier,
    emailError: identifierError,
    submitError: auth.authError,
    authStatus: auth.authStatus,
    isBusy: auth.isBusy,
    isSignedIn: auth.isSignedIn,
    isEmailVerified: auth.isEmailVerified,
    resendCooldown: auth.resendCooldown,
    handleIdentifierChange,
    handleEmailChange: handleIdentifierChange,
    handlePasswordChange,
    handleModeChange,
    handleSubmit,
    checkVerificationStatus: auth.checkVerificationStatus,
    resendVerification: auth.resendVerification,
    signOut: auth.signOut,
  }
}
