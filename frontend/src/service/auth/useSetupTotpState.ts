import { useEffect, useState } from 'react'
import type { useFirebaseTotpAuth } from './useFirebaseTotpAuth'

export const useSetupTotpState = (auth: ReturnType<typeof useFirebaseTotpAuth>) => {
  const [codeError, setCodeError] = useState('')

  useEffect(() => {
    let isMounted = true;

    if (auth.isSignedIn && auth.isEmailVerified && !auth.hasTotpEnabled && !auth.needsTotpEnrollment) {
      auth.startTotpEnrollment().catch((error) => {
        if (isMounted) console.error(error);
      });
    }
    return () => {
      isMounted = false; // update isMounted to false
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
