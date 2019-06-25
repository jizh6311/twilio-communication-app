# twilio-communication-app
Play with multiple Twilio Communication APIs for SMS, voice and video calls and etc

## Local setup

1. Sign up a free-trial account at https://www.twilio.com/try-twilio and add a phone number
2. Add your credentials and trial numbers to local-credentials.json
```
{
  "accountSid": <accountSid>,
  "authToken": <authToken>,
  "fromNumber": <twilio-number>,
  "toNumber": <your-number>
}
```
3. Run ```yarn start```. The server will make a phone call to your number. Remember you have limited usage. Don't play for fun too long.

## API Document
1. Use POST ```/communication/voice``` to send voice call to your phone.
2. Use POST ```/communication/message``` with payload ```{ message: <message> }``` to send message to your phone.
