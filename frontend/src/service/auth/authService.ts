import {
  TotpMultiFactorGenerator,
  createUserWithEmailAndPassword,
  getMultiFactorResolver,
  multiFactor,
  sendEmailVerification,
  signInWithEmailAndPassword,
  signOut,
  type Auth,
  type AuthError,
  type MultiFactorError,
  type MultiFactorInfo,
  type MultiFactorResolver,
  type TotpSecret,
  type User,
} from 'firebase/auth'
import apiClient from '../apiClient';

export type SignInResult =
  | { type: 'signed-in' }
  | { type: 'totp-required'; resolver: MultiFactorResolver; hint: MultiFactorInfo }

export async function signInWithPassword(auth: Auth, email: string, password: string): Promise<SignInResult> {
  try {
    await signInWithEmailAndPassword(auth, email, password)
    return { type: 'signed-in' }
  } catch (error) {
    const authError = error as AuthError
    if (authError.code === 'auth/multi-factor-auth-required') {
      const resolver = getMultiFactorResolver(auth, authError as MultiFactorError)
      const hint = resolver.hints.find((item) => item.factorId === TotpMultiFactorGenerator.FACTOR_ID) ?? null
      if (!hint) {
        throw new Error('TOTP is required, but no TOTP factor is enrolled for this user.')
      }

      return { type: 'totp-required', resolver, hint }
    }

    throw error
  }
}

export async function createAccount(auth: Auth, email: string, password: string): Promise<User> {
  const credential = await createUserWithEmailAndPassword(auth, email, password)
  return credential.user
}

export async function sendVerificationEmail(user: User) {
  await sendEmailVerification(user)
}

export function hasTotpEnrollment(user: User): boolean {
  return multiFactor(user).enrolledFactors.some((factor) => factor.factorId === TotpMultiFactorGenerator.FACTOR_ID)
}

export async function resolveTotpSignIn(resolver: MultiFactorResolver, enrollmentId: string, code: string) {
  const assertion = TotpMultiFactorGenerator.assertionForSignIn(enrollmentId, code)
  await resolver.resolveSignIn(assertion)
}

export async function beginTotpEnrollment(user: User): Promise<{ secret: TotpSecret; otpauthUrl: string }> {
  const mfaSession = await multiFactor(user).getSession()
  const secret = await TotpMultiFactorGenerator.generateSecret(mfaSession)
  const otpauthUrl = secret.generateQrCodeUrl(user.email ?? user.uid, 'Support Copilot')
  return { secret, otpauthUrl }
}

export async function confirmTotpEnrollment(user: User, secret: TotpSecret, code: string) {
  const assertion = TotpMultiFactorGenerator.assertionForEnrollment(secret, code)
  await multiFactor(user).enroll(assertion, 'Authenticator')
}

export async function signOutCurrentUser(auth: Auth) {
  await signOut(auth)
}

export function toErrorMessage(error: unknown, fallback: string): string {
  const authError = error as AuthError
  switch (authError?.code) {
    case 'auth/operation-not-allowed':
      return 'Email/password sign-in is disabled for this Firebase project. Enable Email/Password in Firebase Console > Authentication > Sign-in method.'
    case 'auth/invalid-api-key':
      return 'Firebase API key is invalid. Check VITE_FIREBASE_API_KEY in your frontend environment file.'
    case 'auth/network-request-failed':
      return 'Network error while contacting Firebase. Check internet connectivity and try again.'
    case 'auth/unverified-email':
      return 'Please verify your email first, then sign in again before setting up TOTP.'
    default:
      return error instanceof Error ? error.message : fallback
  }
}

export async function exchangeToken (user: User): Promise<void>{
  try {
    const firebaseToken = await user.getIdToken();
    
   await apiClient.post('/auth/exchange', {      
      firebase_token: firebaseToken
    }, {
        withCredentials: true,
      });
  }catch(error: any){
    console.error("Token exchange failed", error)
    if (error.response && error.response.status === 403 && error.response.data?.error === 'mfa_required') {
      throw new Error('mfa_required');
    }
  throw error
  }    
}

export interface SessionData {
  authenticated: boolean
  user_id?: string
  user_uid?: string
  user_email?: string
  username?: string
  display_name?: string
  scope?: string
  totp_enabled?: boolean
}

export async function getSession(): Promise<SessionData> {
  try {
    const res = await apiClient.get('/api/auth/me', {
      withCredentials: true,
    })
    return res.data
  } catch (err) {
    return { authenticated: false }
  }
}

export async function registerWithBackend(username: string, email: string, password: string): Promise<any> {
  const res = await apiClient.post('/api/auth/register', {
    username,
    email,
    password,
  })
  return res.data
}

export type BackendLoginResult =
  | { type: 'authenticated' }
  | { type: 'totp-required' }

export async function loginWithBackend(
  usernameOrEmail: string,
  password: string,
  totpCode?: string
): Promise<BackendLoginResult> {
  try {
    await apiClient.post('/api/auth/login', {
      username_or_email: usernameOrEmail,
      password,
      totp_code: totpCode || '',
    })
    return { type: 'authenticated' }
  } catch (error: any) {
    if (
      error.response?.status === 403 &&
      (error.response?.data?.error === 'mfa_required' ||
        error.response?.data?.error === 'mfa required' ||
        error.response?.data?.message?.includes('TOTP'))
    ) {
      return { type: 'totp-required' }
    }
    const msg =
      error.response?.data?.error ||
      error.response?.data?.message ||
      error.message ||
      'Login failed'
    throw new Error(msg)
  }
}

export async function setupBackendTotp(): Promise<{ secret: string; qr_uri: string }> {
  const res = await apiClient.post('/api/auth/totp/setup')
  return res.data
}

export async function verifyBackendTotp(code: string): Promise<void> {
  await apiClient.post('/api/auth/totp/verify', { code })
}

export async function disableBackendTotp(): Promise<void> {
  await apiClient.post('/api/auth/totp/disable')
}

export async function logoutBackend(): Promise<void> {
  try {
    await apiClient.post('/api/auth/logout')
  } catch (e) {
    // ignore logout error silently
  }
}

export async function refreshBackendSession(): Promise<void> {
  await apiClient.post('/api/auth/refresh')
}

