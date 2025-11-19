# 🚀 Frontend-Backend Connection Checklist

## ✅ Completed Setup

### Backend Changes
- [x] Added CORS middleware to API Gateway
- [x] Configured to allow all origins (*)
- [x] Handles OPTIONS preflight requests
- [x] All authentication endpoints ready
- [x] Media upload endpoint ready
- [x] JWT authentication working
- [x] Rate limiting configured
- [x] OTP email sending configured

### Frontend Implementation
- [x] Created API service layer (`src/services/api.ts`)
- [x] Implemented Sign Up page with OTP
- [x] Implemented Login page
- [x] Implemented 2FA OTP verification
- [x] Implemented Forgot Password flow
- [x] Implemented Reset Password
- [x] Added navigation guards
- [x] Configured environment variables
- [x] Added error handling
- [x] Cleaned up unused components

## 📋 Testing Steps

### 1. Start Docker Desktop
- Open Docker Desktop application
- Wait for it to fully start

### 2. Start Backend Services
```powershell
cd D:\Codin\TPA\Web\hoshibmatchi
docker-compose -f docker-compose.dev.yml up -d
```

Wait for all services to start (check with `docker ps`).

### 3. Restart API Gateway (to apply CORS)
```powershell
docker-compose -f docker-compose.dev.yml restart api-gateway
```

Verify it's running:
```powershell
docker logs api-gateway --tail 20
```

Should see: "API Gateway listening on port 8000..."

### 4. Start Frontend Development Server
```powershell
cd frontend\hoshi-vue
npm run dev
```

Frontend will be available at: http://localhost:5173

## 🧪 Test Each Flow

### Test 1: Registration Flow

1. **Navigate to Sign Up**
   - Open http://localhost:5173/signup
   - Should see the sign up form

2. **Fill Form**
   - Name: "Test User" (>4 chars, no numbers)
   - Username: "testuser123" (3-30 chars, alphanumeric)
   - Email: "test@example.com"
   - Password: "Test@123" (8+ chars, 1 upper, 1 lower, 1 number, 1 special)
   - Confirm Password: "Test@123"
   - Gender: Select "Male" or "Female"
   - Date of Birth: Select date (must be 13+ years ago)
   - Profile Picture: Upload an image

3. **Request OTP**
   - Click "Request OTP" button
   - Check backend logs: `docker logs email-service --tail 50`
   - Check email (or check logs for OTP code)
   - Button should disable for 60 seconds

4. **Enter OTP**
   - Type the 6-digit code
   - Success message should appear below OTP input

5. **Submit Registration**
   - Click "Sign up" button
   - Should redirect to /login
   - Success message: "Registration successful!"
   - Email should be pre-filled in login form

**Expected Result:** ✅ User created in database, redirected to login

---

### Test 2: Login Flow (Without 2FA)

1. **Navigate to Login**
   - Go to http://localhost:5173/login
   - Should see login form

2. **Enter Credentials**
   - Username: "testuser123" (from registration)
   - Password: "Test@123"

3. **Click Login**
   - Button shows "Logging in..."
   - Should redirect to /feed
   - JWT token stored in localStorage

4. **Verify Authentication**
   - Open browser DevTools > Application > Local Storage
   - Should see `jwt_token` and `user` keys

**Expected Result:** ✅ Logged in, redirected to feed

---

### Test 3: Login Flow (With 2FA)

1. **Register with 2FA Enabled**
   - Follow Test 1 but check "Enable Two-Factor Authentication"

2. **Login**
   - Enter username and password
   - Should redirect to /login-otp

3. **Verify OTP Page**
   - Should show "Two-Factor Authentication" page
   - Should see 6-digit input boxes
   - Resend button disabled for 60 seconds

4. **Check Email for OTP**
   - Check email or logs: `docker logs email-service --tail 50`
   - Enter 6-digit code

5. **Submit OTP**
   - Click "Verify" button
   - Should redirect to /feed
   - JWT token stored

**Expected Result:** ✅ 2FA verification successful, logged in

---

### Test 4: Forgot Password Flow

1. **Navigate to Forgot Password**
   - Go to http://localhost:5173/login
   - Click "Forgot password?" link
   - Should redirect to /forgot-password

2. **Enter Email**
   - Email: "test@example.com"
   - Click "Send Reset Code"

3. **Check for OTP**
   - Check email or logs for 6-digit code
   - Success message should appear
   - OTP input should appear below email

4. **Enter OTP**
   - Type the 6-digit code
   - Click "Verify Code"
   - Should redirect to /reset-password

5. **Enter New Password**
   - New Password: "NewTest@456"
   - Confirm Password: "NewTest@456"
   - Click "Reset Password"

6. **Verify Redirect**
   - Should redirect to /login
   - Success message: "Password reset successful!"

7. **Test New Password**
   - Login with new password: "NewTest@456"
   - Should work

**Expected Result:** ✅ Password changed, can login with new password

---

### Test 5: Google OAuth (Optional)

