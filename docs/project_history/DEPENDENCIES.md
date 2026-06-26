# Dependencies Required

The following Go modules need to be added to go.mod:

```bash
# In the backend directory, run:
go get github.com/sendgrid/sendgrid-go
go get github.com/twilio/twilio-go
go get github.com/pusher/pusher-http-go/v5
```

## External Service Setup Required

### SendGrid (Email)
- Sign up at https://sendgrid.com
- Get API key from Settings → API Keys
- Add to .env: `SENDGRID_API_KEY=your_key`

### Twilio (SMS)
- Sign up at https://twilio.com
- Get Account SID and Auth Token from Console
- Get a Twilio phone number
- Add to .env:
  ```
  TWILIO_ACCOUNT_SID=your_sid
  TWILIO_AUTH_TOKEN=your_token
  TWILIO_FROM_NUMBER=+15551234567
  ```

### Pusher (Real-time)
- Sign up at https://pusher.com
- Create a new app
- Get credentials from App Keys
- Add to .env:
  ```
  PUSHER_APP_ID=your_app_id
  PUSHER_KEY=your_key
  PUSHER_SECRET=your_secret
  PUSHER_CLUSTER=us2
  ```

## Optional: Disable notification services

If you don't want to use these services yet, you can comment out the imports in:
- `backend/internal/services/email_service.go`
- `backend/internal/services/sms_service.go`
- `backend/internal/services/pusher_service.go`

Or simply don't instantiate these services in your application code.
