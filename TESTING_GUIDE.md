# Testing Guide - Phase 1 Authentication

## 🚀 Quick Start Testing

### Prerequisites
1. Backend services running on `http://localhost:8000`
2. Frontend running on `http://localhost:5173`
3. Database seeded with test data (optional)

### Start the Frontend
```powershell
cd frontend/hoshi-vue
npm run dev
```

## 📋 Test Scenarios

### Test 1: New User Registration
**Goal**: Verify complete registration and email verification flow

**Steps**:
1. Navigate to `http://localhost:5173/signup`
2. Fill in the registration form:
   - **Name**: John Doe (>4 characters, no numbers/symbols)
   - **Username**: johndoe123 (3-30 chars, letters/numbers/underscores only)
   - **Email**: johndoe@example.com
   - **Password**: Test@1234 (min 8 chars, 1 upper, 1 lower, 1 number, 1 special)
   - **Confirm Password**: Test@1234
   - **Gender**: Male or Female
   - **Date of Birth**: Select date (must be 13+ years old)
3. Submit form
4. Should redirect to `/verify-otp?email=johndoe@example.com`
5. Check backend logs/email service for 6-digit OTP
6. Enter OTP in verification page
7. Click "Verify Account"
8. Should show success message
9. Auto-redirect to `/login?verified=true&email=johndoe@example.com`
10. See success message: "Email verified successfully!"

**Expected Results**:
- ✅ Form validation shows errors for invalid inputs
- ✅ Loading button shows "Creating account..." during submission
- ✅ Successful redirect to OTP page with email in URL
- ✅ OTP verification succeeds
- ✅ Login page shows verification success message
- ✅ Email field pre-filled on login page

**Failure Cases to Test**:
- Invalid name (too short or contains numbers)
- Invalid username (too short/long or invalid characters)
- Invalid email format
- Weak password (missing uppercase/lowercase/number/special char)
- Password mismatch
- Age under 13
- Duplicate username/email (should show backend error)
- Invalid OTP code (should show error)

---

### Test 2: Standard Login (No 2FA)
**Goal**: Verify login without 2FA requirement

**Steps**:
1. Navigate to `http://localhost:5173/login`
2. Enter credentials:
   - **Username/Email**: Use registered email or username
   - **Password**: Your password
3. Click "Log in"
4. Should redirect to `/feed`

**Expected Results**:
- ✅ Loading button shows "Logging in..."
- ✅ JWT token saved to localStorage
- ✅ User data saved to localStorage
- ✅ Redirect to feed page
- ✅ Auth store `isAuthenticated` is true

**Failure Cases**:
- Wrong password (should show error)
- Non-existent username/email (should show error)
- Unverified account (should show verification prompt)

---

### Test 3: Login with 2FA
**Goal**: Verify two-factor authentication flow

**Setup**: Account must have 2FA enabled in database

**Steps**:
1. Navigate to `http://localhost:5173/login`
2. Enter credentials for 2FA-enabled account
3. Click "Log in"
4. Should redirect to `/login-otp`
5. Check backend logs/email for 6-digit OTP
6. Enter OTP
7. Click "Verify"
8. Should redirect to `/feed`

**Expected Results**:
- ✅ Initial login doesn't complete immediately
- ✅ Redirect to OTP verification page
- ✅ OTP input accepts 6 digits
- ✅ Loading state during verification
- ✅ Successful verification redirects to feed
- ✅ sessionStorage cleared after verification

**Failure Cases**:
- Invalid OTP (should show error)
- Expired OTP (should show error)
- Session expired (should redirect to login)

---

### Test 4: Unverified Account Login
**Goal**: Test handling of inactive/unverified accounts

**Setup**: Register account but don't verify OTP

**Steps**:
1. Navigate to `http://localhost:5173/login`
2. Try to log in with unverified account
3. Should see error message
4. "Click here to verify your account" link should appear
5. Click verification link
6. Should redirect to `/verify-otp?email=your@email.com`
7. Complete verification
8. Return to login

**Expected Results**:
- ✅ Login blocked for unverified account
- ✅ Error message explains account not verified
- ✅ Verification link appears
- ✅ Link redirects to OTP page with correct email
- ✅ After verification, can log in successfully

---

### Test 5: OTP Resend Functionality
**Goal**: Verify OTP resend cooldown and functionality

**Steps**:
1. Reach any OTP verification page (registration or 2FA)
2. Observe 60-second countdown
3. Wait for countdown to reach 0
4. Click "Resend Code"
5. Countdown should restart at 60
6. Check backend logs for new OTP

**Expected Results**:
- ✅ Initial 60-second cooldown active
- ✅ Resend button disabled during cooldown
- ✅ Timer displays remaining seconds
- ✅ Resend button enabled at 0 seconds
- ✅ Clicking resend triggers new OTP
- ✅ New 60-second cooldown starts
- ✅ Loading state during resend

---

### Test 6: Navigation and Persistence
**Goal**: Verify auth state persists across page refreshes

