# Authentication, TOTP MFA, and Session Management

This document details the authentication architecture, Multi-Factor Authentication (MFA), and session management in the **Support Copilot** application.

---

## 1. Architecture Overview

Support Copilot implements a unified, state-independent session pattern with dual login strategies and an integrated RFC 6238 Time-Based One-Time Password (TOTP) Multi-Factor Authentication (MFA) engine:

- **Dual Login Support**:
  - **Username / Email + Password**: Authenticated directly against the PostgreSQL database with Bcrypt hash validation (`POST /api/auth/login`).
  - **Firebase Authentication**: Authenticated on the client via Firebase Auth SDK and exchanged on the backend via Firebase Admin SDK (`POST /api/auth/exchange`).
- **Unified TOTP Multi-Factor Authentication**:
  - Accounts with TOTP enabled require a 6-digit verification code (`totp_code`) regardless of which login strategy is used.
  - If MFA is required and no code (or an invalid code) is provided, the backend returns an HTTP `403 Forbidden` with error code `mfa_required`.
- **Session Establishment**:
  - Successful authentication issues a signed HS256 JWT containing the user identity, scope, and team context.
  - The JWT is transmitted in a secure, `HttpOnly`, `SameSite=Lax` cookie named `support_copilot_session` with a 1-hour validity window.
- **Route Authorization**:
  - Protected API routes under `/api/*` and streaming query routes under `/query/*` are guarded by `AuthMiddleware`, which parses and verifies the session cookie.

---

## 2. Authentication and MFA Lifecycle

```mermaid
sequenceDiagram
    participant Client as Frontend Client
    participant Auth as Go Backend Auth Service
    participant FB as Firebase Admin SDK
    participant DB as PostgreSQL DB

    alt Strategy A: Username / Email + Password Login
        Client->>Auth: POST /api/auth/login { username_or_email, password, totp_code? }
        Auth->>DB: Fetch user record by username or email
        Auth->>Auth: Validate Bcrypt password hash
        alt User Has TOTP Enabled
            alt No totp_code or invalid
                Auth-->>Client: HTTP 403 Forbidden { error: "mfa_required" }
            else Valid 6-digit totp_code supplied
                Auth->>Auth: Verify TOTP against user.totp_secret
            end
        end
        Auth-->>Client: Sets HttpOnly cookie 'support_copilot_session' (JWT)
    else Strategy B: Firebase Login & Token Exchange
        Client->>FB: Authenticate with Firebase Client SDK
        FB-->>Client: Returns Firebase ID Token
        Client->>Auth: POST /api/auth/exchange { firebase_token, totp_code? }
        Auth->>FB: Verify Firebase ID Token
        Auth->>DB: Upsert & synchronize user profile
        alt User Has TOTP Enabled
            alt No totp_code or invalid
                Auth-->>Client: HTTP 403 Forbidden { error: "mfa_required" }
            else Valid 6-digit totp_code supplied
                Auth->>Auth: Verify TOTP against user.totp_secret
            end
        end
        Auth-->>Client: Sets HttpOnly cookie 'support_copilot_session' (JWT)
    end

    Note over Client,Auth: Subsequent Authenticated Requests
    Client->>Auth: GET /api/auth/me (Sends session cookie)
    Auth-->>Client: Returns User profile & active team metadata
```

---

## 3. Unified TOTP Multi-Factor Authentication

Support Copilot incorporates an RFC 6238 TOTP implementation compatible with standard authenticator applications (Google Authenticator, Microsoft Authenticator, Authy, 1Password).

### TOTP Setup and Provisioning

```mermaid
sequenceDiagram
    participant Client as Frontend Client
    participant API as Go Backend API
    participant App as Authenticator App

    Client->>API: POST /api/auth/totp/setup (Authenticated)
    API->>API: Generate Base32 TOTP secret & otpauth:// URI
    API-->>Client: { secret: "JBSWY3DPEHPK3PXP", qr_uri: "otpauth://totp/..." }
    Client->>App: Render QR Code & scan into Authenticator App
    App-->>Client: Generates 6-digit rolling code
    Client->>API: POST /api/auth/totp/verify { code: "123456" }
    API->>API: Validate code using HMAC-SHA1 algorithm
    API-->>Client: { status: "2fa_enabled" }
```

---

## 4. Session Management and Token Rotation

The application avoids database-level stateful session tracking by utilizing signed JWTs wrapped in browser cookies.

```mermaid
sequenceDiagram
    participant Client as Frontend Client
    participant Interceptor as Axios Interceptor
    participant FB as Firebase Auth
    participant API as Go Backend API

    Client->>API: API Request fails with HTTP 401 Unauthorized
    alt User Authenticated via Firebase
        Interceptor->>FB: user.getIdToken(forceRefresh)
        FB-->>Interceptor: Fresh Firebase ID Token
        Interceptor->>API: POST /api/auth/exchange { firebase_token: new_token }
        API-->>Interceptor: Sets refreshed 'support_copilot_session' cookie
        Interceptor->>API: Retry original API request
    else User Authenticated via Password
        Interceptor-->>Client: Redirect to /login for credential re-entry
    end
```