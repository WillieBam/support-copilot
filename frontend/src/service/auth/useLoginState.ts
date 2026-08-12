import { useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useLoginState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
  const [emailError, setEmailError] = useState('')
  const [passwordError, setPasswordError] = useState('')

  const handleEmailChange = (val: string) => {
    auth.setLoginEmail(val)
    if (emailError) setEmailError('')
  }

  const handlePasswordChange = (val: string) => {
    auth.setPassword(val)
    if (passwordError) setPasswordError('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setEmailError('')
    setPasswordError('')

    let hasError = false
    if (!auth.loginEmail.trim()) {
      setEmailError('Email is required')
      hasError = true
    } else if (!/\S+@\S+\.\S+/.test(auth.loginEmail)) {
      setEmailError('Invalid email format')
      hasError = true
    }

    if (!auth.password) {
      setPasswordError('Password is required')
      hasError = true
    } else if (auth.password.length < 6) {
      setPasswordError('Password must be at least 6 characters')
      hasError = true
    }

    if (hasError) return

    await auth.signIn()
  }
  return {
    email: auth.loginEmail,
    password: auth.password,
    emailError,
    passwordError,
    submitError: auth.authError,
    authStatus: auth.authStatus,
    isBusy: auth.isBusy,
    isSignedIn: auth.isSignedIn,
    isEmailVerified: auth.isEmailVerified,
    resendCooldown: auth.resendCooldown,
    handleEmailChange,
    handlePasswordChange,
    handleSubmit,
    checkVerificationStatus: auth.checkVerificationStatus,
    resendVerification: auth.resendVerification,
    signOut: auth.signOut,
  }
}
