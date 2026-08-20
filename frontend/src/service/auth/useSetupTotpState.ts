import { useEffect, useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useSetupTotpState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
  const [codeError, setCodeError] = useState('')

  // Auto-trigger TOTP enrollment as soon as the user lands on this page
  // with a verified email but no TOTP set up yet.
  useEffect(() => {
    let isMounted = true;

    if (auth.isSignedIn && auth.isEmailVerified && !auth.hasTotpEnabled && !auth.needsTotpEnrollment) {
      auth.startTotpEnrollment().catch((error) => {
        // startTotpEnrollment already calls setAuthError internally, so this
        // catch only handles truly unexpected rejections.
        if (isMounted) console.error('Unexpected TOTP enrollment error:', error);
      });
    }
    return () => {
      isMounted = false;
    };
  }, [auth.isSignedIn, auth.isEmailVerified, auth.hasTotpEnabled, auth.needsTotpEnrollment, auth.startTotpEnrollment]);

  const handleCodeChange = (val: string) => {
    auth.setEnrollCode(val)
    if (codeError) setCodeError('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setCodeError('')

    const code = auth.enrollCode.trim()
    if (!code) {
      setCodeError('Verification code is required')
      return
    }
    if (code.length !== 6 || !/^\d+$/.test(code)) {
      setCodeError('Code must be a 6-digit number')
      return
    }

    await auth.enrollTotp()
  }

  return {
    enrollOtpAuthUrl: auth.enrollOtpAuthUrl,
    enrollCode: auth.enrollCode,
    codeError,
    submitError: auth.authError,
    authStatus: auth.authStatus,
    isBusy: auth.isBusy,
    needsTotpEnrollment: auth.needsTotpEnrollment,
    isEmailVerified: auth.isEmailVerified,
    checkVerificationStatus: auth.checkVerificationStatus,
    resendVerification: auth.resendVerification,
    handleCodeChange,
    handleSubmit,
    handleCancel: auth.signOut,
  }
}
