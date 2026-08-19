import { useEffect, useState } from 'react'
import { onIdTokenChanged, type MultiFactorInfo, type MultiFactorResolver, type TotpSecret } from 'firebase/auth'
import { firebaseAuth } from '../../firebase'
import {
  beginTotpEnrollment,
  confirmTotpEnrollment,
  createAccount,
  hasTotpEnrollment,
  resolveTotpSignIn,
  sendVerificationEmail,
  signInWithPassword,
  signOutCurrentUser,
  toErrorMessage,
  exchangeToken,
  getSession,
  registerWithBackend,
  loginWithBackend,
  setupBackendTotp,
  verifyBackendTotp,
  logoutBackend,
} from './authService'

export function useFirebaseTotpAuth() {
  const [token, setToken] = useState('')
  const [rawToken, setRawToken] = useState('')
  const [username, setUsername] = useState('')
  const [loginIdentifier, setLoginIdentifier] = useState('')
  const [loginEmail, setLoginEmail] = useState('')
  const [userEmail, setUserEmail] = useState('')
  const [password, setPassword] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [enrollCode, setEnrollCode] = useState('')
  const [isBusy, setIsBusy] = useState(false)
  const [totpResolver, setTotpResolver] = useState<MultiFactorResolver | null>(null)
  const [totpHint, setTotpHint] = useState<MultiFactorInfo | null>(null)
  const [isBackendMfaPending, setIsBackendMfaPending] = useState(false)
  const [enrollSecret, setEnrollSecret] = useState<TotpSecret | null>(null)
  const [enrollOtpAuthUrl, setEnrollOtpAuthUrl] = useState('')
  const [hasTotpEnabled, setHasTotpEnabled] = useState(false)
  const [isEmailVerified, setIsEmailVerified] = useState(false)
  const [isAuthReady, setIsAuthReady] = useState(false)
  const [authStatus, setAuthStatus] = useState('Not signed in')
  const [authError, setAuthError] = useState('')
  const [authProvider, setAuthProvider] = useState<'backend' | 'firebase'>('backend')
  const [resendCooldown, setResendCooldown] = useState(0)

  // countdown timer for resend cooldown
  useEffect(() => {
    if (resendCooldown <= 0) return
    const timer = setInterval(() => {
      setResendCooldown((prev) => prev - 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [resendCooldown])

  const checkVerificationStatus = async () => {
    const user = firebaseAuth.currentUser
    if (!user) return

    setAuthError('')
    setIsBusy(true)
    try {
      await user.reload()
      const refreshedUser = firebaseAuth.currentUser
      if (refreshedUser) {
        setIsEmailVerified(refreshedUser.emailVerified)
        setHasTotpEnabled(hasTotpEnrollment(refreshedUser))
        if (refreshedUser.emailVerified) {
          await exchangeToken(refreshedUser)
          setRawToken('COOKIE_CONTAINED_SESSION')
          setToken('Bearer COOKIE_CONTAINED_SESSION')
          setAuthStatus('Email verified. You must set up TOTP to access the workspace.')
        } else {
          setAuthStatus('Email is still unverified. Please check your inbox.')
        }
      }
    } catch (error) {
      setAuthError(toErrorMessage(error, 'Email verification check failed'))
    } finally {
      setIsBusy(false)
    }
  }

  useEffect(() => {
    let unsubscribe: (() => void) | null = null
    let isMounted = true

    const initAuth = async () => {
      let activeSessionUid: string | null = null

      try {
        const session = await getSession()
        if (session.authenticated && (session.user_uid || session.user_id)) {
          activeSessionUid = session.user_uid || session.user_id || null
          if (isMounted) {
            setToken('COOKIE_SESSION')
            setRawToken('COOKIE_SESSION')
            setUserEmail(session.user_email || session.user_uid || '')
            setIsEmailVerified(true)
            setHasTotpEnabled(Boolean(session.totp_enabled))
            setAuthStatus(`Authenticated as ${session.user_email || session.user_uid}`)
            setIsAuthReady(true)
          }
        }
      } catch (e) {
        // session verification failed or not logged in
      }

      if (!isMounted) return

      try {
        unsubscribe = onIdTokenChanged(firebaseAuth, async (user) => {
          if (!isMounted) return

          if (!user) {
            if (!activeSessionUid) {
              setToken('')
              setUserEmail('')
              setRawToken('')
              setHasTotpEnabled(false)
              setIsEmailVerified(false)
              setAuthStatus('Not signed in')
            }
            setIsAuthReady(true)
            return
          }

          setAuthProvider('firebase')

          if (!user.emailVerified) {
            setToken('UNVERIFIED_EMAIL_HOLDER')
            setIsEmailVerified(false)
            setHasTotpEnabled(false)
            setAuthStatus('Account created. Verification email sent.')
            setIsAuthReady(true)
            return
          }

          if (activeSessionUid === user.uid) {
            setToken('COOKIE_SESSION')
            setRawToken('COOKIE_SESSION')
            setUserEmail(user.email ?? '')
            setHasTotpEnabled(hasTotpEnrollment(user))
            setIsEmailVerified(user.emailVerified)
            setAuthStatus(`Authenticated securely as ${user.email ?? user.uid}`)
            setIsAuthReady(true)
            return
          }

          try {
            setAuthStatus('Synchronizing session credentials...')
            await exchangeToken(user)
            if (isMounted) {
              setToken('COOKIE_SESSION')
              setRawToken('COOKIE_SESSION')
              setUserEmail(user.email ?? '')
              setHasTotpEnabled(hasTotpEnrollment(user))
              setIsEmailVerified(user.emailVerified)
              setAuthStatus(`Authenticated securely as ${user.email ?? user.uid}`)
              setIsAuthReady(true)
              activeSessionUid = user.uid
            }
          } catch (err: any) {
            if (isMounted) {
              if (err.message === 'mfa_required') {
                setToken('MFA_PENDING_CLIENT_TOKEN')
                setHasTotpEnabled(true)
                setAuthStatus('Multi-factor verification required to establish session.')
              } else {
                setAuthError('Backend synchronization failed. Please sign in again.')
                await signOutCurrentUser(firebaseAuth)
              }
            }
          } finally {
            if (isMounted) {
              setIsAuthReady(true)
            }
          }
        })
      } catch (err) {
        // firebase initialization or network error
        if (isMounted) {
          setIsAuthReady(true)
        }
      }
    }

    const handleWindowFocus = () => {
      const user = firebaseAuth.currentUser
      if (user && !user.emailVerified) {
        checkVerificationStatus()
      }
    }
    window.addEventListener('focus', handleWindowFocus)

    initAuth()

    return () => {
      isMounted = false
      window.removeEventListener('focus', handleWindowFocus)
      if (unsubscribe) {
        unsubscribe()
      }
    }
  }, [])

  const isSignedIn = token !== ''

  // direct backend username or email and password sign in
  const signInBackend = async (identifier?: string, pwd?: string) => {
    if (isBusy) return

    const loginId = (identifier || loginIdentifier || loginEmail).trim()
    const loginPassword = pwd || password

    if (!loginId || !loginPassword) {
      setAuthError('Username/email and password are required')
      return
    }

    setAuthError('')
    setIsBusy(true)
    try {
      const result = await loginWithBackend(loginId, loginPassword)
      if (result.type === 'totp-required') {
        setIsBackendMfaPending(true)
        setAuthProvider('backend')
        setAuthStatus('Enter your 6-digit TOTP to continue.')
        return
      }

      setIsBackendMfaPending(false)
      setAuthProvider('backend')
      setTotpCode('')
      const session = await getSession()
      setToken('COOKIE_SESSION')
      setRawToken('COOKIE_SESSION')
      setUserEmail(session.user_email || loginId)
      setIsEmailVerified(true)
      setHasTotpEnabled(Boolean(session.totp_enabled))
      setAuthStatus('Signed in successfully.')
    } catch (error: any) {
      setAuthError(error.message || 'Login failed')
      setAuthStatus('Login failed')
    } finally {
      setIsBusy(false)
    }
  }

  // direct backend registration
  const registerBackend = async (user?: string, email?: string, pwd?: string) => {
    if (isBusy) return

    const regUser = (user || username).trim()
    const regEmail = (email || loginEmail).trim()
    const regPassword = pwd || password

    if (!regUser || !regEmail || !regPassword) {
      setAuthError('Username, email, and password are required')
      return
    }

    setAuthError('')
    setIsBusy(true)
    try {
      await registerWithBackend(regUser, regEmail, regPassword)
      setAuthStatus('Account created successfully! Please sign in with your credentials.')
      setLoginIdentifier(regUser)
      setLoginEmail(regEmail)
      return true
    } catch (error: any) {
      setAuthError(error.message || 'Registration failed')
      setAuthStatus('Registration failed')
      return false
    } finally {
      setIsBusy(false)
    }
  }

  // firebase sign in with email and password
  const signIn = async () => {
    if (isBusy) return

    setAuthError('')
    setIsBusy(true)
    setAuthProvider('firebase')
    try {
      const result = await signInWithPassword(firebaseAuth, (loginEmail || loginIdentifier).trim(), password)
      if (result.type === 'totp-required') {
        setTotpResolver(result.resolver)
        setTotpHint(result.hint)
        setAuthStatus('Enter your 6-digit TOTP to continue.')
        return
      }

      setTotpResolver(null)
      setTotpHint(null)
      setTotpCode('')
      if (!firebaseAuth.currentUser?.emailVerified) {
        setAuthStatus('Signed in, but email is not verified yet. Verify email first, then set up TOTP.')
      }
      const user = firebaseAuth.currentUser
      if (user) {
        await exchangeToken(user)
        setToken('COOKIE_SESSION')
        setRawToken('COOKIE_SESSION')
        setUserEmail(user.email ?? '')
        setHasTotpEnabled(hasTotpEnrollment(user))
        setIsEmailVerified(user.emailVerified)
      }
    } catch (error) {
      setAuthError(toErrorMessage(error, 'Sign-in failed'))
      setAuthStatus('Sign-in failed')
    } finally {
      setIsBusy(false)
    }
  }

  // verify totp during sign in for either backend or firebase
  const verifyTotpSignIn = async () => {
    if (isBusy) return

    const code = totpCode.trim()
    if (!code) {
      setAuthError('TOTP code is required')
      return
    }

    setAuthError('')
    setIsBusy(true)

    try {
      if (isBackendMfaPending || authProvider === 'backend') {
        const loginId = (loginIdentifier || loginEmail).trim()
        const result = await loginWithBackend(loginId, password, code)
        if (result.type === 'authenticated') {
          setIsBackendMfaPending(false)
          const session = await getSession()
          setUserEmail(session.user_email || loginId)
          setToken('COOKIE_SESSION')
          setRawToken('COOKIE_SESSION')
          setHasTotpEnabled(true)
          setIsEmailVerified(true)
          setTotpCode('')
          setAuthStatus('TOTP verified. Signed in.')
        } else {
          throw new Error('TOTP verification failed')
        }
      } else if (totpResolver && totpHint) {
        await resolveTotpSignIn(totpResolver, totpHint.uid, code)
        const user = firebaseAuth.currentUser
        if (!user) {
          throw new Error('Firebase user missing after TOTP sign in')
        }
        await exchangeToken(user)
        setUserEmail(user.email ?? '')
        setToken('COOKIE_SESSION')
        setRawToken('COOKIE_SESSION')
        setHasTotpEnabled(hasTotpEnrollment(user))
        setIsEmailVerified(user.emailVerified)
        setTotpResolver(null)
        setTotpHint(null)
        setTotpCode('')
        setAuthStatus('TOTP verified. Signed in.')
      }
    } catch (error: any) {
      setAuthError(toErrorMessage(error, error.message || 'TOTP verification failed'))
      setAuthStatus('TOTP verification failed')
    } finally {
      setIsBusy(false)
    }
  }

  // firebase registration
  const register = async () => {
    if (isBusy) return

    setAuthError('')
    setIsBusy(true)
    setAuthProvider('firebase')
    try {
      const user = await createAccount(firebaseAuth, (loginEmail || loginIdentifier).trim(), password)
      await sendVerificationEmail(user)
      setEnrollSecret(null)
      setEnrollOtpAuthUrl('')
      setEnrollCode('')
      setAuthStatus('Account created. Verification email sent. Please check your inbox and verify your email, then click "Check status".')
      setTotpResolver(null)
      setTotpHint(null)
      setTotpCode('')
    } catch (error) {
      setAuthError(toErrorMessage(error, 'Account creation failed'))
      setAuthStatus('Account creation failed')
    } finally {
      setIsBusy(false)
    }
  }

  // start totp enrollment for native backend or firebase
  const startTotpEnrollment = async () => {
    if (isBusy || enrollSecret || enrollOtpAuthUrl) return

    const user = firebaseAuth.currentUser
    if (authProvider === 'firebase' && user) {
      if (!user.emailVerified) {
        setAuthError('Please verify your email first before enrolling TOTP')
        setAuthStatus('Email verification required before TOTP setup')
        return
      }
      if (hasTotpEnrollment(user)) {
        setAuthStatus('TOTP is already enrolled for this account.')
        return
      }

      setAuthError('')
      setIsBusy(true)
      try {
        const { secret, otpauthUrl } = await beginTotpEnrollment(user)
        setEnrollSecret(secret)
        setEnrollOtpAuthUrl(otpauthUrl)
        setEnrollCode('')
        setAuthStatus('Scan QR and enter code to complete TOTP setup.')
      } catch (error) {
        setAuthError(toErrorMessage(error, 'Unable to start TOTP enrollment'))
        setAuthStatus('Unable to start TOTP setup')
      } finally {
        setIsBusy(false)
      }
    } else {
      // native backend totp setup
      setAuthError('')
      setIsBusy(true)
      try {
        const res = await setupBackendTotp()
        setEnrollOtpAuthUrl(res.qr_uri)
        setEnrollCode('')
        setAuthStatus('Scan QR code and enter 6-digit code to enable TOTP.')
      } catch (error: any) {
        setAuthError(error.message || 'Unable to start TOTP setup')
        setAuthStatus('Unable to start TOTP setup')
      } finally {
        setIsBusy(false)
      }
    }
  }

  // complete totp enrollment for native backend or firebase
  const enrollTotp = async () => {
    if (isBusy) return

    const code = enrollCode.trim()
    if (!code) {
      setAuthError('Enrollment code is required')
      return
    }

    setAuthError('')
    setIsBusy(true)
    try {
      const user = firebaseAuth.currentUser
      if (authProvider === 'firebase' && user && enrollSecret) {
        await confirmTotpEnrollment(user, enrollSecret, code)
        setEnrollSecret(null)
        setEnrollOtpAuthUrl('')
        setEnrollCode('')
        setAuthStatus('TOTP enrolled. Sign in again to verify TOTP during login.')
        await signOutCurrentUser(firebaseAuth)
      } else {
        // native backend totp verification
        await verifyBackendTotp(code)
        setEnrollOtpAuthUrl('')
        setEnrollCode('')
        setHasTotpEnabled(true)
        setAuthStatus('TOTP enrolled successfully!')
      }
    } catch (error: any) {
      setAuthError(toErrorMessage(error, error.message || 'TOTP enrollment failed'))
      setAuthStatus('TOTP enrollment failed')
    } finally {
      setIsBusy(false)
    }
  }

  const resendVerification = async () => {
    if (isBusy || resendCooldown > 0) return
    const user = firebaseAuth.currentUser
    if (!user) {
      setAuthError('Sign in first to resend verification email')
      return
    }
    if (user.emailVerified) {
      setAuthStatus('Email is already verified.')
      return
    }

    setAuthError('')
    setIsBusy(true)
    try {
      await sendVerificationEmail(user)
      setResendCooldown(60)
      setAuthStatus('Verification email sent. Check your inbox and spam folder.')
    } catch (error) {
      setAuthError(toErrorMessage(error, 'Failed to resend verification email'))
    } finally {
      setIsBusy(false)
    }
  }

  const signOut = async () => {
    if (isBusy) return

    setAuthError('')
    setIsBusy(true)
    try {
      await logoutBackend()
      await signOutCurrentUser(firebaseAuth).catch(() => {})
      setTotpResolver(null)
      setTotpHint(null)
      setIsBackendMfaPending(false)
      setTotpCode('')
      setEnrollSecret(null)
      setEnrollOtpAuthUrl('')
      setEnrollCode('')
      setHasTotpEnabled(false)
      setIsEmailVerified(false)
      setToken('')
      setRawToken('')
      setUserEmail('')
      setAuthStatus('Signed out')
    } catch (error) {
      setAuthError(toErrorMessage(error, 'Sign-out failed'))
    } finally {
      setIsBusy(false)
    }
  }

  return {
    token,
    rawToken,
    username,
    setUsername,
    loginIdentifier,
    setLoginIdentifier,
    loginEmail,
    setLoginEmail,
    userEmail,
    password,
    setPassword,
    totpCode,
    setTotpCode,
    enrollCode,
    setEnrollCode,
    enrollOtpAuthUrl,
    hasTotpEnabled,
    isEmailVerified,
    isAuthReady,
    resendCooldown,
    authProvider,
    setAuthProvider,
    needsTotpSignIn: (totpResolver !== null && totpHint !== null) || isBackendMfaPending,
    needsTotpEnrollment: enrollSecret !== null || enrollOtpAuthUrl !== '',
    canStartTotpEnrollment: isSignedIn && isEmailVerified && !hasTotpEnabled && enrollSecret === null && enrollOtpAuthUrl === '',
    isBusy,
    isSignedIn,
    authStatus,
    authError,
    signIn,
    signInBackend,
    register,
    registerBackend,
    verifyTotpSignIn,
    startTotpEnrollment,
    enrollTotp,
    resendVerification,
    signOut,
    checkVerificationStatus,
  }
}
