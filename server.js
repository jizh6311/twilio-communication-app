// Your Account Sid and Auth Token from twilio.com/console
// DANGER! This is insecure. See http://twil.io/secure
const config = require('./local-credentials.json');
const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const port = process.env.PORT || 3000;
const accountSid = config.accountSid;
const authToken = config.authToken;
const client = require('twilio')(accountSid, authToken);

// MiddleWare
app.use(bodyParser.urlencoded());
app.use(bodyParser.json({ extended: false }));

// Make voice call
app.post('/communication/voice', function (req, res) {
  client.calls
        .create({
           method: 'GET',
           sendDigits: '1234#',
           url: 'http://demo.twilio.com/docs/voice.xml',
           to: config.toNumber,
           from: config.fromNumber
         })
        .then(call => res.send({ sid: call.sid }))
        .catch(error => res.status(500).json({ error: error.toString() }));
});

// Send message text
app.post('/communication/message', function (req, res) {
  const message = req.body.message,
    toNumber = req.body.toNumber;
  client.messages
    .create({
       body: message,
       from: config.fromNumber,
       to: toNumber,
     })
    .then(message => res.send({
        message: message,
        sid: message.sid
      }
    ))
    .catch(error => res.status(500).json({ error: error.toString() }));
});

app.listen(port, () => console.log(`Communication app is listening on port ${port}!`));
