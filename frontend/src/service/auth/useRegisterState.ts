import { useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useRegisterState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
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

    const hasSpecialChar = /[!@#$%^&*(),.?":{}|<>]/.test(auth.password);

    if (!auth.password) {
      setPasswordError('Password is required')
      hasError = true
    } else if (auth.password.length < 6 || auth.password.length > 8) {
      setPasswordError('Password must be between 6 and 8 characters long')
      hasError = true
    } else if (!hasSpecialChar) {
      setPasswordError('Password must contain at least one special character (!@#$%^&*)')
      hasError = true
    }

    if (hasError) return

    await auth.register()
  }

  return {
    email: auth.loginEmail,
    password: auth.password,
    emailError,
    passwordError,
    submitError: auth.authError,
    authStatus: auth.authStatus,
    isBusy: auth.isBusy,
    handleEmailChange,
    handlePasswordChange,
    handleSubmit,
  }
}
