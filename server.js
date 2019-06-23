// Your Account Sid and Auth Token from twilio.com/console
// DANGER! This is insecure. See http://twil.io/secure
const config = require('./local-credentials.json');
const accountSid = config.accountSid;
const authToken = config.authToken;
const client = require('twilio')(accountSid, authToken);

client.calls
      .create({
         method: 'GET',
         sendDigits: '1234#',
         url: 'http://demo.twilio.com/docs/voice.xml',
         to: config.toNumber,
         from: config.fromNumber
       })
      .then(call => console.log(call.sid));
