# Authentication, TOTP MFA, and Session Management

This document details the authentication architecture, Multi-Factor Authentication (MFA), and session management in the **Support Copilot** application.

---

## 1. Architecture Overview

Support Copilot implements a state-independent session pattern with dual authentication strategies and an integrated Multi-Factor Authentication (MFA) engine:

- **Dual Login Support**:
  - **Username / Email + Password**: Authenticated directly against the PostgreSQL database with Bcrypt hash validation.
  - **Firebase Authentication (Cloud)**: Authenticated on the client via Firebase Auth SDK (Google OAuth / Email + Password) and exchanged on the backend via Firebase Admin SDK.
- **Separation of 2FA Responsibilities**:
  - **Local Users**: Protected by a self-hosted RFC 6238 TOTP engine managed entirely by the Go backend with secrets stored in PostgreSQL.
  - **Firebase Users**: Protected by Firebase Identity Platform Multi-Factor Authentication, with 2FA enrollment and verification handled directly in Firebase Cloud before issuing the Firebase ID token.
  - **Token Exchange**: The backend verifies the Firebase ID token using the Firebase Admin SDK. Because Firebase verifies cloud MFA before signing the ID token, the backend directly issues the session cookie without requiring a duplicate PostgreSQL TOTP secret (unless a local secret was explicitly provisioned).
- **Session Establishment**:
  - Successful authentication issues a signed HS256 JWT containing the user identity, scope, and authentication method.
  - The JWT is transmitted in a secure, HttpOnly, SameSite=Lax cookie named `support_copilot_session` with a 1-hour validity window.

---

## 2. Authentication Lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant Client as Frontend Client
    participant Auth as Go Backend Auth Service
    participant FB as Firebase Admin SDK / Cloud
    participant DB as PostgreSQL DB

    alt Strategy A: Local Username / Email + Password Login
        Client->>Auth: POST /api/auth/login { username_or_email, password, totp_code? }
        Auth->>DB: Fetch user record by username or email
        alt User Not Found
            Auth-->>Client: HTTP 404 Not Found { error: "User not found" }
        else User Exists
            Auth->>Auth: Validate Bcrypt password hash
            alt Password Mismatch
                Auth-->>Client: HTTP 401 Unauthorized { error: "invalid credentials" }
            else Password Matches
                alt User has TOTP Enabled & Secret Configured
                    alt No totp_code or invalid
                        Auth-->>Client: HTTP 403 Forbidden { error: "mfa_required" }
                    else Valid 6-digit totp_code supplied
                        Auth->>Auth: Verify TOTP against user.totp_secret
                        Auth-->>Client: HTTP 200 OK + Sets HttpOnly cookie 'support_copilot_session'
                    end
                else No TOTP Required
                    Auth-->>Client: HTTP 200 OK + Sets HttpOnly cookie 'support_copilot_session'
                end
            end
        end

    else Strategy B: Firebase Login & Token Exchange
        Client->>FB: Authenticate with Firebase Client SDK (Email/Password or OAuth)
        opt Firebase MFA Enrolled
            FB-->>Client: MFA Challenge required (auth/multi-factor-auth-required)
            Client->>FB: Resolve TOTP challenge with 6-digit code
        end
        FB-->>Client: Returns verified Firebase ID Token
        Client->>Auth: POST /auth/exchange { firebase_token }
        Auth->>FB: Verify Firebase ID Token via Firebase Admin SDK
        Auth->>DB: Upsert & synchronize user profile
        Auth-->>Client: HTTP 200 OK + Sets HttpOnly cookie 'support_copilot_session'
    end

    Note over Client,Auth: Subsequent Authenticated Requests
    Client->>Auth: GET /api/auth/me (Sends session cookie)
    Auth-->>Client: Returns User profile, firebase_uid, totp_enabled & team metadata
```

---

## 3. TOTP Multi-Factor Authentication Provisioning

### Local Users (Backend RFC 6238 TOTP)
Local username/password accounts use the Go backend's native RFC 6238 TOTP engine built with standard crypto libraries (`crypto/hmac`, `crypto/sha1`, `encoding/base32`):

```mermaid
sequenceDiagram
    autonumber
    participant Client as Frontend Client
    participant API as Go Backend API
    participant App as Authenticator App (Google Auth / 1Password)

    Client->>API: POST /api/auth/totp/setup (Authenticated)
    API->>API: Generate 20-byte Base32 secret & otpauth:// URI
    API->>API: Save secret to user.totp_secret (totp_enabled remains false)
    API-->>Client: { secret: "JBSWY3D...", qr_uri: "otpauth://totp/..." }
    Client->>App: Render QR Code & scan into Authenticator App
    App-->>Client: Generates 6-digit rolling code
    Client->>API: POST /api/auth/totp/verify { code: "123456" }
    API->>API: Validate code using HMAC-SHA1 against user.totp_secret
    API->>API: Set user.totp_enabled = true
    API-->>Client: HTTP 200 OK { status: "2fa_enabled" }
```

### Firebase Users (Firebase Cloud MFA)
Firebase users enroll in MFA directly via the Firebase Auth SDK, storing the 2FA secret in Google Firebase Cloud without requiring a PostgreSQL secret.

---

## 4. Session Management and Token Rotation

The application uses signed JWTs stored in `HttpOnly` browser cookies:

```mermaid
sequenceDiagram
    autonumber
    participant Client as Frontend Client
    participant Interceptor as Axios Interceptor
    participant FB as Firebase Auth
    participant API as Go Backend API

    Client->>API: API Request fails with HTTP 401 Unauthorized
    alt User Authenticated via Firebase
        Interceptor->>FB: user.getIdToken(forceRefresh)
        FB-->>Interceptor: Fresh Firebase ID Token
        Interceptor->>API: POST /auth/exchange { firebase_token: new_token }
        API-->>Interceptor: Sets refreshed 'support_copilot_session' cookie
        Interceptor->>API: Retry original API request
    else User Authenticated via Local Password
        Interceptor->>API: POST /api/auth/refresh (Sends existing session cookie)
        alt Token within 24-hour refresh grace window
            API-->>Interceptor: Sets renewed 'support_copilot_session' cookie
            Interceptor->>API: Retry original API request
        else Session expired beyond grace window
            Interceptor-->>Client: Redirect to /login for credential re-entry
        end
    end
```