**Steps**:
1. Log in successfully
2. Note the JWT token in localStorage (F12 → Application → Local Storage)
3. Refresh the page
4. Navigate to different routes
5. Close and reopen browser
6. Return to site

**Expected Results**:
- ✅ Token persists in localStorage
- ✅ Auth store loads user from localStorage on mount
- ✅ `isAuthenticated` remains true after refresh
- ✅ User can navigate without re-authenticating
- ✅ Protected routes accessible (once implemented)

---

### Test 7: Logout Flow
**Goal**: Verify logout clears auth state properly

**Steps**:
1. Log in successfully
2. Navigate to any protected page
3. Trigger logout (when logout button is implemented)
4. Check localStorage (should be cleared)
5. Try to access protected route
6. Should redirect to login

**Expected Results**:
- ✅ Token removed from localStorage
- ✅ User data removed from localStorage
- ✅ `isAuthenticated` becomes false
- ✅ Store state cleared
- ✅ Redirect to login page

---

## 🔍 Debugging Tips

### Check Auth Store State
Open browser console and run:
```javascript
// Check if user is authenticated
JSON.parse(localStorage.getItem('jwt_token'))

// Check user data
JSON.parse(localStorage.getItem('user'))

// Check store state (if Vue DevTools installed)
// Vue DevTools → Pinia → auth
```

### Common Issues

#### Issue: "Cannot find module '@/stores/auth'"
**Solution**: TypeScript language server needs reload
- Save any file to trigger reload
- Or: `Ctrl+Shift+P` → "Developer: Reload Window"
- This is a cosmetic IDE issue, not a runtime error

#### Issue: CORS errors
**Solution**: Ensure API Gateway allows frontend origin
- Backend should have CORS middleware configured
- Check `Access-Control-Allow-Origin` in backend

#### Issue: 401 Unauthorized after login
**Solution**: Check JWT token
- Verify token is being saved to localStorage
- Check `Authorization` header in network requests
- Ensure axios interceptor is adding token

#### Issue: Network errors
**Solution**: Verify backend is running
- Check `http://localhost:8000/health` (if health endpoint exists)
- Verify API Gateway is running on port 8000
- Check docker containers: `docker ps`

## 🧪 Network Request Verification

### Using Browser DevTools
1. Open DevTools (F12)
2. Go to Network tab
3. Perform authentication action
4. Verify requests:

#### Expected POST /api/register
```json
// Request Payload
{
  "name": "John Doe",
  "username": "johndoe123",
  "email": "johndoe@example.com",
  "password": "Test@1234",
  "confirm_password": "Test@1234",
  "gender": "male",
  "date_of_birth": "2000-01-01"
}

// Response (201 Created)
{
  "message": "Registration successful. Please verify your email.",
  "email": "johndoe@example.com"
}
```

#### Expected POST /api/login (No 2FA)
```json
// Request Payload
{
  "email_or_username": "johndoe123",
  "password": "Test@1234"
}

// Response (200 OK)
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user_id": 1,
  "username": "johndoe123",
  "email": "johndoe@example.com"
}
```

#### Expected POST /api/login (With 2FA)
```json
// Response (200 OK)
{
  "requires_2fa": true,
  "is_2fa_required": true,
  "user_id": 1,
  "username": "johndoe123"
}
```

#### Expected POST /api/verify-otp
```json
// Request Payload
{
  "email": "johndoe@example.com",
  "otp_code": "123456"
}

// Response (200 OK)
{
  "message": "Account verified successfully"
}
```

## 📊 Test Coverage Checklist

### Registration Flow
- [ ] Valid registration succeeds
- [ ] Invalid name shows error
- [ ] Invalid username shows error
- [ ] Invalid email shows error
- [ ] Weak password shows error
- [ ] Password mismatch shows error
- [ ] Under 13 age shows error
- [ ] Duplicate username/email shows backend error
- [ ] Successful redirect to OTP page
- [ ] OTP verification succeeds
- [ ] Invalid OTP shows error
- [ ] OTP resend works
- [ ] Redirect to login after verification

### Login Flow
- [ ] Valid login succeeds (no 2FA)
- [ ] Invalid credentials show error
- [ ] Unverified account shows prompt
- [ ] 2FA login redirects to OTP
- [ ] 2FA OTP verification succeeds
- [ ] Token saved to localStorage
- [ ] User data saved to localStorage
- [ ] Redirect to feed after login

### State Management
- [ ] Auth store loading state works
- [ ] Error messages display correctly
- [ ] Auth persists after refresh
- [ ] Logout clears state
- [ ] Store getters return correct values

### UI/UX
- [ ] Loading buttons show correct text
- [ ] Error alerts are dismissible
- [ ] Success messages appear
- [ ] Form validation is real-time
- [ ] Links navigate correctly
- [ ] Back buttons work
- [ ] Redirects happen automatically

---

## 🎯 Ready for Phase 2

Once all tests pass, you're ready to proceed with:
- Feed implementation
- Post creation
- User profiles
- Follow/unfollow functionality

---

**Last Updated**: Phase 1 Complete
**Next**: Phase 2 - Feed & Posts