**Prerequisites:**
1. Get Google OAuth Client ID from [Google Cloud Console](https://console.cloud.google.com/)
2. Add to `.env.development`:
   ```
   VITE_GOOGLE_CLIENT_ID=your-client-id-here
   ```
3. Add redirect URI in Google Console:
   ```
   http://localhost:5173/auth/google/callback
   ```
4. Restart frontend: `npm run dev`

**Test Steps:**
1. Go to /signup or /login
2. Click "Sign up with Google" or "Log in with Google"
3. Should redirect to Google OAuth
4. Select Google account
5. Should redirect back to frontend
6. (Note: Callback handler needs implementation)

**Expected Result:** ⏳ Redirects to Google, callback pending

---

## 🔍 Debugging

### Check Backend API Gateway Logs
```powershell
docker logs api-gateway --tail 50 -f
```

Look for:
- CORS headers being set
- Incoming requests
- Response status codes

### Check Email Service Logs
```powershell
docker logs email-service --tail 50 -f
```

Look for:
- OTP codes being generated
- SMTP connection success
- Email sending confirmation

### Check User Service Logs
```powershell
docker logs user-service --tail 50 -f
```

Look for:
- User registration success
- Login attempts
- JWT token generation

### Check Media Service Logs
```powershell
docker logs media-service --tail 50 -f
```

Look for:
- File upload success
- MinIO connection
- Image optimization

### Check Frontend Console
- Open Browser DevTools (F12)
- Go to Console tab
- Look for:
  - API request/response logs
  - Error messages
  - Network errors

### Check Network Tab
- Open Browser DevTools (F12)
- Go to Network tab
- Filter by "XHR"
- Check:
  - Request URL (should be http://localhost:8000)
  - Request Headers (Authorization header if logged in)
  - Response Status (200 OK, 201 Created, 401 Unauthorized, etc.)
  - Response Data

## ❌ Common Issues

### Issue: CORS Error
**Symptom:** Browser console shows "CORS policy" error

**Solution:**
1. Restart API Gateway: `docker-compose -f docker-compose.dev.yml restart api-gateway`
2. Check API Gateway logs for CORS headers
3. Verify frontend is calling http://localhost:8000

---

### Issue: Network Error / Connection Refused
**Symptom:** "Network Error" or "ERR_CONNECTION_REFUSED"

**Solution:**
1. Check Docker is running: `docker ps`
2. Check API Gateway is up: `curl http://localhost:8000/health`
3. Verify all services are running: `docker ps | findstr Up`

---

### Issue: OTP Not Received
**Symptom:** No email received after clicking "Request OTP"

**Solution:**
1. Check email-service logs: `docker logs email-service --tail 50`
2. Look for OTP code in logs (for testing)
3. Verify SMTP credentials in docker-compose.dev.yml
4. Rate limit: Wait 60 seconds between requests

---

### Issue: Invalid OTP Error
**Symptom:** "Invalid verification code" error

**Solution:**
1. OTP is case-sensitive (should be 6 digits)
2. OTP expires in 5 minutes
3. Check Redis: `docker logs redis --tail 20`
4. Request new OTP if expired

---

### Issue: 401 Unauthorized
**Symptom:** All API calls return 401 after login

**Solution:**
1. Check JWT token in localStorage
2. Verify JWT_SECRET matches in backend
3. Check token expiry (24h default)
4. Clear localStorage and login again

---

### Issue: Profile Picture Upload Fails
**Symptom:** "Failed to upload file" error

**Solution:**
1. Check media-service logs: `docker logs media-service --tail 50`
2. Check MinIO is running: `docker ps | findstr minio`
3. Verify file size (check for limits)
4. Check MinIO credentials in docker-compose.dev.yml

---

## ✅ Success Indicators

### Backend
- [x] All 24 containers running
- [x] API Gateway accessible at http://localhost:8000/health
- [x] CORS headers present in responses
- [x] Email service sending OTPs
- [x] MinIO accepting file uploads

### Frontend
- [x] Dev server running at http://localhost:5173
- [x] All pages load without errors
- [x] API calls go to http://localhost:8000
- [x] JWT token stored after login
- [x] Protected routes redirect to login

### End-to-End
- [ ] Can register new user
- [ ] Can receive OTP via email
- [ ] Can upload profile picture
- [ ] Can login with credentials
- [ ] Can verify 2FA OTP
- [ ] Can reset password
- [ ] JWT token works for authenticated requests

## 📝 Next Steps After Testing

1. **If all tests pass:**
   - Start implementing Feed page
   - Add post creation UI
   - Implement profile page

2. **If tests fail:**
   - Check specific error logs
   - Review debugging section above
   - Verify all environment variables set
   - Ensure Docker Desktop is running

3. **Google OAuth:**
   - Implement callback handler in frontend
   - Create `/auth/google/callback` component
   - Handle id_token from Google
   - Send to backend and store JWT

## 🎯 Ready to Test!

Run these commands in order:

```powershell
# 1. Start Docker Desktop (if not running)

# 2. Start all backend services
cd D:\Codin\TPA\Web\hoshibmatchi
docker-compose -f docker-compose.dev.yml up -d

# 3. Wait for services to start (30-60 seconds)
docker ps

# 4. Restart API Gateway for CORS
docker-compose -f docker-compose.dev.yml restart api-gateway

# 5. Start frontend
cd frontend\hoshi-vue
npm run dev

# 6. Open browser
# Go to http://localhost:5173/signup
```

**Then follow Test 1-5 above!** 🚀
