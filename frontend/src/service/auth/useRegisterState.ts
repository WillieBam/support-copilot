import { useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useRegisterState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [authMode, setAuthMode] = useState<'backend' | 'firebase'>('backend')
  const [usernameError, setUsernameError] = useState('')
  const [emailError, setEmailError] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [registrationSuccess, setRegistrationSuccess] = useState(false)

  const handleUsernameChange = (val: string) => {
    setUsername(val)
    auth.setUsername(val)
    if (usernameError) setUsernameError('')
  }

  const handleEmailChange = (val: string) => {
    setEmail(val)
    auth.setLoginEmail(val)
    if (emailError) setEmailError('')
  }

  const handlePasswordChange = (val: string) => {
    setPassword(val)
    auth.setPassword(val)
    if (passwordError) setPasswordError('')
  }

  const handleModeChange = (mode: 'backend' | 'firebase') => {
    setAuthMode(mode)
    auth.setAuthProvider(mode)
    setUsernameError('')
    setEmailError('')
    setPasswordError('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setUsernameError('')
    setEmailError('')
    setPasswordError('')

    let hasError = false

    if (authMode === 'backend') {
      if (!username.trim()) {
        setUsernameError('Username is required')
        hasError = true
      } else if (username.trim().length < 3) {
        setUsernameError('Username must be at least 3 characters')
        hasError = true
      }
    }

    if (!email.trim()) {
      setEmailError('Email is required')
      hasError = true
    } else if (!/\S+@\S+\.\S+/.test(email.trim())) {
      setEmailError('Invalid email format')
      hasError = true
    }

    const hasSpecialChar = /[!@#$%^&*(),.?":{}|<>]/.test(password)

    if (!password) {
      setPasswordError('Password is required')
      hasError = true
    } else if (password.length < 6 || password.length > 8) {
      setPasswordError('Password must be between 6 and 8 characters long')
      hasError = true
    } else if (!hasSpecialChar) {
      setPasswordError('Password must contain at least one special character (!@#$%^&*)')
      hasError = true
    }

    if (hasError) return

    if (authMode === 'backend') {
      const ok = await auth.registerBackend(username, email, password)
      if (ok) {
        setRegistrationSuccess(true)
      }
    } else {
      await auth.register()
    }
  }

  return {
    username,
    email,
    password,
    authMode,
    usernameError,
    emailError,
    passwordError,
    registrationSuccess,
    submitError: auth.authError,
    authStatus: auth.authStatus,
    isBusy: auth.isBusy,
    handleUsernameChange,
    handleEmailChange,
    handlePasswordChange,
    handleModeChange,
    handleSubmit,
  }
}
